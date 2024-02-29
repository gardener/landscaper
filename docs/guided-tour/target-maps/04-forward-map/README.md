---
title: Mapping Targets to Target Map in Subinstallation
sidebar_position: 4
---
# Mapping individual Targets to a Target Map in Subinstallation

The example defined [here](https://github.com/gardener/landscaper/blob/master/docs/guided-tour/target-maps/04-forward-map/component) deploys the same config maps on the target cluster as the others. The
difference is that it imports 3 targets as standard targets and in this
[Subinstallation](https://github.com/gardener/landscaper/blob/master/docs/guided-tour/target-maps/04-forward-map/component/blueprint/sub/subinst.yaml) it converts these into a target map.
