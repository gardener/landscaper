package utils

import (
	"bytes"
	"context"
	"fmt"
	"github.com/cloudflare/cfssl/log"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
	"os"
	"runtime"
	"runtime/pprof"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"strings"
	"time"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"

	"github.com/shirou/gopsutil/v4/process"
)

const separator = " ***** "

var maxHeapInUse uint64 = 1000 * 1000 * 1000

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

	if maxHeapInUse < m.HeapInuse {
		var buf bytes.Buffer
		if err := pprof.WriteHeapProfile(&buf); err != nil {
			log.Error(err, "Failed to get heap profile with HeapInuse "+strconv.FormatUint(m.HeapInuse, 10)+" bytes")
			return
		}

		if err := storeHeapProfile(ctx, &buf, hostUncachedClient, prefix); err != nil {
			log.Error(err, "Failed to write heap profile with HeapInuse "+strconv.FormatUint(m.HeapInuse, 10)+" bytes")
			return
		}

		maxHeapInUse = m.HeapInuse
	}
}

func storeHeapProfile(ctx context.Context, buf *bytes.Buffer, hostUncachedClient client.Client, prefix string) error {

	data := buf.Bytes()

	const chunkSize = 500 * 1024 // 500 kB

	// Split the byte array into chunks
	var chunks [][]byte
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize

		if end > len(data) {
			end = len(data)
		}

		chunks = append(chunks, data[i:end])
	}

	read_write_layer.GetSecret()

	return nil
}
