# Targets

Targets are a specific type of import and contains additional information that is interpreted by deployers.
The concept of a Target is to define the environment where a deployer installs/deploys software.
This means that targets could contain additional information about that environment (e.g. that the target cluster is in a fenced environment and needs to be handled by another deployer instance).

The configuration structure of targets is defined by their type (currently the type is only for identification but later we plan to add some type registration with checks.)
