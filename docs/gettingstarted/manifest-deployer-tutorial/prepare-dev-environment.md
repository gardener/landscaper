# Prepare demo environment

You need to have:

- 1 Kubernetes cluster under your control
- Latest version of landscaper-cli and component-cli installed

## The K8S cluster

There are several ways to get a K8S cluster, we prefer to use Gardener Kubernetes clusters.

Go here: <https://dashboard.garden.canary.k8s.ondemand.com/login> to get a Gardener Kubernetes Cluster (trial).

When you create the Gardener Kubernetes cluster, ensure the following:

- Length of cluster name should not exceed 5 letters, e.g. "demo"
- In the Add-ons section, select the "Nginx Ingress" (Default ingress-controller with static configuration...)

## The tools

Install the Landscaper CLI with the [installation instructions](https://github.com/gardener/landscapercli/blob/master/docs/installation.md) which will include Component CLI as well or do a manual binary download as described below.

### Getting and setting up the Landscaper CLI

1. Check, which is the latest version of Landscaper CLI on <https://github.com/gardener/landscapercli/releases/latest>

2. Get latest release e.g.

    ```bash
    curl -L -O https://github.com/gardener/landscapercli/releases/download/v0.17.0/landscapercli-linux-amd64.gz
    ```

3. Unpack with

    ```bash
    gzip -d ./landscapercli-linux-amd64.gz
    ```

4. Make it executable with

    ```bash
    chmod +x ./landscapercli-linux-amd64
    ```

5. (optional) Copy to bin with

    ```bash
    sudo mv landscapercli-linux-amd64 /usr/local/bin/landscaper-cli
    ```

### Getting and setting up the Component CLI

1. Check, which is the latest version of Component CLI on https://github.com/gardener/component-cli/releases/latest
2. 
3. Get latest release e.g.

    ```bash
    curl -L -O https://github.com/gardener/component-cli/releases/download/v0.40.0/componentcli-linux-amd64.gz
    ```

4. Unpack with

    ```bash
    gzip -d ./componentcli-linux-amd64.gz
    ```

5. Make it executable with

    ```bash
    chmod +x ./componentcli-linux-amd64
    ```

6. (optional) Copy to bin with

    ```bash
    sudo mv componentcli-linux-amd64 /usr/local/bin/component-cli
    ```
