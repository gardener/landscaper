# Target Map Tests

This component contains several blueprints for the integration tests of TargetMaps.

## Loops over Target Maps

### Installation 1: Multiple DeployItems

Blueprint `blueprint-multiple-items` of installation 1 imports a TargetMap and a DataObject. The DataObject contains a 
map of data with the same keys as the TargetMap. The blueprint loops over the TargetMap. For each entry, it fetches the
corresponding entry of the DataObject, and creates a DeployItem. Each DeployItem creates a ConfigMap with the entry
of the DataObject as content.

### Installation 2

The blueprint of installation 2 forwards all imports to a subinstallation which does the same as installation 1.

### Installation 3

The blueprint of installation 3 forwards all imports to a subinstallation which does the same as installation 2.

```text
Installation targetmaps-3-root                        ->  blueprint-targetmap-ref-ref
└── Installation targetmap-ref                        ->  blueprint-targetmap-ref
    └── Installation inst-blueprint-multiple-items    ->  blueprint-multiple-items
        ├── DeployItem ...item-blue
        └── DeployItem ...item-red
```

### Installation 4: Multiple Subinstallations

Blueprint `blueprint-multiple-subinst` of installation 4 imports a TargetMap and a DataObject. The DataObject contains a
map of data with the same keys as the TargetMap. The blueprint loops over the entries of the DataObject. For each entry, 
it fetches the corresponding Target, and creates a sub-installation with blueprint `blueprint-single-item`. 
Each of these sub-installations creates a DeployItem, which deploys a ConfigMap with the data entry as content.

### Installation 5

The blueprint of installation 5 forwards all imports to a subinstallation which does the same as installation 4.

### Installation 6

The blueprint of installation 6 forwards all imports to a subinstallation which does the same as installation 5.

```text
Installation targetmaps-6-root                        ->  blueprint-targetmap-ref-ref
└── Installation targetmap-ref                        ->  blueprint-targetmap-ref
    └── Installation inst-blueprint-multiple-subinst  ->  blueprint-multiple-subinst
        ├── Installation single-item-blue             ->  blueprint-single-item
        │   └── DeployItem ...item-blue
        └── Installation single-item-red              ->  blueprint-single-item
            └── DeployItem ...item-red
```

## Composing Target Maps

### Installation 7

The root installation imports individual Targets, and passes them as TargetMap to a sub-installation with
blueprint "blueprint-multiple-subinst".

```text
Installation targetmaps-7-root                        ->  blueprint-composition
└── Installation multiple-subinst                     ->  blueprint-multiple-subinst
    ├──Installation single-item-blue                  ->  blueprint-single-item
    │   └── DeployItem ...item-blue
    └── Installation single-item-red                  ->  blueprint-single-item
        └── DeployItem ...item-red
```

### Installation 8

The blueprint of installation 8 forwards all imports to a subinstallation which does the same as installation 7.

## Composing Target Maps from Exports

### Installation 9

The root installation creates per entry in the imported configs DataObject a sub-installation with the blueprint "blueprint-exported".
Each of them exports one Target. Another sub-installation imports all these Targets as TargetMap.
Its blueprint "blueprint-multiple-subinst" creates one sub-installation per entry in the configs DataObject and corresponding Target.

```text
Installation targetmaps-9-root                        ->  blueprint-export-composition
├── Installation exporter-blue                        ->  blueprint-exporter
├── Installation exporter-red                         ->  blueprint-exporter
└── Installation multiple-subinst                     ->  blueprint-multiple-subinst
    ├── Installation single-item-blue                 ->  blueprint-single-item
    │   └── DeployItem ...item-blue
    └── Installation single-item-red                  ->  blueprint-single-item
        └── DeployItem ...item-red
```

### Installation 10

The blueprint of installation 10 forwards all imports to a subinstallation which does the same as installation 9.
