# Performance Analysis

This document describes the last state of the performance analysis of Landscaper used in the context
of the Landscaper as a Service ([LaaS](https://github.com/gardener/landscaper-service)) with
[Gardener Clusters](https://github.com/gardener/gardener). 

## Initial Situation

Tests with Landscaper version v.0.90.0.

Installations were created in one namespace in steps of 200. 

The following shows the duration for a packet of 200 Installations to be finished:

- First 200:   183 s
- Next  200:   326 s
- Next  200:   501 s
- Next  200:   598 s
- Next  200:   771 s
- Next  200:   976 s
- Next  200:   1170 s (1 Installation failed)
- Next  200:   1242 s (6 Installations failed)

After the creation of these 1600 Installations in one namespace another packet of 200 Installations were created in 
another namespace. The duration for this was 185s.

Conclusion: If the number of Installations in one namespace increases, also the duration for their executions increases 
heavily. If there are already 1000 Installations in a namespace, the execution of further 200 Installations requires about 
20 minutes. The reason for this is the huge amount of list operations with label selectors, the Landscaper executes against 
the API server of the resource cluster. 

## Improvements

The following improvements where implemented to reduce the number of list operations with label selections:

- DeployItems cache in the status of Executions: [PR](https://github.com/gardener/landscaper/pull/935)
  - Used to directly access the DeployItems instead of fetching them via list oerations 
- Subinstallation cache in the status of Installations: [PR](https://github.com/gardener/landscaper/pull/936)
  - Used to directly access the Subinstallations instead of fetching them via list oerations
- Sibling import/export hints: [PR](https://github.com/gardener/landscaper/pull/937)
  - Prevent list operations to compute predecessor and successor installations if no data is exchanged   

The improvements were tested with the following test setup:

- One Lansdscaper instance with 10 namespaces. 
- In every namespace about 1000 Installations with 1000 Executions and 1000 DeployItems. The DeployItems just
  install a configmap. There are no sibling exports or imports and these flags are set on true in the Installations.
- One helm deployer pod with 120 worker threads. 
- Ome main controller pod with 60 worker threads for Installations and 60 worker threads for Executions.

The tests were executed with an old Landscaper version v.0.90.0 and a Landscaper with the improvements described above.

Test results: 

- Creation of 1000 Installations/1000 Executions/1000 Deploy Items in one namespace
    - Duration before optimisation: 3046s
    - Duration after optimisation:  1050s

- Update of 1000 Installations/1000 Executions/1000 Deploy Items in one namespace
    - Duration before optimisation: 3601s
    - Duration after optimisation:  1166s

The creation and update time for 1000/1000/1000 objects remain stable until 20.0000 Installations with 20.0000 Executions 
and 20.0000 DeployItems were created in 20 different namespaces. No tests were executed with more objects so far.

## Comparison with cached client

The optimized version was compared with a version using a cached k8s client. The test setup was similar to the chapter
before.

- Creation of 500 Installations/500 Executions/500 Deploy Items in another namespace
  - Duration with optimisation: 400s
  - Duration with cached client: 228s

- Update of 500 Installations/500 Executions/500 Deploy Items in one namespace
  - Duration with optimisation: 389s
  - Duration with cached client: 217s

The memory consumption of the version with the cached client was about ten time more than for the optimized version:


**Memory consumption of the optimised version:**

```
      NAME                                                             CPU(cores)   MEMORY(bytes)   
      container-test0001-2f9e5e91-container-deployer-5f646cff6-5vqjd   2m           98Mi            
      helm-test0001-2f9e5e91-helm-deployer-d8b7744b6-wxslx             312m         318Mi           
      landscaper-test0001-2f9e5e91-7f844f9f7c-9mfsq                    9m           157Mi           
      landscaper-test0001-2f9e5e91-main-545ccccc6d-75qpl               164m         343Mi           
      manifest-test0001-2f9e5e91-manifest-deployer-7c555589bd-wdwf8    2m           79Mi 
```

**Memory consumption of version with cached client:**

```
      NAME                                                              CPU(cores)   MEMORY(bytes)   
      container-test0001-2f9e5e91-container-deployer-697d7b6449-6b6vf   15m          240Mi           
      helm-test0001-2f9e5e91-helm-deployer-6ff7686c6f-zl5rc             1664m        4445Mi          
      landscaper-test0001-2f9e5e91-7776698fb-lx56p                      32m          627Mi           
      landscaper-test0001-2f9e5e91-main-6bcbd8788c-j25mh                508m         2268Mi          
      manifest-test0001-2f9e5e91-manifest-deployer-6d546d9c6c-46dz2     20m          845Mi   
```

## Duration for small numbers without sibling hints

The following shows the duration to create or update only a few number of Installations/Executions/DeployItems in a new 
and empty namespace whereby the sibling hints of optimisation three are not used. The cluster already contains about 
20.0000 Installations with 20.0000 Executions and 20.0000 DeployItems in 20 namespaces. 

100/100/100: create: 173s - update: 159s - delete: 63s
200/200/200: create: 323s - update: 272s - delete: 115s
300/300/300: create: 413s - update: 401s - delete: 175s
400/400/400: create: 543s - update: 542s - delete: 285s
500/500/500: create: 678s - update: 659s - delete: 394s

Here the corresponding numbers if the sibling hints are activated:

100/100/100: create: 109s - update: 106s - delete: 59s
200/200/200: create: 183s - update: 176s - delete: 120s
300/300/300: create: 263s - update: 242s - delete: 204s
400/400/400: create: 356s - update: 329s - delete: 365s
500/500/500: create: 429s - update: 436s - delete: 532s


## Improve startup behaviour

With more k8s objects in a resource cluster, the startup times for the Landscaper become much slower because all watched
objects are presented first to the controller. When restarting the Landscaper watching a resource cluster with about 
20.0000 Installations with 20.0000 Executions and 20.0000 DeployItems in 20 namespaces, it requires about 10 minutes 
until Landscaper starts processing newly created Installations.

After introducing a startup cache ([see](https://github.com/gardener/landscaper/pull/948)) Landscaper requires only 30 s 
until the processing the newly created Installations starts.

Beside the startup problem, also the periodic reconciliation of all watched items of a controller every 10 hours, prevents
the execution of modified items for several minutes. Therefore, the frequency of this operation was reduced to 1000
days, such that this should not happen anymore, because the pods are usually restarted before at least during the regular 
updates.

## Reduce impact of unintended complete reconciliations

Sometimes about 15 minute after a restart, all watched objects are reconciled again, though there were no modifications.
Currently, we have no explanation for this behaviour of the controller runtime, and it is unclear how often this happens. 
By improving the startup cache, storing all finished objects ([see](https://github.com/gardener/landscaper/pull/953)), 
such load peaks could be handled withing less than one minute for about 20.000 objects. 




