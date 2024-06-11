---
title: General Remarks about Sub Installations
sidebar_position: 1
---

# General Remarks about Sub Installations

During the following examples the concept of sub installations is introduced. Every installation can have a set of 
sub installations, each responsible to deploy different artifacts. An installation which is not the sub installation
of another installation is called a root installation. This allows you to divide large and complex deployments into 
smaller ones. It also provides you the possibility to reuse deployments (components with blueprints) in other
deployments.

Installations and sub installations can exchange data, e.g. you might have an installation consisting of two
sub installations. The first sub installation creates a Gardener shoot cluster and provides the access data
for the shoot cluster to the second sub installation. The second sub installations uses these data to deploy some 
application to the shoot cluster. 

Remark: DeployItems could not exchange data directly with other DeployItems. 

An installation with sub installations is processed as follows. When the processing of the root installation is 
triggered, the root installations creates the sub installations and provides them their import data. A sub installation
starts working if all its import data, including those from its sibling sub installations, is available. When all
sub installations have finished their work (including the executions of their sub installations), the root installation
fetches the export data of the sub installations and finishes its work.

Deleting a root installation triggers the deletion of all sub installation in the following order. A sub installation 
Sub-A is removed when all sub installations, which requires import data from Sub-A, were removed. 
