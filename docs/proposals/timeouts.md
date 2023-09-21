# Refactoring timeouts

This document describes the current state of the timeout implementation, its problems and a potential solution
how these might be resolved.

## Current state

### Which timeouts currently exists?

This is just a short list of timeouts which currently exists:

Globale:
- LandscaperConfiguration.DeployItemTimeouts.Pickup
- LandscaperConfiguration.DeployItemTimeouts.ProgressingDefault

DeployItem:
- DeployItem.DeployItemSpec.Timeout (set on LandscaperConfiguration.DeployItemTimeouts.ProgressingDefault if nothing
  is specified)

Helm Deployer
- Configuration.ExportConfiguration.DefaultTimeout
- DeployItem...ProviderConfiguration.DeleteTimeout

- For real helm deployer
    - ...HelmInstallConfiguration.Timeout
    - ...HelmUpgradeConfiguration.Timeout
    - ...HelmDeleteConfiguration.Timeout


Manifest Deployer
- Configuration.ExportConfiguration.DefaultTimeout
- DeployItem...ProviderConfiguration.DeleteTimeout


Readiness-Checks as part of the helm and manifest deployer provider config
- ProviderConfiguration...ReadinessCheckConfiguration.Timeout
- ProviderConfiguration...ReadinessCheckConfiguration.CustomReadinessCheckConfiguration.Timeout

### Current strategy

Currently, either the deployer runs into one of its different and very unintuitive timeouts and sets its state on failure
or the central timeout controller finds out that a DeployItem is processed longer than DeployItem.DeployItemSpec.Timeout
and this controller sets the DeployItem on failure.

### Problems with the current strategy

- If the deployer does not finish in time, the timeout controller sets a DeployItem on failure. This results in two problems:
  - If there is some error in the deployer execution, this is not stored in the DeployItem because of a write conflict.
  - Even if the timeout controller sets the DeployItem on failure, the deployer might continue processing on it, e.g. if it is
    current deploying the helm chart via helm 3 api. Retriggering a new reconcilation will not start working on that 
    DeployItem before the deployer has finished its work.

- Too many possible settings with sometimes strange semantice/implementations
  - Understanding the timeouts and their exact semantics is very complicated. For example Steampunk:
    - The set DeployItem.DeployItemSpec.Timeout=15m and ProviderConfiguration...ReadinessCheckConfiguration.Timeout=15min 
      for every DeployItem
    - This means that the DeployItem is set on failed after 15 minutes (by the timeout controller) but if the deployment 
      required already required 10 minutes should the deployer another 15 minutes for the readiness checks?
    - The is no timeout set for the helm chart deployment with helm 3. What does this mean? How long might the deployer
      hang in this step and thereby preventing another reconciliation even after the timeout controller has set the
      state on failure?
      - There are similar questions for the manifest and helm-manifest deployer. What is the timeout for the http client etc.?
  - There are some really strange or even inconsistent implementations e.g. 
    - Timeout in customreadinesscheck. Checks are executed one after the other in an undefined order. So it depends on 
      that order how much time was already spend before.
    - deleteTimeout is used in the context of manifest and helm manifest deployer when deleting orphaned resources during
      an upgrade. This setting has nothing to do with the deletion of a DeployItem. 

- Pickup timeout: If there is high load on the system, the pickup of particular DeployItems by a deployer is deferred 
  because the worker threads might be exhausted and therefore their state is set on failure by the timeout controller. 
  Just waiting some additional time would have been resulted in their proper processing. 


## Solution proposal

- Remove pickup timeout or set it on very large value:
  - The main reason for the pickup timeout was, to resolve the problem that if someone wrongly configures a DeployItem
    and no deployer would be responsible, that this DeployItem is finished. This use case is mainly relevant during
    development. Comparing this with the problem of failed DeployItems in productive systems under heavy load, seems to
    be a bad deal. Furthermore, for the development scenario, it is already possible to find out that a DeployItem
    is not picked up, by its empty status and such DeployItems could be stopped by the interrupt annotation. 
    Therefore, the removal of this timeout or setting it on a high value might be a good idea. 
  - If we want to be more defencive, we could add a pickup timeout to the DeployItem itself. Then the customer could 
    decide by itself if he wants this feature e.g. in his development scenario.

- Reduce number of processing timeout settings
    - One processing timeout on deploy item level
    - Optional delete timeout on deploy item level (if not set some default or use same as for progressing timeout)
    - Exclude the manifest and helm deployer from the timeout controller
      - When we have also found a similar solution for the container deployer we could perhaps completely remove the central
        processing timeout controller?
- Deployment/Upgrade/Deletion
  - Check the timeout between the main steps like templating, deployment, readiness checks, export generation. A
    particular step gets an allowed duration computed by (timeout - required duration of steps before). When the timout 
    is exceeded the DeployItem the processing is interrupted and set on failed 
  - Running requests like helm 3 deployments or manifest deployments are canceled if they require too much time
  - The helm 3 timeout is recomputed with respect to the remaining duration - no matter if it is set or not.

## Open Question

- When does the processing timeout start, when a DeployItem is triggered of when the deployer starts working on it?
  - Alternative one has the advantage, that a system under load where it might need some time until a deployer could
    pick up a DeployItem, is not running out of time. It seems also to be much easier for a customer to know how long a 
    deployment for a DeployItems needs, compared to how long a deployer requires to pick up a DeployItem.
  - In case of an error and an immediate retry of the reconciliation of the DeployItem, should the required time
    start again with 0 or include the required time of the erroneous reconciliations from before? Alternative 2 looks
    as the better alternative as it prevents infinite loops.
    
  