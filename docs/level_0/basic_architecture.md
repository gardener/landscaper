# Basic Landscaper Architecture

This document provides a first high level overview of the Landscaper architecture.

## Involved Clusters and Target Environments

In this chapter we want to clarify the overall system setup. A typical Landscaper set up consists of the following
k8s clusters and target environments:

- Landscaper Cluster: The k8s cluster where Landscaper runs.

- Resource Cluster: On this cluster, end users are creating their `Installation` custom resources to trigger the 
  intended installation processes. Landscaper is watching these `Installation` resources and initiates the specified 
  actions.

- Target Environments (including Target Clusters): Target environments  describe the environment where 
  software/components are installed/deployed. Examples are target k8s cluster where some helm charts should be deployed 
  or some network infrastructure on which a virtual network should be set up.

## Landscaper Controller

Landscaper consists of two controllers:

- Installation Controller: The Installation Controller watches the `Installation` custom resources. If all
  import data of an `Installation` is available it provides the input data to the DeployItems of the Blueprint
  referenced in the `Installation`. It creates so called `Execution` custom resources which are more or less
  collections of DeployItems with their import data. 
  
- Execution Controller: The Execution Controller watches the `Execution` custom resources and splits them into
  the individual DeployItems by creating `DeployItem` custom resources. A particular `DeployItem` custom resource
  is only created when all its specified predecessors are already available.
  
## Deployer

There exist DeployItems of different types describing different installation methods, e.g. DeployItems of type
helm describe the installation of helm charts. For every type there exists a Deployer watching the corresponding 
DeployItems and executing the installation instructions specified by these DeployItems. 

Currently available types with corresponding Deployers are:

  - helm: DeployItems specifying the installation of a helm chart
  - manifest: DeployItems specifying the installation of kubernetes manifests
  - container: DeployItems specifying a container image with some executable
  - terraform:  DeployItems specifying a terraform installation

Deployer usually run separated from Landscaper. It is possible to extend Landscaper by introducing new types for 
DeployItems with corresponding Deployer.