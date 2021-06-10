# Landscaper Cli Usage

- [Render Blueprints locally](#render-blueprints)


### Render Blueprints

During the execution of a blueprint with an installation, deployitems and subinstallation are created by the landscaper based on the imported values.

The generated resources can be locally tested by using the landsacper cli and its `render` command.

```shell script
landscaper-cli blueprints render [path to blueprint directory]

landscaper-cli blueprints render [path to blueprint directory] -f values.yaml -c component-descriptor.yaml
```

A component descriptor can be defined by using the `-c` flag that is a path to the component descriptor.

The command renders the resulting DeployItems(with the templating state) and the subinstallations and prints them to stdout.<br>
The output will print the rendered resources in the following structure
```shell script
--------------------------------------
-- state
--------------------------------------
state:
  <execution name>: ...

--------------------------------------
-- deployitems <deployitem name>
--------------------------------------
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: DeployItem
...

--------------------------------------
-- subinstllations <subinstallation name>
--------------------------------------
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
...
```

Alternatively, the rendered resources can be written to a directory by specifying `-w /path/to/output`.
The resources are written as files in the following directory structure to the given path.
```
/path/to/output
├── deployitems
│   └── mydeployitem
├── state
└── subinstallations
    └── mysubinstallation
```

#### Import Values

The imported values can be defined using value files and reference them via commandline flag `-f`.
```yaml
# /dev/values.yaml
imports:
  <import name>: <my value>
```
```shell script
landscaper-cli blueprints render -f /dev/values.yaml [path to blueprint directory]
```

__Example__
```
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Blueprint
imports:
- name: myFirstImport
  type: data
  schema:
    type: string
- name: mySecondImport
  type: target
  targetType: landscaper.gardener.cloud/kubernetes-cluster
```
```
imports:
  myFirstImport: "this is a import"
  mySecondImport: 
    metadata:
      name: my-target
      namespace: default
    spec:
       type: landscaper.gardener.cloud/kubernetes-cluster
       config:
         kubeconfig: |
            apiVersion: ....
```

