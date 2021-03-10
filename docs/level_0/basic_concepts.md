# Basic Concepts

This document describes the basic concepts, artifacts and their relationships of Landscaper. 

## DeployItem

The main objective of Landscaper is to set up large cloud environments with complex relationships consisting of many
small installation tasks like helm deployments, network setup etc. These elementary installation tasks are defined by 
DeployItems. There are different types of DeployItems for example for deploying helm charts, plain kubernetes manifests 
or terraform configurations.

A DeployItem can define import parameter for data needed for the installation task, e.g. the target cluster where 
a helm chart should be deployed. A DeployItem can also define output parameter for data it creates and which can be 
consumed by others.

## Blueprint

Several DeployItems can be collected in a Blueprint. In this sense, a Blueprint is a set of installation tasks. 

A Blueprint can define an interface for import data, required by its DeployItems. It can also define an export 
interface to expose output data of its DeployItems. Furthermore, the execution order of the DeployItems of a Blueprint 
could be specified. 

Blueprints are reusable collections of combined installation tasks which could be parameterized through their import 
specification/parameters. They describe system setups consisting of a set of installations task and their dependencies. 

## Blueprint Component and Component Descriptor

A component is a quite general term usually describing some IT artifact ranging from a small program to large
and complex systems. In the context of Landscaper we focus on components setting up some cloud environment 
described with a Blueprint. This could range from quite simple installations of one or two helm charts, or the
setup up of a complex system like a Garden landscape.

A Blueprint Component usually consists of a Blueprint, and the resources required by the DeployItems included in the 
Blueprint, like helm charts, docker images, json schema etc. All these required resources are described by a Component 
Descriptor as a yaml file. 

Component Descriptors are a complete description of all resources belonging to a component and could be used for
different tasks like security scanning or transport.

## Component Archive

Usually Component Descriptors are stored in Component Archive in an OCI registry. A Component Archive 
contains a Component Descriptor and optionally some other resources. Be aware that not all resources
of a component are stored in one Component Archive. The Component Descriptor can also just contain
references to resources located somewhere else, like a helm chart in some remote helm chart repository.

Blueprints are also stored in an OCI registry. They can be stored as a part of the Component Archive 
they belong to. Alternatively, they can be stored as a standalone OCI artifact referenced by the Component Descriptor of
a Component Archive.

More details about Component Descriptors and Component Descriptor Artifacts can be found 
[here](https://github.com/gardener/component-spec).

## Installation

Blueprints, Component Descriptors, and Component Archives do not trigger any installation process. A Blueprints is a reusable 
definition of the installation process of a particular cloud environment. An instance of such a Blueprint with particular
import data is called an Installation. Installations are kubernetes Custom Resources (CR) defined by Landscaper.
An installation references the Blueprint of a Component via the corresponding Component Archive.
It specifies the import data and how to handle the export data. 

An Installation is deployed to the cluster where Landscaper watches these CRs. If Landscaper recognizes an installation
with all input data available, it starts the setup specified in the Blueprint. This executes the installation tasks 
defined in the DeployItems.

## Aggregated Blueprint

Very complex systems require hundreds of installation steps resulting in Blueprints with many DeployItems. 
Landscaper allows defining subsystems each described by a separate Blueprint and to combine these in an aggregated 
Blueprint. An aggregated Blueprint references the Blueprints of the subsystems not directly but via Sub-Installations.
Sub-Installations are again Installations referencing a Blueprint, which describes the subsystem.

Assume an Installation referencing an aggregated Blueprint is deployed. When all input data of the Installation is 
available, all Sub-Installations are created. Sub-Installations get their input data from the parent installation as well
as from the export data of their sibling Installations. Execution of a Sub-Installation is started, when all its import
data is available. 

The hierarchy of installations/blueprints can be arbitrarily deep but in practice two levels should be sufficient.
