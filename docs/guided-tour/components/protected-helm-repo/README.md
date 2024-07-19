---
title: Chart from a Protected Helm Repository
sidebar_position: 2
---

# Chart from a Protected Helm Repository

In this example we explain how to deploy a chart from a protected Helm Repository.
The template for the deploy item references a Helm chart resource of the component descriptor:

```yaml
deployItems:
  - name: item
    config:
      chart:
        resourceRef: {{ getResourceKey `cd://resources/chart` }}
```

The resource in the component descriptor specifies the helm repository and chart:

```yaml
resources:
  - name: chart
    type: helmChart
    version: 1.0.0
    access:
      type: helm
      helmChart: ${helmChart}            # for example mariadb:12.2.7 
      helmRepository: ${helmRepository}  # for example https://charts.bitnami.com/bitnami
```

The format of this access type is defined in the [OCM Input and Access Types](https://ocm.software/docs/tutorials/input-and-access-types/#helm-1). 


We assume that the Helm repository is protected. The credentials to read the chart are provided in the 
[Context](installation/context.yaml.tpl) resource:

```yaml
configurations:
  helmChartRepoCredentials:
    auths:
      - url: <common prefix of the url of the index.yaml and chart>
        authHeader: <auth header>
```

Note that the auth header is used both: reading the index.yaml of the Helm repository, and reading the chart whose URL is 
in an entry of the index.yaml.
The URL prefix `configurations.helmChartRepoCredentials.auths[].url` must be chosen in such a way that both URLs
start with this prefix.
Alternatively, you can maintain two entries in the Context:

```yaml
configurations:
  helmChartRepoCredentials:
    auths:
      - url: <prefix of the url of the index.yaml>
        authHeader: <auth header>
      - url:  <prefix of the url of the chart>
        authHeader: <auth header>
```
