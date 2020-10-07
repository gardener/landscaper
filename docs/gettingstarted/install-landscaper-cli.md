# Install and Configure the Landscaper cli
By default, all runtime resources are CR so they can and should be simply accessed using `kubectl`.

The landscaper also interacts with resources that are not stored in a cluster.
Some of these resources include Blueprints, ComponentDescriptors or jsonschemas that are stored remote in a oci registry.

The Landscaper cli tool is mainly build to support human users interacting with these remote resources.
We may also think to improve the kubectl experiece but this will then be rather a kubectl plugin than its own cli tool.
(ref https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/)

## Install

The landscaper cli can be simple installed via go:

```shell script
go get github.com/gardener/landscaper/cmd/landscaper-cli

# or with a specific version
go get github.com/gardener/landscaper/cmd/landscaper-cli@v0.1.0
```
:warning: Make sure that the go bin path is set in your `$PATH` env var. `export PATH=$PATH:$GOPATH/bin`
