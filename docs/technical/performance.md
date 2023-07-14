# Performance Analysis

This document describes the current state of the performance analysis of Landscaper used in the context
of the Landscaper as a Service ([LaaS](https://github.com/gardener/landscaper-service)) with
[Gardener Clusters](https://github.com/gardener/gardener). 

# Test 1

## Test Setup

Usage of one Landscaper instance of the Dev-Landscape with the test data from 
[here](https://github.com/gardener/landscaper-examples/tree/master/scaling/many-deployitems/installation3) consisting of:

- 6 root installations
- 50 sub installations for every root installation
- One deploy item for every sub installation
- Every deploy item deploys a helm chart with a config map with about 1,3kB input data 

## Test Results

This chapter shows the duration for deploying the 6 root installations for different versions of the Landscaper and 
our current interpretation of the results.

### Current Landscaper Version

Tests with the current official Landscaper release with LaaS v0.71.0

- **Duration: 25:00 (minutes/seconds)**

Investigations showed that main reason for the bad performance in the first tests was due to request rate limits of the 
kubernetes clients. You can find the following entries in the logs indicating this:

```
Waited for 8.78812882s due to client-side throttling, not priority and fairness, request: GET:https://api.je09c359.laasds.shoot.live.k8s-hana.ondemand.com/apis/landscaper.gardener.cloud/v1alpha1
```

### Landscaper with improved client request rate limits

Tests with a Landscaper with a client having very high request rate limits (burst rate and queries per second = 10000).

- 30 worker threads for installations, executions, deploy items (LaaS version: v0.72.0-dev-11d2919a8e2bce4a02c3928f7a49fe183d35f63d) 
  - **Duration: 4.16**
  

- 60 worker threads for installations, executions, deploy items (LaaS version: v0.72.0-dev-8db791bf996047f1b849207472ff9d97bac80481)
  - **Duration: 4:10**


- 120 worker threads for installations, executions, deploy items (LaaS version: v0.72.0-dev-66eb650b1156d7eaced0b3e63def4a8dc0f6cbff)
  - **Duration: 5:02**


- 310 worker threads for installations, executions, deploy items (LaaS version: v0.72.0-dev-eb5bb0f8424f25a6ae2871e0bc9f1c50d35228f8)
  - **Duration: 4:41**

    
The tests show: 
  - The performance is much better compared to the k8s client with rate limiting. 
  - The number of parallel worker threads should not be increased too far.

### Landscaper with improved client request rate limits and parallelisation

Tests with a Landscaper with a client having very high request rate limits (burst rate and queries per second (qps) = 10000)
and multiple replicas for the pods running the controller for installations, executions and helm deploy items.

LaaS version: v0.72.0-dev-7f456ae4edb6a86847bb210e25ef9c3f26ed6ada

- 1 pods für inst, exec, di controller: **Duration: 4:16**

- 2 pods für inst, exec, di controller: **Duration: 2:24**

- 3 pods für inst, exec, di controller: **Duration: 1:21**

- 4 pods für inst, exec, di controller: **Duration: 1:30**

- 5 pods für inst, exec, di controller: 
  - error with message: 'Op: CreateImportsAndSubobjects - Reason: ReconcileExecution - Message:
      Op: errorWithWriteID - Reason: write - Message: write operation w000022 failed
      with Get "https://[::1]:443/api/v1/namespaces/cu-test/resourcequotas": dial
      tcp [::1]:443: connect: connection refused'

The tests show:

- Activating the parallelization results in a similar performance for the one pod scenario, though there are more
  requests to the API server for synchronization. 
- Further increasing the number of pods results in a better performance.
- Going beyond some number of pods, the API server becomes overloaded and the deployment fails.

### Landscaper with restricted Burst and QPS rates

These tests were executed with restricted burst and qps rates and no parallelization. 

LaaS version: v0.72.0-dev-baa5654c9e727a70e568e24407277181c0aef1b3

- burst=30, qps=20: **Duration: 6:48**
- burst=60, qps=40: **Duration: 4:25** (default settings)
- burst=80, qps=60: **Duration: 4:20**

The results sho that the default settings give quite good results.

For settings other than the default, the configuration of the root installation of a landscaper instance in a LaaS 
landscape has to be adapted as follows:

```yaml
    landscaperConfig:
      k8sClientSettings:            # changed
        resourceClient:             # changed
          burst: <newValue>         # changed
          qps: <newValue>           # changed
      deployers:
      - helm
      - manifest
      - container
      deployersConfig:              # changed
        helm:                       # changed
          deployer:                 # changed
            k8sClientSettings:      # changed
              resourceClient:       # changed
                burst: <newValue>   # changed
                qps: <newValue>     # changed
        manifest:                   # changed
          deployer:                 # changed
            k8sClientSettings:      # changed
              resourceClient:       # changed
                burst: <newValue>   # changed
                qps: <newValue>     # changed
```

## Conclusions

The communication with the API server of the resource cluster has a big influence on the Landscaper performance. 
Increasing the request restrictions of the k8s client used by the Landscaper results in a speed-up of about 6.
Parallelization could further improve the performance by a factor of 3.

Unfortunately, if the number of requests to the API server becomes too high, the API server might become unresponsive 
resulting in deployment errors. Due to the large amount of different usage scenarios, it is currently hard to judge 
which setup is optimal with respect to performance and stability.

For now we decide to release the Landscaper with no parallelization and the default restricted burst and qps rates 
(60/40). If there will be problems with an overloaded API server, the values could be reduced accordingly.

So far the tests were quite restricted and other usage pattern might also show different bottlenecks like huge memory 
consumption etc. Therefore, we need to investigate this on our productive landscapes for the different customer scenarios. 
  
