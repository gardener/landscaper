# Testing Root Installations

In this scenario we create two root installations. The second installation imports the exports of the first one.
Initially, the installations do not have the reconcile annotation, so that they will not be processed directly after
their creation.

The installations are used in test 
[rootinstallations/rootinstallations.go](../../../rootinstallations/rootinstallations.go).
Both installations are created, then only the first one is triggered with a reconcile annotation. 
When the first has finished, it should trigger the second. 
We check that both finish. We also check the exports, the deployed ConfigMaps, and thereby the processing order.
Finally, we delete the installations and check that they are gone.
(Currently we do not check the deletion order.)
