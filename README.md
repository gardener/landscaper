# landscaper
Development of Gardener Installation Operator

### Executors

There are 3 different types of executors:
- Container
- Script
- Templates


The deployers communicate with the landscaper through a dedicated DeployItem.
These DeployItems are the extensions of the landscaper which means that they execute the actual components.

Deployer act upon DeployItem CRDs that are created and updated by the landscaper.
By default the landscaper is deployed with 2 default Deployers: Container and Script.
Other Deployers can be used with the templates executor that templates such DeployItems.
