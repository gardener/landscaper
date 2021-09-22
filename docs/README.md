## Documentation Index

### Concepts
- [Glossary](./concepts/Glossary.md)
- [Installation-Blueprint relationship](./concepts/InstallationBlueprintRelationship.md)

### Getting Started
- [Install the landscaper cli](https://github.com/gardener/landscapercli/blob/master/docs/installation.md)
- [Install the landscaper](./gettingstarted/install-landscaper-controller.md)

### Usage
- [Blueprints](./usage/Blueprints.md)
- [Installations](./usage/Installations.md)
- [Template Executors](./usage/TemplateExecutors.md)
- [JSON Schema](./usage/JSONSchema.md)
- [Component Overwrites](./usage/ComponentOverwrites.md)
- [Conditional Imports](./usage/ConditionalImports.md)

### Deployers
- [Overview](./deployer)
- [Mock](./deployer/mock.md)
- [Helm](./deployer/helm.md)
- [Kubernetes Manifest](./deployer/manifest.md)
- [Container](./deployer/container.md)

### Tutorials
- [Local Setup with local OCI Registry](tutorials/00-local-setup.md)
- [Simple NGINX Ingress blueprint](tutorials/01-create-simple-blueprint.md)
- [Simple NGINX Ingress blueprint with Local Artifacts](tutorials/02-local-simple-blueprint.md)
- [HTTP Echo Server with Import from nginx ingress blueprint](tutorials/03-simple-import.md)
- [Aggregated Blueprint that includes the nginx-ingress and the echo-server](tutorials/04-aggregated-blueprint.md)
- [Use shared JSONSchemas in Blueprints](tutorials/05-external-jsonschema.md)

### Development
- [Local Setup](./development/local-setup.md)
- [Extend the API](./development/extend-the-api.md)
- [Creating Tutorial Resources](./development/tutorials.md)
- [Testing](./development/testing.md)
- [Deployer Library Extension Hooks](./development/dep-lib-extension-hooks.md)

### API Reference
- [Types](./technical/types.md)
- [Core](./api-reference/core.md)
- [Deployer Contract](./technical/deployer_contract.md)
- [Target Types](./technical/target_types.md)
- [Deployer Lifecycle Management](technical/deployer_lifecycle_management.md)
