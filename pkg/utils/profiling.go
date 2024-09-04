package utils

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"

	lserrors "github.com/gardener/landscaper/apis/errors"

	"github.com/cloudflare/cfssl/log"
	"github.com/shirou/gopsutil/v4/process"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
)

const separator = " ***** "
const suffix = "-heap"
const keyNumberOfDataSecrets = "keyNumberOfDataSecrets"
const keyHeapInUse = "keyHeapInUse"
const keyHeapAlloc = "keyHeapAlloc"
const keyBytes = "keyBytes"
const keyPodname = "keyPodname"
const keyStorageDate = "keyStorageDate"
const keyLockTime = "keyLockTime"

// labelKeyProfilingPrefix is a label of secrets that store a chunk of a heap dump.
// The value is the prefix of the controller.
const labelProfilingPrefix = v1alpha1.LandscaperDomain + "/profiling-prefix"

var maxHeapInUse uint64 = 1000 * 1000 * 300

func LogMemStatsPeriodically(ctx context.Context, interval time.Duration, hostUncachedClient client.Client, prefix string) {
	log, ctx := logging.FromContextOrNew(ctx, nil)

	log.Info("Starting LogMemStats loop")
	for {
		if err := ctx.Err(); err != nil {
			log.Info("LogMemStats loop was cancelled: %w", err)
			return
		}

		LogMemStats(ctx, hostUncachedClient, prefix)
		time.Sleep(interval)
	}
}

func LogMemStats(ctx context.Context, hostUncachedClient client.Client, prefix string) {
	w := &strings.Builder{}
	writeMemStats(ctx, w, hostUncachedClient, prefix)
	writeProcessMemoryInfo(w)
	log.Info(w.String())
}

func writeMemStats(ctx context.Context, w *strings.Builder, hostUncachedClient client.Client, prefix string) {
	// See: https://golang.org/pkg/runtime/#MemStats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	storeHeap(ctx, &m, hostUncachedClient, prefix)

	w.WriteString("MemStats: ")
	fmt.Fprintf(w, "Alloc: %v MiB, ", bToMiB(m.Alloc))
	fmt.Fprintf(w, "TotalAlloc: %v MiB, ", bToMiB(m.TotalAlloc))
	fmt.Fprintf(w, "Sys: %v MiB, ", bToMiB(m.Sys))
	fmt.Fprintf(w, "Lookups: %v, ", m.Lookups)
	fmt.Fprintf(w, "Mallocs: %v, ", m.Mallocs)
	fmt.Fprintf(w, "Frees: %v, ", m.Frees)
	fmt.Fprintf(w, "HeapAlloc: %v MiB, ", bToMiB(m.HeapAlloc))
	fmt.Fprintf(w, "HeapSys: %v MiB, ", bToMiB(m.HeapSys))
	fmt.Fprintf(w, "HeapIdle: %v MiB, ", bToMiB(m.HeapIdle))
	fmt.Fprintf(w, "HeapInuse: %v MiB, ", bToMiB(m.HeapInuse))
	fmt.Fprintf(w, "HeapReleased: %v MiB, ", bToMiB(m.HeapReleased))
	fmt.Fprintf(w, "HeapObjects: %v, ", m.HeapObjects)
	fmt.Fprintf(w, "StackInuse: %v MiB, ", bToMiB(m.StackInuse))
	fmt.Fprintf(w, "StackSys: %v MiB, ", bToMiB(m.StackSys))
	fmt.Fprintf(w, "MSpanInuse: %v MiB, ", bToMiB(m.MSpanInuse))
	fmt.Fprintf(w, "MSpanSys: %v MiB, ", bToMiB(m.MSpanSys))
	fmt.Fprintf(w, "MCacheInuse: %v MiB, ", bToMiB(m.MCacheInuse))
	fmt.Fprintf(w, "MCacheSys: %v MiB, ", bToMiB(m.MCacheSys))
	fmt.Fprintf(w, "BuckHashSys: %v MiB, ", bToMiB(m.BuckHashSys))
	fmt.Fprintf(w, "GCSys: %v MiB, ", bToMiB(m.GCSys))
	fmt.Fprintf(w, "OtherSys: %v MiB, ", bToMiB(m.OtherSys))
	fmt.Fprintf(w, "NextGC: %v MiB, ", bToMiB(m.NextGC))
	fmt.Fprintf(w, "LastGC: %v, ", time.Unix(0, int64(m.LastGC)).Format(time.RFC3339))
	fmt.Fprintf(w, "PauseTotalNs: %v ns, ", m.PauseTotalNs)
	fmt.Fprintf(w, "NumGC: %v, ", m.NumGC)
	fmt.Fprintf(w, "NumForcedGC: %v, ", m.NumForcedGC)
	fmt.Fprintf(w, "GCCPUFraction: %v, ", m.GCCPUFraction)
	fmt.Fprintf(w, "EnableGC: %v, ", m.EnableGC)
	fmt.Fprintf(w, "DebugGC: %v, ", m.DebugGC)
}

