// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/opencontainers/go-digest"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/gardener/landscaper/legacy-component-cli/ociclient/metrics"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cache Test Suite")
}

var _ = Describe("Cache", func() {

	Context("Cache", func() {
		It("should read data from the in memory cache", func() {
			c, err := NewCache(logr.Discard(), WithInMemoryOverlay(true))
			Expect(err).ToNot(HaveOccurred())
			defer c.Close()

			desc, data := exampleDataSet(10)
			Expect(c.Add(desc, data)).To(Succeed())

			r, err := c.Get(desc)
			Expect(err).ToNot(HaveOccurred())
			buf := readIntoBuffer(r)
			Expect(buf.Len() > 0).To(BeTrue(), "The cache should return some data")
			r, err = c.Get(desc)
			Expect(err).ToNot(HaveOccurred())
			buf = readIntoBuffer(r)
			Expect(buf.Len() > 0).To(BeTrue(), "The cache should return some data")
		})

		It("should detect tampered data and remove the tempered blob", func() {
			path, err := os.MkdirTemp(os.TempDir(), "ocicache")
			Expect(err).ToNot(HaveOccurred())

			c, err := NewCache(logr.Discard(), WithBasePath(path))
			Expect(err).ToNot(HaveOccurred())
			defer c.Close()

			desc, data := exampleDataSet(10)
			Expect(c.Add(desc, data)).To(Succeed())

			// temper data
			Expect(os.WriteFile(filepath.Join(path, Path(desc)), exampleData(10).Bytes(), os.ModePerm)).To(Succeed())

			_, err = c.Get(desc)
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(ErrNotFound))
		})

		It("should detect tampered data (size shortcut) and remove the tempered blob", func() {
			path, err := os.MkdirTemp(os.TempDir(), "ocicache")
			Expect(err).ToNot(HaveOccurred())

			c, err := NewCache(logr.Discard(), WithBasePath(path))
			Expect(err).ToNot(HaveOccurred())
			defer c.Close()

			desc, data := exampleDataSet(5)
			Expect(c.Add(desc, data)).To(Succeed())

			// temper data
			Expect(os.WriteFile(filepath.Join(path, Path(desc)), exampleData(10).Bytes(), os.ModePerm)).To(Succeed())

			_, err = c.Get(desc)
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(ErrNotFound))
		})

		Context("metrics", func() {
			It("should read data from the in memory cache", func() {
				uid := "unit-test"
				dir, err := os.MkdirTemp(os.TempDir(), "test-")
				Expect(err).ToNot(HaveOccurred())
				defer func() {
					Expect(os.RemoveAll(dir)).To(Succeed())
				}()
				metrics.CachedItems.Reset()
				metrics.CacheHitsDisk.Reset()
				metrics.CacheHitsMemory.Reset()
				metrics.CacheDiskUsage.Reset()
				metrics.CacheMemoryUsage.Reset()

				c, err := NewCache(logr.Discard(), WithBasePath(dir), WithInMemoryOverlay(true), WithUID(uid))
				Expect(err).ToNot(HaveOccurred())
				defer c.Close()

				desc, data := exampleDataSet(10)
				Expect(c.Add(desc, data)).To(Succeed())

				expected := `
				# HELP ociclient_cache_items_total Total number of items currently cached by instance.
                # TYPE ociclient_cache_items_total gauge
                ociclient_cache_items_total{id="%s"} 1
				`
				Expect(testutil.CollectAndCompare(metrics.CachedItems, strings.NewReader(fmt.Sprintf(expected, uid)))).To(Succeed())
				expected = `
				# HELP ociclient_cache_disk_hits_total Total number of hits for items cached on disk by an instance.
                # TYPE ociclient_cache_disk_hits_total counter
                ociclient_cache_disk_hits_total{id="%s"} 0
				`
				Expect(testutil.CollectAndCompare(metrics.CacheHitsDisk, strings.NewReader(fmt.Sprintf(expected, uid)))).To(Succeed())
				expected = `
				# HELP ociclient_cache_memory_hits_total Total number of hits for items cached in memory by an instance.
                # TYPE ociclient_cache_memory_hits_total counter
                ociclient_cache_memory_hits_total{id="%s"} 0
				`
				Expect(testutil.CollectAndCompare(metrics.CacheHitsMemory, strings.NewReader(fmt.Sprintf(expected, uid)))).To(Succeed())
				expected = `
				# HELP ociclient_cache_memory_usage_bytes Bytes in memory currently used by cache instance.
                # TYPE ociclient_cache_memory_usage_bytes gauge
                ociclient_cache_memory_usage_bytes{id="%s"} 0
				`
				Expect(testutil.CollectAndCompare(metrics.CacheMemoryUsage, strings.NewReader(fmt.Sprintf(expected, uid)))).To(Succeed())

				r, err := c.Get(desc)
				Expect(err).ToNot(HaveOccurred())
				buf := readIntoBuffer(r)
				Expect(buf.Len() > 0).To(BeTrue(), "The cache should return some data")
				r, err = c.Get(desc)
				Expect(err).ToNot(HaveOccurred())
				buf = readIntoBuffer(r)
				Expect(buf.Len() > 0).To(BeTrue(), "The cache should return some data")

				expected = `
				# HELP ociclient_cache_items_total Total number of items currently cached by instance.
                # TYPE ociclient_cache_items_total gauge
                ociclient_cache_items_total{id="%s"} 1
				`
				Expect(testutil.CollectAndCompare(metrics.CachedItems, strings.NewReader(fmt.Sprintf(expected, uid)))).To(Succeed())
				expected = `
				# HELP ociclient_cache_disk_usage_bytes Bytes on disk currently used by cache instance.
                # TYPE ociclient_cache_disk_usage_bytes gauge
                ociclient_cache_disk_usage_bytes{id="%s"} 10
				`
				Expect(testutil.CollectAndCompare(metrics.CacheDiskUsage, strings.NewReader(fmt.Sprintf(expected, uid)))).To(Succeed())
				expected = `
				# HELP ociclient_cache_disk_hits_total Total number of hits for items cached on disk by an instance.
                # TYPE ociclient_cache_disk_hits_total counter
                ociclient_cache_disk_hits_total{id="%s"} 1
				`
				Expect(testutil.CollectAndCompare(metrics.CacheHitsDisk, strings.NewReader(fmt.Sprintf(expected, uid)))).To(Succeed())
				expected = `
				# HELP ociclient_cache_memory_hits_total Total number of hits for items cached in memory by an instance.
                # TYPE ociclient_cache_memory_hits_total counter
                ociclient_cache_memory_hits_total{id="%s"} 1
				`
				Expect(testutil.CollectAndCompare(metrics.CacheHitsMemory, strings.NewReader(fmt.Sprintf(expected, uid)))).To(Succeed())
				expected = `
				# HELP ociclient_cache_memory_usage_bytes Bytes in memory currently used by cache instance.
                # TYPE ociclient_cache_memory_usage_bytes gauge
                ociclient_cache_memory_usage_bytes{id="%s"} 10
				`
				Expect(testutil.CollectAndCompare(metrics.CacheMemoryUsage, strings.NewReader(fmt.Sprintf(expected, uid)))).To(Succeed())
			})
		})
	})

	Context("GC", func() {
		It("should garbage collect when the cache reaches its max size", func() {
			c, err := NewCache(logr.Discard(), WithBaseSize("1Ki"))
			Expect(err).ToNot(HaveOccurred())
			defer c.Close()

			Expect(c.Add(exampleDataSet(500))).To(Succeed())
			Expect(c.baseFs.index.entries).To(HaveLen(1))
			Expect(c.Add(exampleDataSet(500))).To(Succeed())
			Eventually(func() map[string]IndexEntry {
				return c.baseFs.index.entries
			}).Should(HaveLen(1))
		})

		It("should garbage collect the in memory cache but not the base cache if the size exceeds", func() {
			c, err := NewCache(logr.Discard(), WithInMemoryOverlay(true), WithInMemoryOverlaySize("1Ki"))
			Expect(err).ToNot(HaveOccurred())
			defer c.Close()

			desc1, buf := exampleDataSet(500)
			Expect(c.Add(desc1, buf)).To(Succeed())
			desc2, buf := exampleDataSet(500)
			Expect(c.Add(desc2, buf)).To(Succeed())
			// let both files be added to the in memory cache by reading them
			r, err := c.Get(desc1)
			Expect(err).ToNot(HaveOccurred())
			Expect(r.Close()).To(Succeed())
			r, err = c.Get(desc2)
			Expect(err).ToNot(HaveOccurred())
			Expect(r.Close()).To(Succeed())

			Eventually(func() map[string]IndexEntry {
				return c.baseFs.index.entries
			}).Should(HaveLen(2))
			Eventually(func() map[string]IndexEntry {
				return c.overlayFs.index.entries
			}).Should(HaveLen(1))
		})

		It("should delete files until the low threshold has been reached", func() {
			c, err := NewCache(logr.Discard(), WithBaseSize("1Ki"))
			Expect(err).ToNot(HaveOccurred())
			defer c.Close()

			desc1, buf := exampleDataSet(500)
			Expect(c.Add(desc1, buf)).To(Succeed())
			r, err := c.Get(desc1)
			Expect(err).ToNot(HaveOccurred())
			Expect(r.Close()).To(Succeed())

			for i := 0; i < 100; i++ {
				Expect(c.Add(exampleDataSet(100 + i))).To(Succeed())
			}

			Eventually(c.baseFs.CurrentSize).Should(BeNumerically("<", 1024))
		})
	})

	Context("Index", func() {

		It("should add 2 entries to the index", func() {
			index := NewIndex()
			index.Add("a", 500, time.Now())
			index.Add("b", 500, time.Now())

			list := index.PriorityList()
			Expect(list).To(HaveLen(2))
		})

		It("should return a prioritised entries based on hits", func() {
			index := NewIndex()
			index.Add("a", 500, newDate("0:01AM"))
			index.Add("b", 500, newDate("0:01AM"))
			index.Hit("a")
			list := index.PriorityList()
			Expect(list).To(HaveLen(2))

			Expect(list[0].Name).To(Equal("b"))
			Expect(list[1].Name).To(Equal("a"))

			index = NewIndex()
			index.Add("a", 500, newDate("0:01AM"))
			index.Add("b", 500, newDate("0:01AM"))
			index.Hit("b")
			list = index.PriorityList()
			Expect(list).To(HaveLen(2))

			Expect(list[0].Name).To(Equal("a"))
			Expect(list[1].Name).To(Equal("b"))
		})

		It("should return a prioritised entries based on added date", func() {
			index := NewIndex()
			index.Add("b", 500, newDate("0:01AM"))
			index.Add("a", 500, newDate("0:03AM"))
			list := index.PriorityList()
			Expect(list).To(HaveLen(2))

			Expect(list[0].Name).To(Equal("b"))
			Expect(list[1].Name).To(Equal("a"))

			index = NewIndex()
			index.Add("a", 500, newDate("0:01AM"))
			index.Add("b", 500, newDate("0:03AM"))
			list = index.PriorityList()
			Expect(list).To(HaveLen(2))

			Expect(list[0].Name).To(Equal("a"))
			Expect(list[1].Name).To(Equal("b"))
		})

		It("should return a prioritised entries based on hits and added date", func() {
			index := NewIndex()
			index.Add("a", 500, newDate("0:01AM"))
			index.Add("b", 500, newDate("0:02AM"))
			index.Add("c", 500, newDate("0:03AM"))
			index.Add("d", 500, newDate("0:04AM"))
			index.Hit("b")
			index.Hit("c")
			index.Hit("c")
			list := index.PriorityList()
			Expect(list).To(HaveLen(4))

			Expect(list[0].Name).To(Equal("a"))
			Expect(list[1].Name).To(Equal("d"))
			Expect(list[2].Name).To(Equal("b"))
			Expect(list[3].Name).To(Equal("c"))
		})

		Context("Hits", func() {
			It("should add hits to a entry", func() {
				index := NewIndex()
				index.Add("a", 500, newDate("0:01AM"))
				index.Hit("a")
				list := index.PriorityList()
				Expect(list).To(HaveLen(1))
				Expect(list[0].Hits).To(Equal(int64(1)))

				index.Hit("a")
				list = index.PriorityList()
				Expect(list).To(HaveLen(1))
				Expect(list[0].Hits).To(Equal(int64(2)))
			})

			It("should reset hits and keep 100% of newly added hits", func() {
				index := NewIndex()
				index.Add("a", 500, newDate("0:01AM"))

				index.Hit("a")
				index.Hit("a")
				index.Hit("a")
				index.Hit("a")
				index.Reset()
				list := index.PriorityList()
				Expect(list).To(HaveLen(1))
				Expect(list[0].Hits).To(Equal(int64(4)))
			})

			It("should reset hits and keep 100% of newly added hits and preserve 50% of old hits", func() {
				index := NewIndex()
				index.Add("a", 500, newDate("0:01AM"))

				index.Hit("a")
				index.Hit("a")
				index.Reset()
				index.Hit("a")
				index.Hit("a")
				index.Reset()
				list := index.PriorityList()
				Expect(list).To(HaveLen(1))
				Expect(list[0].Hits).To(Equal(int64(3)))
			})
		})

		Context("CalculatePriority", func() {
			It("should prioritise more hits if added at the same time", func() {
				var minHits, maxHits int64 = 2, 6
				oldest := newDate("0:01AM")
				newest := newDate("11:59AM")

				entryA := IndexEntry{
					Hits:      3,
					CreatedAt: newDate("0:04AM"),
				}
				valA := CalculatePriority(entryA, minHits, maxHits, oldest, newest)

				entryB := IndexEntry{
					Hits:      4,
					CreatedAt: newDate("0:04AM"),
				}
				valB := CalculatePriority(entryB, minHits, maxHits, oldest, newest)

				Expect(valA).To(BeNumerically("<", valB))
			})

			It("should prioritise the creation date if the hits are the same", func() {
				var minHits, maxHits int64 = 2, 6
				oldest := newDate("0:01AM")
				newest := newDate("11:59AM")

				entryA := IndexEntry{
					Hits:      4,
					CreatedAt: newDate("0:04AM"),
				}
				valA := CalculatePriority(entryA, minHits, maxHits, oldest, newest)

				entryB := IndexEntry{
					Hits:      4,
					CreatedAt: newDate("0:03AM"),
				}
				valB := CalculatePriority(entryB, minHits, maxHits, oldest, newest)

				Expect(valA).To(BeNumerically(">", valB))
			})

			It("should prioritise the hits over the creation date", func() {
				var minHits, maxHits int64 = 2, 6
				oldest := newDate("0:01AM")
				newest := newDate("11:59AM")

				entryA := IndexEntry{
					Hits:      3,
					CreatedAt: newDate("0:04AM"),
				}
				valA := CalculatePriority(entryA, minHits, maxHits, oldest, newest)

				entryB := IndexEntry{
					Hits:      4,
					CreatedAt: newDate("0:03AM"),
				}
				valB := CalculatePriority(entryB, minHits, maxHits, oldest, newest)

				Expect(valA).To(BeNumerically("<", valB))
			})
		})

	})

})

func readIntoBuffer(r io.ReadCloser) *bytes.Buffer {
	var data bytes.Buffer
	_, err := io.Copy(&data, r)
	Expect(err).ToNot(HaveOccurred())
	Expect(r.Close()).To(Succeed())
	return &data
}

func exampleDataSet(size int) (ocispecv1.Descriptor, io.ReadCloser) {
	buf := exampleData(size)
	desc := exampleDesc(buf)
	return desc, io.NopCloser(buf)
}

func exampleDesc(buf *bytes.Buffer) ocispecv1.Descriptor {
	return ocispecv1.Descriptor{
		MediaType: "application/octet-stream",
		Size:      int64(buf.Len()),
		Digest:    digest.FromBytes(buf.Bytes()),
	}
}

func exampleData(size int) *bytes.Buffer {
	data := make([]byte, size)
	_, err := rand.Read(data)
	Expect(err).ToNot(HaveOccurred())
	return bytes.NewBuffer(data)
}

func newDate(val string) time.Time {
	t, err := time.Parse(time.Kitchen, val)
	Expect(err).ToNot(HaveOccurred())
	return t
}
