// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	crypto "crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"math/rand"
	"net/http"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/modern-go/reflect2"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"sigs.k8s.io/yaml"

	ocmlog "github.com/open-component-model/ocm/pkg/logging"
)

// PrintPrettyYaml prints the given objects as yaml if enabled.
func PrintPrettyYaml(obj interface{}, enabled bool) {
	if !enabled {
		return
	}

	data, err := yaml.Marshal(obj)
	if err != nil {
		//nolint: forbidigo // Intentional Println to not mess up potential output parsers.
		fmt.Println(err)
		return
	}

	//nolint: forbidigo // Intentional Println.
	fmt.Println(string(data))
}

// GetFileType returns the mimetype of a file.
func GetFileType(fs vfs.FileSystem, path string) (string, error) {
	file, err := fs.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	// see http://golang.org/pkg/net/http/#DetectContentType for the 512 bytes
	buf := make([]byte, 512)
	_, err = file.Read(buf)
	if err != nil {
		return "", err
	}
	return http.DetectContentType(buf), nil
}

// CleanMarkdownUsageFunc removes Markdown tags from the long usage of the command.
// With this func it is possible to generate the Markdown docs but still have readable commandline help func.
// Note: currently only "<pre>" tags are removed.
func CleanMarkdownUsageFunc(cmd *cobra.Command) {
	defaultHelpFunc := cmd.HelpFunc()
	cmd.SetHelpFunc(func(cmd *cobra.Command, s []string) {
		cmd.Long = strings.ReplaceAll(cmd.Long, "<pre>", "")
		cmd.Long = strings.ReplaceAll(cmd.Long, "</pre>", "")
		defaultHelpFunc(cmd, s)
	})
}

// RawJSON converts an arbitrary value to json.RawMessage.
func RawJSON(value interface{}) (*json.RawMessage, error) {
	jsonval, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return (*json.RawMessage)(&jsonval), nil
}

// Gzip applies gzip compression to an arbitrary byte slice.
func Gzip(data []byte, compressionLevel int) ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})
	gzipWriter, err := gzip.NewWriterLevel(buf, compressionLevel)
	if err != nil {
		return nil, fmt.Errorf("unable to create gzip writer: %w", err)
	}
	defer gzipWriter.Close()

	if _, err = gzipWriter.Write(data); err != nil {
		return nil, fmt.Errorf("unable to write to stream: %w", err)
	}

	if err = gzipWriter.Close(); err != nil {
		return nil, fmt.Errorf("unable to close writer: %w", err)
	}

	return buf.Bytes(), nil
}

var chars = []rune("abcdefghijklmnopqrstuvwxyz1234567890")

// RandomString creates a new random string with the given length.
func RandomString(n int) string {
	b := make([]rune, n)
	for i := range b {
		var value int
		if v, err := crypto.Int(crypto.Reader, big.NewInt(int64(len(chars)))); err == nil {
			value = int(v.Int64())
		} else {
			// insecure fallback to provide a valid result
			ocmlog.Logger().Error("failed to generate random number", "error", err.Error())
			value = rand.Intn(len(chars)) //nolint: gosec // only used as fallback
		}
		b[i] = chars[value]
	}
	return string(b)
}

// SafeConvert converts a byte slice to string.
// If the byte slice is nil, an empty string is returned.
func SafeConvert(bytes []byte) string {
	if bytes == nil {
		return ""
	}

	return string(bytes)
}

const (
	BYTE = 1.0 << (10 * iota)
	KIBIBYTE
	MEBIBYTE
	GIBIBYTE
)

// BytesString converts bytes into a human-readable string.
// This function is inspired by https://www.reddit.com/r/golang/comments/8micn7/review_bytes_to_human_readable_format/
func BytesString(bytes uint64, accuracy int) string {
	unit := ""
	value := float32(bytes)

	switch {
	case bytes >= GIBIBYTE:
		unit = "GiB"
		value /= GIBIBYTE
	case bytes >= MEBIBYTE:
		unit = "MiB"
		value /= MEBIBYTE
	case bytes >= KIBIBYTE:
		unit = "KiB"
		value /= KIBIBYTE
	case bytes >= BYTE:
		unit = "bytes"
	case bytes == 0:
		return "0"
	}

	stringValue := strings.TrimSuffix(
		fmt.Sprintf("%.2f", value), "."+strings.Repeat("0", accuracy),
	)

	return fmt.Sprintf("%s %s", stringValue, unit)
}