func writeProcessMemoryInfo(w *strings.Builder) {
	p, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		w.WriteString(separator)
		fmt.Fprintf(w, "Process memory info unavailable: %v", err)
		return
	}

	writeGenericProcessMemoryInfo(w, p)
	writePlatformSpecificProcessMemoryInfo(w, p)
}

func writeGenericProcessMemoryInfo(w *strings.Builder, p *process.Process) {
	w.WriteString(separator)

	memInfo, err := p.MemoryInfo()
	if err != nil {
		fmt.Fprintf(w, "Generic process memory info unavailable: %v", err)
		return
	}

	w.WriteString("Generic process memory info: ")
	fmt.Fprintf(w, "RSS: %v MiB, ", bToMiB(memInfo.RSS))
	fmt.Fprintf(w, "VMS: %v MiB, ", bToMiB(memInfo.VMS))
	fmt.Fprintf(w, "HWM: %v MiB, ", bToMiB(memInfo.HWM))
	fmt.Fprintf(w, "Data: %v MiB, ", bToMiB(memInfo.Data))
	fmt.Fprintf(w, "Stack: %v MiB, ", bToMiB(memInfo.Stack))
	fmt.Fprintf(w, "Locked: %v MiB, ", bToMiB(memInfo.Locked))
	fmt.Fprintf(w, "Swap: %v MiB ", bToMiB(memInfo.Swap))
}

func writePlatformSpecificProcessMemoryInfo(w *strings.Builder, p *process.Process) {
	w.WriteString(separator)

	memInfoEx, err := p.MemoryInfoEx()
	if err != nil {
		fmt.Fprintf(w, "Platform specific process memory info unavailable: %v", err)
		return
	}

	fmt.Fprintf(w, "Platform specific process memory info: %+v", *memInfoEx)
}

func bToMiB(numOfBytes uint64) uint64 {
	return numOfBytes / (1024 * 1024)
}

func storeHeap(ctx context.Context, m *runtime.MemStats, hostUncachedClient client.Client, prefix string) {
	log, ctx := logging.FromContextOrNew(ctx, nil)

	if maxHeapInUse+maxHeapInUse/10 < m.HeapInuse {
		log.Info("storeHeapProfile check with", "maxHeapInUse+maxHeapInUse/10", maxHeapInUse+maxHeapInUse/10, "m.HeapInuse", m.HeapInuse)

		var buf bytes.Buffer
		if err := pprof.WriteHeapProfile(&buf); err != nil {
			log.Error(err, "Failed to get heap profile with HeapInuse "+strconv.FormatUint(m.HeapInuse, 10)+" bytes")
			return
		}

		if err := storeHeapProfile(ctx, &buf, hostUncachedClient, prefix, m); err != nil {
			log.Error(err, "storeHeapProfile failed to write heap profile with HeapInuse "+strconv.FormatUint(m.HeapInuse, 10)+" bytes")
			return
		}

		log.Info("storeHeapProfile succeeded")

		maxHeapInUse = m.HeapInuse
	}
}

func storeHeapProfile(ctx context.Context, buf *bytes.Buffer, hostUncachedClient client.Client, prefix string, m *runtime.MemStats) error {
	data := buf.Bytes()

	const chunkSize = 500 * 1024

	// Split the byte array into chunks
	var chunks [][]byte
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize

		if end > len(data) {
			end = len(data)
		}

		chunks = append(chunks, data[i:end])
	}

	// create base secret or take over base secret and lock it
	heapSecret, err := fetchOrCreateSecret(ctx, hostUncachedClient, prefix)
	if err != nil {
		return err
	}

	// cleanup old data secrets
	if err = cleanupOldDataSecrets(ctx, hostUncachedClient, prefix); err != nil {
		return err
	}

	// update base secret
	if err = updateSecret(ctx, hostUncachedClient, heapSecret, m, len(chunks)); err != nil {
		return err
	}

	// create data secrets
	if err = createDataSecrets(ctx, hostUncachedClient, prefix, chunks); err != nil {
		return err
	}

	// unlock base secret
	delete(heapSecret.Data, keyLockTime)
	return hostUncachedClient.Update(ctx, heapSecret)
}

