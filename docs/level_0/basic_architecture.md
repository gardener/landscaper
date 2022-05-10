# Basic Landscaper Architecture

This document provides a first high level overview of the Landscaper architecture.

## Clusters and Target Environments

Let's first of all clarify the overall system setup. A typical Landscaper setup consists of the following
k8s clusters and target environments:

- Landscaper Cluster: The k8s cluster in which the Landscaper controllers run. In that sense, this is the cluster hosting the control plane of the Landscaper.

- Resource Cluster: On this cluster, end users create their `Installation` and `Target`custom resources to trigger the 
  intended installation processes. The Landscaper controller is watching these `Installation` resources and initiates the needed 
  actions.

- Target Environments (including Target Clusters): Target environments describe the environment where 
  software/components are installed/deployed. Examples are target cluster where applications should be deployed to 
  or some network infrastructure on which a virtual network should be set up.

## Landscaper Controller

Landscaper consists of two controllers:

- Installation Controller: The Installation Controller watches the `Installation` custom resources. If all required
  import data for an `Installation` is available, it provides this data to the `DeployItems` of the `Blueprint(s)`
  referenced in the `Installation`. It creates so called `Execution` custom resources, which are
  collections of DeployItems with their required import data. 
  
- Execution Controller: The Execution Controller watches the `Execution` custom resources and splits them into
  the individual DeployItems by creating `DeployItem` custom resources. A particular `DeployItem` custom resource
  is only created when all its specified predecessors are already available.
  
## Deployer

DeployItems exist for different types, each describing different installation methods, e.g. DeployItems of type
`Helm` describe the installation of Helm charts. For every type, a dedicated Deployer is responsible. This Deployer is watching the corresponding 
DeployItems and executes the installation instructions specified in these DeployItems. 

Currently available types with corresponding Deployers are:

  - helm: DeployItems specifying the installation of a Helm chart
  - manifest: DeployItems specifying the installation of Kubernetes manifests
  - container: DeployItems specifying a container image
  - terraform:  DeployItems specifying a Terraform installation

Deployer usually run separated from the Landscaper. It is possible to extend Landscaper by introducing new types for 
DeployItems with a corresponding Deployer.