// WriteFileToTARArchive writes a new file with name=filename and content=contentReader to archiveWriter.
func WriteFileToTARArchive(filename string, contentReader io.Reader, archiveWriter *tar.Writer) error {
	if filename == "" {
		return errors.New("filename must not be empty")
	}

	if contentReader == nil {
		return errors.New("contentReader must not be nil")
	}

	if archiveWriter == nil {
		return errors.New("archiveWriter must not be nil")
	}

	tempfile, err := os.CreateTemp("", "")
	if err != nil {
		return fmt.Errorf("unable to create tempfile: %w", err)
	}
	defer func() {
		tempfile.Close()
		os.Remove(tempfile.Name())
	}()

	fsize, err := io.Copy(tempfile, contentReader)
	if err != nil {
		return fmt.Errorf("unable to copy content to tempfile: %w", err)
	}

	if _, err := tempfile.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("unable to seek to beginning of tempfile: %w", err)
	}

	header := tar.Header{
		Name:    filename,
		Size:    fsize,
		Mode:    0o600,
		ModTime: time.Now(),
	}

	if err := archiveWriter.WriteHeader(&header); err != nil {
		return fmt.Errorf("unable to write tar header: %w", err)
	}

	if _, err := io.Copy(archiveWriter, tempfile); err != nil {
		return fmt.Errorf("unable to write file to tar archive: %w", err)
	}

	return nil
}

func IndentLines(orig string, gap string, skipfirst ...bool) string {
	return JoinIndentLines(strings.Split(strings.TrimPrefix(orig, "\n"), "\n"), gap, skipfirst...)
}

func JoinIndentLines(orig []string, gap string, skipfirst ...bool) string {
	if len(orig) == 0 {
		return ""
	}
	skip := false
	for _, b := range skipfirst {
		skip = skip || b
	}

	s := ""
	if !skip {
		s = gap
	}
	return s + strings.Join(orig, "\n"+gap)
}

func StringMapKeys[K ~string, E any](m map[K]E) []K {
	if m == nil {
		return nil
	}
	keys := maps.Keys(m)
	slices.Sort(keys)
	return keys
}

type Comparable[K any] interface {
	Compare(o K) int
}

func Sort[K Comparable[K]](a []K) {
	sort.Slice(a, func(i, j int) bool { return a[i].Compare(a[j]) < 0 })
}

func MapKeys[K comparable, E any](m map[K]E) []K {
	if m == nil {
		return nil
	}

	keys := []K{}
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

type ComparableMapKey[K any] interface {
	Comparable[K]
	comparable
}

func SortedMapKeys[K ComparableMapKey[K], E any](m map[K]E) []K {
	if m == nil {
		return nil
	}

	keys := []K{}
	for k := range m {
		keys = append(keys, k)
	}
	Sort(keys)
	return keys
}

// Optional returns the first optional non-zero element given as variadic argument,
// if given, or the zero element as default.
func Optional[T any](list ...T) T {
	var zero T
	for _, e := range list {
		if !reflect.DeepEqual(e, zero) {
			return e
		}
	}
	return zero
}

// OptionalDefaulted returns the first optional non-nil element given as variadic
// argument, or the given default element. For value types a given zero
// argument is excepted, also.
func OptionalDefaulted[T any](def T, list ...T) T {
	for _, e := range list {
		if !reflect2.IsNil(e) {
			return e
		}
	}
	return def
}

// OptionalDefaultedBool checks all args for true. If arg is given
// the given default is returned.
func OptionalDefaultedBool(def bool, list ...bool) bool {
	if len(list) == 0 {
		return def
	}
	for _, e := range list {
		if e {
			return e
		}
	}
	return false
}

// GetOptionFlag returns the flag value used to set a bool option
// based on optionally specified explicit value(s).
// The default value is to enable the option (true).
func GetOptionFlag(list ...bool) bool {
	return OptionalDefaultedBool(len(list) == 0, list...)
}

// Must expect a result to be provided without error.
func Must[T any](o T, err error) T {
	if err != nil {
		panic(fmt.Errorf("expected a %T, but got error %w", o, err))
	}
	return o
}

func IgnoreError(_ error) {
}

func BoolP[T ~bool](b T) *bool {
	v := bool(b)
	return &v
}

func AsBool(b *bool, def ...bool) bool {
	if b == nil && len(def) > 0 {
		return Optional(def...)
	}
	return b != nil && *b
}
