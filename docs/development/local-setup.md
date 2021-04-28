# Preparing the Setup

Conceptually, all Landscaper components are designated to run as a Pod inside a Kubernetes cluster.
The Landscaper extends the Kubernetes API by using CRD's that are automatically deployed by the Landscaper controller during startup.
If you want to develop it, you may want to work locally with the Landscaper without building a Docker image and deploying it to a cluster each and every time.
That means that the Landscaper controller runs outside a Kubernetes cluster which requires providing a [Kubeconfig](https://kubernetes.io/docs/tasks/access-application-cluster/authenticate-across-clusters-kubeconfig/) in your local filesystem and point the Landscaper controller to it when starting it (see below).

Deployers are separate controllers that extend the Landscaper and reconcile DeployItems.
These Deployers are designed to run as separate controllers in the cluster.
However, for development purposes most deployers are included in the Landscaper controller and can be run with the Landscaper controller.

Further details could be found in

1. [Principles of Kubernetes](https://kubernetes.io/docs/concepts/), and its [components](https://kubernetes.io/docs/concepts/overview/components/)
1. [Kubernetes Development Guide](https://github.com/kubernetes/community/tree/master/contributors/devel)

This setup is based on [k3d](https://github.com/rancher/k3d).
Docker for Desktop, [minikube](https://github.com/kubernetes/minikube) or [kind](https://github.com/kubernetes-sigs/kind) are also supported.

## Installing Golang environment

Install latest version of Golang. For MacOS you could use [Homebrew](https://brew.sh/):

```bash
brew install golang
```

For other OS, please check [Go installation documentation](https://golang.org/doc/install).

:warning: The Landscaper uses some features that are only available since golang version 1.16. Make sure to use at least golang version 1.16.x.

## Installing kubectl and helm

As already mentioned in the introduction, the communication with the Gardener happens via the Kubernetes (Garden) cluster it is targeting. To interact with that cluster, you need to install `kubectl`. Please make sure that the version of `kubectl` is at least `v1.19.x`.

On MacOS run

```bash
brew install kubernetes-cli
```

Please check the [kubectl installation documentation](https://kubernetes.io/docs/tasks/tools/install-kubectl/) for other OS.

The Landscaper and its Deployers are deployed using Helm Charts, so you also need the [Helm](https://github.com/kubernetes/helm) CLI:

On MacOS run

```bash
brew install helm
```

On other OS please check the [Helm installation documentation](https://helm.sh/docs/intro/install/).

## Installing git

We use `git` as VCS which you need to install.

On MacOS run

```bash
brew install git
```

On other OS, please check the [Git installation documentation](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git).


## Installing K3d

You'll need to have [Docker](https://docs.docker.com/get-docker/) installed and running.

Follow the [k3s installation guide](https://k3d.io/#installation) to start a kubernetes cluster.

:warning: note that with the default k8s cluster installation some development resources like target may not work as on some os's docker is running in VM with different network access.
We therefore recommend to use our local setup script in [hack/setup-local-env.sh](../../hack/setup-local-env.sh) which automatically setups a local environment. 
In that environment controllers running in- and outside of the cluster can access the apiserver. (Note that this script modifies the `/etc/hosts` file which may require root permissions)

## [MacOS only] Install GNU core utilities

When running on MacOS you have to install the GNU core utilities:

```bash
brew install coreutils gnu-sed
```

This will create symbolic links for the GNU utilities with `g` prefix in `/usr/local/bin`, e.g., `gsed` or `gbase64`. To allow using them without the `g` prefix please put `/usr/local/opt/coreutils/libexec/gnubin` at the beginning of your `PATH` environment variable, e.g., `export PATH=/usr/local/opt/coreutils/libexec/gnubin:$PATH`.

## [Optional] Installing gcloud SDK

In case you have to create a new release or a new hotfix of the Landscaper you have to push the resulting Docker image into a Docker registry. Currently, we are using the Google Container Registry (this could change in the future). Please follow the official [installation instructions from Google](https://cloud.google.com/sdk/downloads).

## Local Landscaper setup

### Get the sources

Clone the repository from GitHub into your `$GOPATH`.

```bash
mkdir -p $GOPATH/src/github.com/gardener
cd $GOPATH/src/github.com/gardener
git clone git@github.com:gardener/landscaper.git
cd landscaper
```

> Note: Landscaper is using Go modules and cloning the repository into `$GOPATH` is not a hard requirement. However it is still recommended to clone into `$GOPATH` because `k8s.io/code-generator` does not work yet outside of `$GOPATH` - [kubernetes/kubernetes#86753](https://github.com/kubernetes/kubernetes/issues/86753).

### Before you start

:warning: Before you start developing, please have an understanding about the [principles of Kubernetes](https://kubernetes.io/docs/concepts/), and its [components](https://kubernetes.io/docs/concepts/overview/components/), what their purpose is and how they interact with each other.

#### Run the Landscaper

When the Kubernetes cluster is running, start the Landscaper controller with the cluster's kubeconfig.
The controller must be started without any webhooks as they can only be used inside the cluster.

```bash
go run ./cmd/landscaper-controller --disable-webhooks=all --kubeconfig=$KUBECONFIG
```

(Optional) Configure the landscaper controller
The Landscaper controller is configured with a configuration file that can be provided via `--config` flag.

```yaml
apiVersion: config.landscaper.gardener.cloud/v1alpha1
kind: LandscaperConfiguration

registry:
  oci:
    allowPlainHttp: true # for localhost:5000 clusters
    insecureSkipVerify: true 
    configFiles:
    - "/path/to/docker/auth/config.yaml"

metrics:
  port: 8080
crdManagement:
  deployCrd: true
  forceUpdate: true
```
