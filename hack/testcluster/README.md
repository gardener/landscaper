# testcluster helper tool

This package contains a helper tool that is used in the Landscaper integration tests and local development to create and delete test clusters and test registries.

The testcluster tool consists of 2 subcommands: `cluster` and `registry` and each one is also exposed for the testmachinery as their own TestDefinition.
See the definition in [.test-defs](../../.test-defs).

#### Cluster

The cluster subcommand can create and delete kubernetes clusters (backed by k3d) in a kubernetes cluster.

```
$ testcluster cluster create --kubeconfig HOST_KUBECONFIG --id [UNIQUE_ID]
```
```
$ testcluster cluster delete --kubeconfig HOST_KUBECONFIG --id [UNIQUE_ID]
```

The cluster is scheduled as a pod in a kubernetes cluster and is only reachable within the k8s cluster with the pod ip.
Corresponding kubeconfig can be exported to a file using the `--export PATH` flag.

#### Registry

The registry subcommand can create and delete oci registries (backed by registry/registry) in a kubernetes cluster.

```
$ testcluster registry create --kubeconfig HOST_KUBECONFIG --id [UNIQUE_ID]
```
```
$ testcluster registry delete --kubeconfig HOST_KUBECONFIG --id [UNIQUE_ID]
```
The registry is scheduled as a pod in a kubernetes cluster and is only reachable within the k8s cluster with the service name.
The registry serves a https endpoint with a self-signed certificate that has included the service name (`<service-name>.<service-namespace>`and `<service-name>.<service-namespace>`).

> Note: the certificates root CA is currently not exported but only stored in a secret in the cluster.

Corresponding docker auth file can be exported to a file using the `--registry-auth PATH` flag.