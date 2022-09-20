# Referencing the Repository Context via a Context Resource

In most of our test scenarios, the Installation has specified its component like this, with the `repositoryContext`
"inline" in `componentDescriptor.ref`:

```yaml
  componentDescriptor:
    ref:
      componentName: github.com/gardener/landscaper/integration-tests/inline-base
      repositoryContext:
        baseUrl: eu.gcr.io/gardener-project/landscaper/integration-tests
        type: ociRegistry
      version: v0.1.0
```

In the present scenario, the repository context comes instead from a [Context](./context.yaml) custom resource. 
The Installation references the Context like this:

```yaml
  context: example-context

  componentDescriptor:
    ref:
      componentName: github.com/gardener/landscaper/integration-tests/inline-base
      version: v0.1.0
```

Besides this, the present scenario is equal to [installation-inline-base](../installation-inline-base).
