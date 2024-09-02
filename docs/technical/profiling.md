# Profiling Landscaper Pods

This page describes how to obtain information about the memory consumption of the Landscaper pods.
The procedure is supported for the helm and manifest deployer, as well as for the main and central Landscaper pods.


## Preparation

Modify the Deployment of the controller in question: add the environment variable `ENABLE_PROFILER` with string value `true`
to the template of the container:

```yaml
spec:
  template:
    spec:
      containers:
      - name: ...
        env:
        - name: ENABLE_PROFILER
          value: "true"
```

## Memory Information in the Logs

If the environment variable `ENABLE_PROFILER` is set as described above, the controller writes every minute a log entry
with information about the memory consumption of the pod:

```text
MemStats: Alloc: 28 MiB, TotalAlloc: 948 MiB, Sys: 78 MiB, Lookups: 0, Mallocs: 3914825, Frees: 3704375, HeapAlloc: 28 MiB, HeapSys: 71 MiB, HeapIdle: 39 MiB, HeapInuse: 31 MiB, HeapReleased: 25 MiB, HeapObjects: 210450, StackInuse: 0 MiB, StackSys: 0 MiB, MSpanInuse: 0 MiB, MSpanSys: 0 MiB, MCacheInuse: 0 MiB, MCacheSys: 0 MiB, BuckHashSys: 1 MiB, GCSys: 3 MiB, OtherSys: 0 MiB, NextGC: 53 MiB, LastGC: 2024-08-02T14:16:11Z, PauseTotalNs: 7400816 ns, NumGC: 77, NumForcedGC: 0, GCCPUFraction: 0.005614411177248498, EnableGC: true, DebugGC: false,  
***** 
Generic process memory info: RSS: 104 MiB, VMS: 5442 MiB, HWM: 0 MiB, Data: 0 MiB, Stack: 0 MiB, Locked: 0 MiB, Swap: 0 MiB  
***** 
Platform specific process memory info: {"rss":109842432,"vms":5707378688,"shared":57397248,"text":40964096,"lib":0,"data":0,"dirty":117972992}
```

You can search for `MemStats` in the logs to find these entries.

The comments in [MemStats](https://pkg.go.dev/runtime#MemStats) explain the fields of the MemStats structure.


## Heap Dump

If the environment variable `ENABLE_PROFILER` is set as described above, you can obtain a heap dump as follows.

1. On the Landscaper host cluster define a port-forward:

   ```shell
   kubectl port-forward -n <LANDSCAPER_NAMESPACE> <POD_NAME> 8081:8081
   ```

2. Trigger a garbage collection first, generate a heap dump and download it to some file (here `heap.out`):

   ```shell
   curl 'http://localhost:8081/debug/pprof/heap?gc=1' > heap.out
   ```

   Afterwards, you can stop the `port-forward`.

3. Display the generated heap dump with the pprof tool:

   ```shell
   go tool pprof -http=:8082 heap.out
   ```

## Automatic Heap Dump

If the environment variable `ENABLE_PROFILER` is set as described above a heap dump is automatically created if

- there was no automatic heap dump created for the pod since it started and the current HeapInUse is more than 330 MB
- there was already an automatic heap dump created for the pod since it started and the current HeapInUse is more than
  10 percentage higher than for the formerly created heap dump. The older heap dump is overwritten by the newer one.

An automatic heap dump is written into a set of secrets, consisting of a base secret with some meta data and several
data secrets containing the byte data of the heap dump. 

The base secret is stored in the same namespace as the corresponding pob. Its name is as follows for the different 
pods:

- Landscaper main pod with Installation and Execution controllers: main-landscaper-heap
- Landscaper pod for rest: central-landscaper-heap
- Helm deployer pod: helm-deployer-heap
- Manifest deployer pod: manifest-deployer-heap

A base secret contains the following entries:

- keyNumberOfDataSecrets: number of secrets containing the data of the profile
- keyHeapInUse: heap in use in bytes
- keyHeapAlloc: heap alloc in bytes
- keyPodname: name of the pod for which a heap dump was stored
- keyStorageDate: timestamp when the heap dump was stored

For every base secret with Name `<prefix>-heap` there is a set of secrets with names `<prefix>-heap-0`, `<prefix>-heap-1`
etc. containing the heap data. 

To analyse a heap dump you have to execute the following commands:

``` bash
# store the heap data in different files
kubectl get secret -n <namespace> <prefix>-heap-0 -o jsonpath="{.data.keyBytes}" | base64 -d > heap0.out
kubectl get secret -n <namespace> <prefix>-heap-1 -o jsonpath="{.data.keyBytes}" | base64 -d > heap1.out
...

# concatenate the files
cat heap0.out heap1.out ... > heap.out

# analyse the result
go tool pprof -http=:8082 heap.out
```

## Some other important commands

Sometimes it might be interesting to see the memory consumption of a pod and the containers running in it.

To see the memory usage of a pod use:

   ```shell
   kubectl top pods -n <LANDSCAPER_NAMESPACE>
   ```
If you want to see the memory consumption of a particular container in a pod call:

   ```shell
   kubectl debug <POD_NAME> -n <LANDSCAPER_NAMESPACE> -it --image=jonbaldie/htop --share-processes=true --target <CONTAINER_NAME>
   ```

The names of the containers of a pod cou be found with:

   ```shell
   kubectl describe pods -n <LANDSCAPER_NAMESPACE> <POD_NAME>
   ```

## References

https://pkg.go.dev/runtime#MemStats

https://pkg.go.dev/runtime/pprof

https://pkg.go.dev/net/http/pprof

https://github.com/google/pprof/blob/main/proto/README.md