func createDataSecrets(ctx context.Context, hostUncachedClient client.Client, prefix string, chunks [][]byte) error {
	for i := 0; i < len(chunks); i++ {
		nextDataSecret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      getHeapSecretDataName(prefix, i),
				Namespace: GetCurrentPodNamespace(),
				Labels: map[string]string{
					labelProfilingPrefix: prefix,
				},
			},
			Data: map[string][]byte{
				keyBytes: chunks[i],
			},
		}

		if err := hostUncachedClient.Create(ctx, nextDataSecret); err != nil {
			return err
		}
	}

	return nil
}

func updateSecret(ctx context.Context, hostUncachedClient client.Client, heapSecret *v1.Secret, m *runtime.MemStats, numOfChunks int) error {
	heapSecret.Data = map[string][]byte{
		keyNumberOfDataSecrets: []byte(strconv.Itoa(numOfChunks)),
		keyHeapInUse:           []byte(strconv.FormatUint(m.HeapInuse, 10)),
		keyHeapAlloc:           []byte(strconv.FormatUint(m.HeapAlloc, 10)),
		keyPodname:             []byte(GetCurrentPodName()),
		keyStorageDate:         []byte(time.Now().String()),
		keyLockTime:            heapSecret.Data[keyLockTime],
	}

	if err := hostUncachedClient.Update(ctx, heapSecret); err != nil {
		return err
	}

	return nil
}

func cleanupOldDataSecrets(ctx context.Context, hostUncachedClient client.Client, prefix string) error {
	secretList := &v1.SecretList{}
	err := read_write_layer.ListSecrets(ctx, hostUncachedClient, secretList, read_write_layer.R000007,
		client.InNamespace(GetCurrentPodNamespace()),
		client.MatchingLabels{labelProfilingPrefix: prefix})
	if err != nil {
		return err
	}

	for i := range secretList.Items {
		if err = hostUncachedClient.Delete(ctx, &secretList.Items[i]); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
		}
	}

	return nil
}

func fetchOrCreateSecret(ctx context.Context, hostUncachedClient client.Client, prefix string) (*v1.Secret, error) {
	heapSecret := &v1.Secret{}
	key := client.ObjectKey{Namespace: GetCurrentPodNamespace(), Name: getHeapSecretName(prefix)}
	if err := hostUncachedClient.Get(ctx, key, heapSecret); err != nil {
		if !errors.IsNotFound(err) {
			return nil, err
		}

		heapSecret = &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      getHeapSecretName(prefix),
				Namespace: GetCurrentPodNamespace(),
			},

			Data: map[string][]byte{
				keyNumberOfDataSecrets: []byte("0"),
				keyHeapInUse:           []byte("0"),
				keyHeapAlloc:           []byte("0"),
				keyPodname:             []byte(GetCurrentPodName()),
				keyStorageDate:         []byte(time.Now().String()),
				keyLockTime:            []byte(time.Now().Format(time.RFC3339)),
			},
		}

		if err := hostUncachedClient.Create(ctx, heapSecret); err != nil {
			return nil, err
		}

		// return new secret
		return heapSecret, nil
	}

	lockTimeBytes, contained := heapSecret.Data[keyLockTime]
	if isEmptyOrOutdated(ctx, contained, lockTimeBytes) {
		heapSecret.Data[keyLockTime] = []byte(time.Now().Format(time.RFC3339))
		if err := hostUncachedClient.Update(ctx, heapSecret); err != nil {
			return nil, err
		}
		return heapSecret, nil
	}

	return nil, lserrors.NewError("fetchOrCreateSecret", "HeapSecretLocked", "heap secret locked")
}

func isEmptyOrOutdated(ctx context.Context, contained bool, lockTimeBytes []byte) bool {
	log, _ := logging.FromContextOrNew(ctx, nil)

	if !contained {
		return true
	}

	lockTime, err := time.Parse(time.RFC3339, string(lockTimeBytes))
	if err != nil {
		log.Error(err, "Failed to parse lock time"+string(lockTimeBytes))
		return false
	}

	if lockTime.Add(time.Hour).Before(time.Now()) {
		return true
	}

	return false
}

func getHeapSecretName(prefix string) string {
	return prefix + suffix
}

func getHeapSecretDataName(prefix string, i int) string {
	return getHeapSecretName(prefix) + "-" + strconv.Itoa(i)
}
