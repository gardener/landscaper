# Kustomize Deployments 

## Problem Description

Many potential customers deploy their k8s resources with kustomize, which is currently not supported by Landscaper.
To also support kustomize, it is possible to develop a particular kustomize deployer based on the kustomize
golang library, similar to the already existing helm deployer. Such an approach requires some effort not 
only during development but also during maintenance. Therefore, this proposal discusses how to use the 
[Flux kustomize controller](https://fluxcd.io/flux/components/kustomize/) in combination with the already existing
Landscaper manifest deployer. This approach will probably require much less resources and might also be a blueprint 
for further Landscaper/third party combinations to integrate further deployment technologies without much effort. 

## Solution Overview

The basic idea is, that a Landscaper customer creates Installations and Deploy Items which deploy Flux Kustomization
k8s custom resources (CRs) with the manifest deployer. A Flux installation, is watching these CRs and executes the 
kustomize deployment.

The approach to combine the Landscaper manifest deployer with some other tooling is a general solution to add further 
deployment technologies to the Landscaper, without implementing a special Landscaper deployer for each of them. 

## Detailed Analysis

### Combination of Execution Strategies

Both the Landscaper and Flux are tools to orchestrate the deployment of artefacts with different strategies.

The Landscaper executes the deployments via Installations and Deploy Items in a quite controlled way, triggered by
an operation annotation and in the order of the data flow between the different Deploy Items. After the deployment,
it stops and waits until the next processing is triggered by another operation annotation.

Flux allows different deployment strategies but its main focus is on regularly repeating the deployment according to
specified intervals. 

Using the Landscaper to deploy Flux kustomize CRs, results in a setup where the Installations and Deploy Items are
still controlled with the strict Landscaper strategy but even if a Deploy Item is finished on that level, its 
corresponding kustomize CR might reconcile periodically and possibly updates its data, including those used as exports
or readiness checks in the Deploy Items. 

The first question is, can this setup somehow break the Landscaper execution strategy resulting
in inconsistent states of the execution graph of Installations, Subinstallations and Deploy Items? The simple answer is
no. Though, Flux brings its own opinionated control strategy, this does not change the overall setup completely.
Already with a Helm deployment, it is possible to deploy some operator which modifies export data etc. even if
the corresponding Deploy Item is already finished. If the export data of a Deploy Item available and the readiness 
checks are successful, a Deploy Item is successfully finished and all subsequent changes at the data are ignored.

Nevertheless, the combination of the two explicit control strategies might result in quite complicated 
situations. The implications of the combination of the Landscaper and the Flux control strategies should be described
in detail. Furthermore, we should discuss particular solutions to combine both in our Guided Tour including:

- How to deploy k8s manifests with kustomize based completely on the Landscaper strategy. This approach should explain 
  how to prevent regular redeployments initiated by Flux. The target group are customers using kustomize but not Flux. 

- How to take advantage of the Flux way to regularly redeploy kustomize manifests without interfering with the 
  Landscaper controller. 

### Extend Exports, Readiness Checks and Deletion Groups of the Manifest Deployer

The manifest deployer deploys k8s manifests to a "customer" cluster. In the corresponding Deploy Item it is possible
to reference k8s resources on that cluster to specify exports data as well as readiness conditions. Furthermore,
it is possible to defined deletion groups with respect to the k8s resources on the "customer" cluster.

With the combination of the manifest deployer deploying a Flux Kustomize CR, this situation has changed a little bit.
The manifest deployer deploys the Kustomize CRs on a "customer" cluster, but the "real" data are subsequently deployed
by Flux to a second customer cluster, which might be different from the first one. Of course, it would be reasonable,
to allow the specification of exports, readiness checks and deletion groups not only with respect to the first but
also to the second "customer" cluster, which requires some extension to the manifest deployer.

### Extend the Landscaper Control Strategy

It might be interesting to investigate how to extend the Landscaper control strategy, such that it can react to 
data changes, e.g. produced by Flux reconciliations, even if the processing of a Deploy Item is already finished.

### Clarify Flux Setup in a LaaS Landscape

In a LaaS landscape it must be clarified where Flux controller are running and how they are set up.   

## Summary

This is a summary of the planned tasks:

- Extend documentation to clarify the combination of Flux and Landscaper to deploy kustomize project.
- Extend Guided Tour with different examples. At least one with full Landscaper control and one with full Flux control
  is required.
- Extend manifest deployer to reference data from a so-called second cluster to specify exports, readiness checks
  and deletion groups.
- Clarify and implement Flux setup for LaaS Landscapes
- (Optional) Extend the Landscaper control strategy to react consistently on late data changes, i.e. changes 
  produced even if a Deploy Item is already finished.









