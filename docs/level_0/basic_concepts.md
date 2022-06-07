# Basic Concepts

This document describes the basic concepts, artifacts and their relationships within the Landscaper universe. 

TODO: Provide an overview picture containing all relevant entities like blueprint, deployExecution, installation and so on ...


## Main Objective of the Landscaper

The Landscaper is typically used to set up large cloud environments with complex relationships, consisting of many
small installation tasks like Helm deployments, network setup etc. These elementary installation tasks are defined by 
DeployItems. There are different types of DeployItems, for example for deploying Helm charts, plain kubernetes manifests 
or terraform configurations.

A DeployItem can be configured such that it can be used in different scenarios. Typical examples of configurable parts
of a DeployItem are the target cluster or namespace where a helm chart should be deployed to. A DeployItem can also 
define output parameters for data it creates and which can be consumed by others.

## Reusable Installation Tasks

So-called `Blueprints` are effectively reusable collections of combined installation tasks, which can be parameterized through import
parameters.

Several DeployItems can be defined in a single Blueprint. The execution order of the DeployItems contained in a Blueprint can also be specified.

These Blueprints define an interface for import data, as required by its DeployItems. They can also define an export 
interface to expose output data of its DeployItems. 

## Blueprint Component and Component Descriptor

A component is a quite general term usually describing some IT artifact ranging from a small program to large
and complex systems. In the context of Landscaper we focus on components setting up some cloud environment 
described with a Blueprint. This could range from quite simple installations of one or two helm charts, or the
setup up of a complex system like a Gardener landscape.

A Blueprint Component usually consists of a Blueprint, and the resources required by the DeployItems, like helm charts, docker images, json schema etc. All these required resources are described by a Component 
Descriptor as a yaml file. 

Component Descriptors are a complete description of all resources belonging to a component and could be used for
different tasks like security scanning or transport.

Usually Component Descriptors and Blueprints are stored in an OCI registry. More details about Component 
Descriptors and Component Descriptor Artifacts can be found [here](https://github.com/gardener/component-spec/blob/master/doc/proposal/00-overview.md).

## Installation

A Blueprints is a reusable definition of the installation process of a particular cloud environment. 
An instance of such a Blueprint with particular import data is called an Installation. Installations are Kubernetes 
Custom Resources (CR) defined by Landscaper. An installation references the Blueprint of a Component via the 
corresponding Component Descriptor. It specifies the import data and how to handle the export data. 

An Installation is deployed to the cluster where Landscaper watches these CRs. If Landscaper recognizes an installation
with all input data available, it starts the setup specified in the Blueprint. This executes the installation tasks 
defined in the DeployItems.

## Aggregated Blueprint

Very complex systems require hundreds of installation steps resulting in Blueprints with many DeployItems. 
Landscaper allows defining subsystems each described by a separate Blueprint and to combine these in an aggregated 
Blueprint. 

Assume an Installation referencing an aggregated Blueprint is deployed. When all input data of the Installation is 
available, the Sub-Installations for all contained Blueprints are created. Sub-Installations get their input data from 
the input data of the parent installation as well as from the export data of their sibling Sub-Installations. 
Execution of a Sub-Installation is started, when all its import data is available. 

The hierarchy of Blueprints can be arbitrarily deep.
