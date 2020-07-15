# DataObjects

### Own DataObject for every import and export key
```
imports:
- key: abc
  type: string
- key: xyz
  type: number
```
```
kind: DataObject
name: my-namspace.abc
type: string
data: xxx
generation: 5
---
kind: DataObject
name: my-namspace.xyz # inst1.inst2.inst3.xyz
type: string
data: xxx
generation: 5
```

*Advantages*:
- only dependent on dataobjects and their generation not on installations
- more possibilities for initial config bootstrap
- operator do not need to know the exporting component to look up the value

*Disadvantages*:
- a lot of dataobjects with overhead
  - need to copy all exported values to exporting namespaces
  - need to copy all imported values to importing namespaces
- refactor of existing dataobjects

### Aggregated export and import dataobjects
```
kind: Installation
status:
  importRef: 
     name: do-abc
  exportRef: 
     name: do-abc
  imports:
  - key: abc
    generation: x
    reference: 
      name: root1
```
```
kind: DataObject
name: my-namspace.abc
type: string
data: 
  abc: xxx
  xyz: xxx
```

*Advantages*:
- less dataobjects and overhead
- one time read for all imports from a component

*Disadvantages*:
- construct new dataobject for every copying of values (copying needs to be done for every import and export)
- read key from dataobject
- import check is based on siblings and parents