---
title: Component References
sidebar_position: 2
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
