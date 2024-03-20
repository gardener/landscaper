---
title: Landscaper Critical Problems
sidebar_position: 19
---

# Critical Problems

Some of the most critical problems of the Landscaper are panics like nil pointer accesses, which occurs in the 
complex processing of Installations, Executions and DeployItems. Therefore, the most frequent panics 
are handled by a particular function `HandlePanics` preventing a crash of the process. Furthermore, the next 
reconciliation trial of the affected resource is delayed for 5 minutes. 

```
func HandlePanics(ctx context.Context, result *reconcile.Result, hostUncachedClient client.Client)
```

For the most important controllers (for Installations, Executions and DeployItems), if a handled panic
occurs, an entry with timestamp is added to `spec.criticalProblem` in a custom resource of type `criticalproblems`
with the name `critical-problems`. This custom resource is in the same namespace as the pod in which the controller 
with the panic is running. 

An example of a `criticalproblems` custom resource is shown here, which contains 3 critical problem entries. 
If there are more than 10 entries, the oldest will be deleted. 

```code
kubectl get criticalproblems -n ls-system  critical-problems -oyaml


kind: CriticalProblems
metadata:
  creationTimestamp: "2024-03-19T13:57:44Z"
  generation: 8
  name: critical-problems
  namespace: test0001-2f9e5e91
  resourceVersion: "214647151"
  uid: 368b0856-e844-4bea-b42e-7f7333faac76
spec:
  criticalProblem:
  - creationTime: "2024-03-19T13:57:44Z"
    podName: test0001-2f9e5e91
  - creationTime: "2024-03-19T14:02:44Z"
    podName: test0001-2f9e5e91
  - creationTime: "2024-03-19T14:07:44Z"
    podName: test0001-2f9e5e91
```

Every problem entry possesses a creation timestamp and the pod name, on which the corresponding problem happened.

The `criticalproblems` custom resource could be used to regularly check if something critical happened with respect
to the processing of Installations, Executions and DeployItems. If this is the case, you should check the logs to further
analyse the problems.
