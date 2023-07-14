# Scaling of Landscaper Pods

## Horizontal Pod Autoscaling

The horizontal scaling behavior of the Landscaper pods is defined in [HorizontalPodAutoscaler][1] (HPA) objects.
Each HPA specifies a minimum and maximum number of pods. Within these limits, a new pod will be started if the
average cpu or memory consumption exceeds 80% of the value specified in the corresponding Deployment.
The pods are distributed evenly across nodes and zones.

- [Landscaper HPA](../../charts/landscaper/charts/landscaper/templates/hpa-landscaper.yaml):  
  Currently, we start exactly 1 landscaper pod, because the controllers are not yet prepared to run with multiple 
  replicas in parallel.  
- [Webhok HPA](../../charts/landscaper/charts/landscaper/templates/hpa-webhook.yaml):  
  We run at least 2 webhook pods, because users would directly notice if the webhook were unavailable.
- [Container deployer HPA](../../charts/container-deployer/templates/hpa.yaml)
- [Helm deployer HPA](../../charts/helm-deployer/templates/hpa.yaml)
- [Manifest deployer HPA](../../charts/manifest-deployer/templates/hpa.yaml)
- [Mock deployer HPA](../../charts/mock-deployer/templates/hpa.yaml)



## Statistics

### Memory and CPU

The Landscaper logs periodically cpu and memory data from the status of the HPA objects:

```json
{
  "level":"info",
  "ts":"2023-07-13T11:33:30.205Z",
  "logger":"controllers.landscaper-monitoring",
  "msg":"HPA Statistics",
  "resource":"ls-system/helm-default-helm-deployer",
  "currentReplicas":3,
  "desiredReplicas":3,
  "memoryAverageUtilization":20,       // percentage of the value specified in the Deployment
  "memoryAverageValue":"64625322666m",
  "cpuAverageUtilization":108,         // percentage of the value specified in the Deployment
  "cpuAverageValue":"325m"
}
```

### Worker Count

Each controller has a certain number of worker threads per pod. These numbers can be configured in the values of the 
corresponding helm chart. Each controller pod counts how many of these worker threads are currently in use.
The counter is logged at the beginning of a reconciliation if it exceeds 70% of the maximum.
To find these logs, search for info messages "worker threads of controller".

```json
{
  "level":"info",
  "msg":"worker threads of controller installations",
  "reconciledResourceKind":"Installation",
  "usedWorkerThreads":10
}
```

## Locking

There are **controllers** reconciling **objects**, for example the helm deployer reconciles DeployItems. 
Several **replicas** (pods) of a controller can be started, for example as a consequence of horizontal pod autoscaling. 
We want to ensure that no object is processed by two replicas of a controller in parallel.
Therefore, the replicas of a controller must synchronize which of them processes an object. This is done via
**SyncObject** custom resources. A SyncObject serves as a lock for a controller-object pair.

### One SyncObjects per Controller-Object Pair

There exists (at most) one SyncObject for a controller-object pair.

The name of a SyncObject is composed of two parts:
- an identifier of the controller (`container`, `helm`, `manifest`, `mock`, `landscaper-helm`, ...)
- and an identifier of the object, namely its UID. (Not its name!)

As identifier of an object we use its UID, not its name. If you delete an object and then create one with the same name, 
we consider it as a new object, and it gets new SyncObjects.

Note that two **different** controllers are allowed to process the same object in parallel, for example a 
deployer and the timeout controller. So, an object can have more than one SyncObject, namely for different controllers. 
What the synchronisation prevents is that different replicas of the **same** controller process the same object in 
parallel.

### Locking and Unlocking

If a replica of a controller is about to process an object, it tries to obtain the lock for this controller-object pair.
This is done by creating/updating the corresponding SyncObject with the identifier of the replica.
As identifier of a replica we use the pod name.

Optimistic locking of the create/update operation ensures that only one replica can obtain the lock.

Unlocking is done at the end of a reconciliation. The processing replica (pod) removes its identifier (pod name) from 
the SyncObject. It does not delete the SyncObject. 

### Controlling the Lock Owner

Other replicas of the same deployer, which do not get the lock, check whether the current owner of the lock still exists. 
So if a pod dies without unlocking, the other pods will recognise this, and can take over the lock.

### Cleanup

A go function deletes SyncObjects whose corresponding object (the object with the matching UID) does not exist. 

It does not matter if in the meantime a new object with the same name is being created. The locking of the deleted old object
and the new object is done by different SyncObjects, because of the different UIDs. 
(If we would use the object name instead of the UID, the cleanup might delete a SyncObject which was just recycled and 
locks a new object with the same name. The delete operation would not conflict with a parallel update, because there is 
no optimistic locking for delete operations.)



## Responsibility Check Based on Metadata

Every deployer watches all DeployItems. For example, the container deployer receives reconcile events for Helm 
DeployItems. Therefore, deployers check at first their responsibility for a DeployItem. 
Moreover, DeployItem can be large, for example due to large Helm values. Therefore, the deployers check their
responsibility by only reading the small **metadata** of a DeployItem.

The information required for the responsibility check are the deployer type (container, helm, manifest, mock) and the
name of the Target. These data actually belong to the spec of a DeployItem. We redundantly store them in the metadata as 
annotations:

- annotation `landscaper.gardener.cloud/deployer-type` contains the deployer type (same value as field `spec.type`)
- annotation `landscaper.gardener.cloud/deployer-target-name` contains the Target name (same value as field `spec.target.name`)

### Steps of the Responsibility Check
 
- Read the metadata of the DeployItem to get the deployer type and the target name from the annotations.
- (If new annotations are missing (directly after upgrade), read the full DeployItem.)  
- Responsibility check 1: check the deployer type, and return if not responsible.  
- Read the Target.  
- Responsibility check 2: check target selectors, and return if not responsible.  
- Resolve Target.  
- Lock.  
- Reconcile (the main part).    
- Unlock.  


<!-- References -->

[1]: https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/  



