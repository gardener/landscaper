# API Reference

## Packages
- [landscaper.gardener.cloud/v1alpha1](#landscapergardenercloudv1alpha1)


## landscaper.gardener.cloud/v1alpha1

Package v1alpha1 is a version of the API.

Package v1alpha1 is the v1alpha1 version of the API.



#### AnyJSON



AnyJSON enhances the json.RawMessages with a dedicated openapi definition so that all it is correctly generated.



_Appears in:_
- [BlueprintStaticDataSource](#blueprintstaticdatasource)
- [Context](#context)
- [ContextConfiguration](#contextconfiguration)
- [DataObject](#dataobject)
- [Default](#default)
- [InlineBlueprint](#inlineblueprint)
- [InstallationSpec](#installationspec)
- [InstallationTemplate](#installationtemplate)
- [InstallationTemplateBlueprintDefinition](#installationtemplateblueprintdefinition)
- [StaticDataSource](#staticdatasource)
- [TargetSpec](#targetspec)
- [TargetTemplate](#targettemplate)
- [TemplateExecutor](#templateexecutor)



#### AutomaticReconcile



AutomaticReconcile allows to configure automatically repeated reconciliations.



_Appears in:_
- [InstallationSpec](#installationspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `succeededReconcile` _[SucceededReconcile](#succeededreconcile)_ | SucceededReconcile allows to configure automatically repeated reconciliations for succeeded installations.<br />If not set, no such automatically repeated reconciliations are triggered. |  |  |
| `failedReconcile` _[FailedReconcile](#failedreconcile)_ | FailedReconcile allows to configure automatically repeated reconciliations for failed installations.<br />If not set, no such automatically repeated reconciliations are triggered. |  |  |


#### AutomaticReconcileStatus



AutomaticReconcileStatus describes the status of automatically triggered reconciles.



_Appears in:_
- [InstallationStatus](#installationstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `generation` _integer_ | Generation describes the generation of the installation for which the status holds. |  |  |
| `numberOfReconciles` _integer_ | NumberOfReconciles is the number of automatic reconciles for the installation with the stored generation. |  | Format: int32 <br /> |
| `lastReconcileTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#time-v1-meta)_ | LastReconcileTime is the time of the last automatically triggered reconcile. |  |  |
| `onFailed` _boolean_ | OnFailed is true if the last automatically triggered reconcile was done for a failed installation. |  |  |




#### BlueprintDefinition



BlueprintDefinition defines the blueprint that should be used for the installation.



_Appears in:_
- [InstallationSpec](#installationspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `ref` _[RemoteBlueprintReference](#remoteblueprintreference)_ | Reference defines a remote reference to a blueprint |  |  |
| `inline` _[InlineBlueprint](#inlineblueprint)_ | Inline defines a inline yaml filesystem with a blueprint. |  |  |






#### ComponentDescriptorDefinition



ComponentDescriptorDefinition defines the component descriptor that should be used
for the installation



_Appears in:_
- [InstallationSpec](#installationspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `ref` _[ComponentDescriptorReference](#componentdescriptorreference)_ | ComponentDescriptorReference is the reference to a component descriptor |  |  |
| `inline` _[ComponentDescriptor](#componentdescriptor)_ | InlineDescriptorReference defines an inline component descriptor |  | Schemaless: {} <br />Type: object <br /> |




#### ComponentDescriptorReference



ComponentDescriptorReference is the reference to a component descriptor.
given an optional context.



_Appears in:_
- [ComponentDescriptorDefinition](#componentdescriptordefinition)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `repositoryContext` _[UnstructuredTypedObject](#unstructuredtypedobject)_ | RepositoryContext defines the context of the component repository to resolve blueprints. |  | Schemaless: {} <br />Type: object <br /> |
| `componentName` _string_ | ComponentName defines the unique of the component containing the resource. |  |  |
| `version` _string_ | Version defines the version of the component. |  |  |


#### ComponentVersionOverwrite



ComponentVersionOverwrite defines an overwrite for a specific component and/or version of a component.



_Appears in:_
- [ComponentVersionOverwriteList](#componentversionoverwritelist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `source` _[ComponentVersionOverwriteReference](#componentversionoverwritereference)_ | Source defines the component that should be replaced. |  |  |
| `substitution` _[ComponentVersionOverwriteReference](#componentversionoverwritereference)_ | Substitution defines the replacement target for the component or version. |  |  |


#### ComponentVersionOverwriteList

_Underlying type:_ _[ComponentVersionOverwrite](#componentversionoverwrite)_

ComponentVersionOverwriteList is a list of component overwrites.



_Appears in:_
- [ComponentVersionOverwrites](#componentversionoverwrites)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `source` _[ComponentVersionOverwriteReference](#componentversionoverwritereference)_ | Source defines the component that should be replaced. |  |  |
| `substitution` _[ComponentVersionOverwriteReference](#componentversionoverwritereference)_ | Substitution defines the replacement target for the component or version. |  |  |


#### ComponentVersionOverwriteReference



ComponentVersionOverwriteReference defines a component reference by



_Appears in:_
- [ComponentVersionOverwrite](#componentversionoverwrite)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `repositoryContext` _[UnstructuredTypedObject](#unstructuredtypedobject)_ | RepositoryContext defines the context of the component repository to resolve blueprints. |  | Schemaless: {} <br /> |
| `componentName` _string_ | ComponentName defines the unique of the component containing the resource. |  |  |
| `version` _string_ | Version defines the version of the component. |  |  |


#### ComponentVersionOverwrites



ComponentVersionOverwrites contain overwrites for specific (versions of) components.



_Appears in:_
- [ComponentVersionOverwritesList](#componentversionoverwriteslist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `overwrites` _[ComponentVersionOverwriteList](#componentversionoverwritelist)_ | Overwrites defines a list of component overwrites |  |  |




#### Condition



Condition holds the information about the state of a resource.



_Appears in:_
- [DeployItemStatus](#deployitemstatus)
- [ExecutionStatus](#executionstatus)
- [InstallationStatus](#installationstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _[ConditionType](#conditiontype)_ | DataType of the Shoot condition. |  |  |
| `lastTransitionTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#time-v1-meta)_ | Last time the condition transitioned from one status to another. |  |  |
| `lastUpdateTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#time-v1-meta)_ | Last time the condition was updated. |  |  |
| `reason` _string_ | The reason for the condition's last transition. |  |  |
| `message` _string_ | A human readable message indicating details about the transition. |  |  |
| `codes` _[ErrorCode](#errorcode) array_ | Well-defined error codes in case the condition reports a problem. |  |  |




#### ConditionType

_Underlying type:_ _string_

ConditionType is a string alias.



_Appears in:_
- [Condition](#condition)





#### Context



Context is a resource that contains shared information of installations.
This includes information about the repository context like the context itself or secrets to access the oci artifacts.
But it can also contain deployer specific config.



_Appears in:_
- [ContextList](#contextlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `repositoryContext` _[UnstructuredTypedObject](#unstructuredtypedobject)_ | RepositoryContext defines the context of the component repository to resolve blueprints. |  | Schemaless: {} <br /> |
| `useOCM` _boolean_ | UseOCM defines whether OCM is used to process installations that reference this context. |  |  |
| `registryPullSecrets` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#localobjectreference-v1-core) array_ | RegistryPullSecrets defines a list of registry credentials that are used to<br />pull blueprints, component descriptors and jsonschemas from the respective registry.<br />For more info see: https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/<br />Note that the type information is used to determine the secret key and the type of the secret. |  |  |
| `configurations` _object (keys:string, values:[AnyJSON](#anyjson))_ | Configurations contains arbitrary configuration information for dedicated purposes given by a string key.<br />The key should use a dns-like syntax to express the purpose and avoid conflicts. |  | Schemaless: {} <br />Type: object <br /> |
| `componentVersionOverwrites` _string_ | ComponentVersionOverwritesReference is a reference to a ComponentVersionOverwrites object<br />The overwrites object has to be in the same namespace as the context.<br />If the string is empty, no overwrites will be used. |  |  |
| `verificationSignatures` _object (keys:string, values:[VerificationSignature](#verificationsignature))_ | VerificationSignatures maps a signature name to the trusted verification information |  |  |


#### ContextConfiguration







_Appears in:_
- [Context](#context)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `repositoryContext` _[UnstructuredTypedObject](#unstructuredtypedobject)_ | RepositoryContext defines the context of the component repository to resolve blueprints. |  | Schemaless: {} <br /> |
| `useOCM` _boolean_ | UseOCM defines whether OCM is used to process installations that reference this context. |  |  |
| `registryPullSecrets` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#localobjectreference-v1-core) array_ | RegistryPullSecrets defines a list of registry credentials that are used to<br />pull blueprints, component descriptors and jsonschemas from the respective registry.<br />For more info see: https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/<br />Note that the type information is used to determine the secret key and the type of the secret. |  |  |
| `configurations` _object (keys:string, values:[AnyJSON](#anyjson))_ | Configurations contains arbitrary configuration information for dedicated purposes given by a string key.<br />The key should use a dns-like syntax to express the purpose and avoid conflicts. |  | Schemaless: {} <br />Type: object <br /> |
| `componentVersionOverwrites` _string_ | ComponentVersionOverwritesReference is a reference to a ComponentVersionOverwrites object<br />The overwrites object has to be in the same namespace as the context.<br />If the string is empty, no overwrites will be used. |  |  |
| `verificationSignatures` _object (keys:string, values:[VerificationSignature](#verificationsignature))_ | VerificationSignatures maps a signature name to the trusted verification information |  |  |




#### CriticalProblem



CriticalProblem contains information about one critical problem.



_Appears in:_
- [CriticalProblemsSpec](#criticalproblemsspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `podName` _string_ | PodName contains the name of the pod where the problem occurred |  |  |
| `creationTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#time-v1-meta)_ | CreationTime contains the timestamp when the problem occured |  |  |
| `description` _string_ | Description contains an error description |  |  |


#### CriticalProblems



CriticalProblems contains a list of critical landscaper problems.



_Appears in:_
- [CriticalProblemsList](#criticalproblemslist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[CriticalProblemsSpec](#criticalproblemsspec)_ | Spec contains the specification |  |  |




#### CriticalProblemsSpec



CriticalProblemsSpec contains the specification for a CriticalProblems object.



_Appears in:_
- [CriticalProblems](#criticalproblems)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `criticalProblem` _[CriticalProblem](#criticalproblem) array_ |  |  |  |




#### DataExport



DataExport is a data object export.



_Appears in:_
- [InstallationExports](#installationexports)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name the internal name of the imported/exported data. |  |  |
| `dataRef` _string_ | DataRef is the name of the in-cluster data object. |  |  |


#### DataImport



DataImport is a data object import.



_Appears in:_
- [InstallationImports](#installationimports)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name the internal name of the imported/exported data. |  |  |
| `dataRef` _string_ | DataRef is the name of the in-cluster data object.<br />The reference can also be a namespaces name. E.g. "default/mydataref" |  |  |
| `version` _string_ | Version specifies the imported data version.<br />defaults to "v1" |  |  |
| `secretRef` _[LocalSecretReference](#localsecretreference)_ | SecretRef defines a data reference from a secret.<br />This method is not allowed in installation templates. |  |  |
| `configMapRef` _[LocalConfigMapReference](#localconfigmapreference)_ | ConfigMapRef defines a data reference from a configmap.<br />This method is not allowed in installation templates. |  |  |


#### DataObject



DataObject are resources that can hold any kind json or yaml data.



_Appears in:_
- [DataObjectList](#dataobjectlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `data` _[AnyJSON](#anyjson)_ | Data contains the data of the object as string. |  | Schemaless: {} <br /> |






#### Default



Default defines a default value (future idea: also reference?).



_Appears in:_
- [ImportDefinition](#importdefinition)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `value` _[AnyJSON](#anyjson)_ |  |  |  |


#### DependentToTrigger







_Appears in:_
- [InstallationStatus](#installationstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the name of the dependent installation |  |  |


#### DeployItem



DeployItem defines a resource that should be processed by a external deployer



_Appears in:_
- [DeployItemList](#deployitemlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[DeployItemSpec](#deployitemspec)_ |  |  |  |


#### DeployItemCache



DeployItemCache contains the existing deploy items



_Appears in:_
- [ExecutionStatus](#executionstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `activeDIs` _[DiNamePair](#dinamepair) array_ |  |  |  |
| `orphanedDIs` _string array_ |  |  |  |




#### DeployItemPhase

_Underlying type:_ _string_





_Appears in:_
- [DeployItemStatus](#deployitemstatus)



#### DeployItemSpec



DeployItemSpec contains the definition of a deploy item.



_Appears in:_
- [DeployItem](#deployitem)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _[DeployItemType](#deployitemtype)_ | Type is the type of the deployer that should handle the item. |  |  |
| `target` _[ObjectReference](#objectreference)_ | Target specifies an optional target of the deploy item.<br />In most cases it contains the secrets to access a evironment.<br />It is also used by the deployers to determine the ownernship. |  |  |
| `context` _string_ | Context defines the current context of the deployitem. |  |  |
| `config` _[RawExtension](#rawextension)_ | Configuration contains the deployer type specific configuration. |  | EmbeddedResource: {} <br /> |
| `timeout` _[Duration](#duration)_ | Timeout specifies how long the deployer may take to apply the deploy item.<br />When the time is exceeded, the deploy item fails.<br />Value has to be parsable by time.ParseDuration (or 'none' to deactivate the timeout).<br />Defaults to ten minutes if not specified. |  | Type: string <br /> |
| `updateOnChangeOnly` _boolean_ | UpdateOnChangeOnly specifies if redeployment is executed only if the specification of the deploy item has changed. |  |  |
| `onDelete` _[OnDeleteConfig](#ondeleteconfig)_ | OnDelete specifies particular setting when deleting a deploy item |  |  |




#### DeployItemTemplate



DeployItemTemplate defines a execution element that is translated into a deploy item.



_Appears in:_
- [DeployItemTemplateList](#deployitemtemplatelist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the unique name of the execution. |  |  |
| `type` _[DeployItemType](#deployitemtype)_ | DataType is the DeployItem type of the execution. |  |  |
| `target` _[ObjectReference](#objectreference)_ | Target is the object reference to the target that the deploy item should deploy to. |  |  |
| `labels` _object (keys:string, values:string)_ | Labels is the map of labels to be added to the deploy item. |  |  |
| `config` _[RawExtension](#rawextension)_ | ProviderConfiguration contains the type specific configuration for the execution. |  | EmbeddedResource: {} <br /> |
| `dependsOn` _string array_ | DependsOn lists deploy items that need to be executed before this one |  |  |
| `timeout` _[Duration](#duration)_ | Timeout specifies how long the deployer may take to apply the deploy item.<br />When the time is exceeded, the deploy item fails.<br />Value has to be parsable by time.ParseDuration (or 'none' to deactivate the timeout).<br />Defaults to ten minutes if not specified. |  | Type: string <br /> |
| `updateOnChangeOnly` _boolean_ | UpdateOnChangeOnly specifies if redeployment is executed only if the specification of the deploy item has changed. |  |  |
| `onDelete` _[OnDeleteConfig](#ondeleteconfig)_ | OnDelete specifies particular setting when deleting a deploy item |  |  |


#### DeployItemTemplateList

_Underlying type:_ _[DeployItemTemplate](#deployitemtemplate)_

DeployItemTemplateList is a list of deploy item templates



_Appears in:_
- [ExecutionSpec](#executionspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the unique name of the execution. |  |  |
| `type` _[DeployItemType](#deployitemtype)_ | DataType is the DeployItem type of the execution. |  |  |
| `target` _[ObjectReference](#objectreference)_ | Target is the object reference to the target that the deploy item should deploy to. |  |  |
| `labels` _object (keys:string, values:string)_ | Labels is the map of labels to be added to the deploy item. |  |  |
| `config` _[RawExtension](#rawextension)_ | ProviderConfiguration contains the type specific configuration for the execution. |  | EmbeddedResource: {} <br /> |
| `dependsOn` _string array_ | DependsOn lists deploy items that need to be executed before this one |  |  |
| `timeout` _[Duration](#duration)_ | Timeout specifies how long the deployer may take to apply the deploy item.<br />When the time is exceeded, the deploy item fails.<br />Value has to be parsable by time.ParseDuration (or 'none' to deactivate the timeout).<br />Defaults to ten minutes if not specified. |  | Type: string <br /> |
| `updateOnChangeOnly` _boolean_ | UpdateOnChangeOnly specifies if redeployment is executed only if the specification of the deploy item has changed. |  |  |
| `onDelete` _[OnDeleteConfig](#ondeleteconfig)_ | OnDelete specifies particular setting when deleting a deploy item |  |  |


#### DeployItemType

_Underlying type:_ _string_

DeployItemType defines the type of the deploy item



_Appears in:_
- [DeployItemSpec](#deployitemspec)
- [DeployItemTemplate](#deployitemtemplate)



#### DeployerInformation



DeployerInformation holds additional information about the deployer that
has reconciled or is reconciling the deploy item.



_Appears in:_
- [DeployItemStatus](#deployitemstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `identity` _string_ | Identity describes the unique identity of the deployer. |  |  |
| `name` _string_ | Name is the name of the deployer. |  |  |
| `version` _string_ | Version is the version of the deployer. |  |  |


#### DiNamePair



DiNamePair contains the spec name and the real name of a deploy item



_Appears in:_
- [DeployItemCache](#deployitemcache)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `specName` _string_ |  |  |  |
| `objectName` _string_ |  |  |  |


#### Duration



Duration is a wrapper for time.Duration that implements JSON marshalling and openapi scheme.

_Validation:_
- Type: string

_Appears in:_
- [DeployItemSpec](#deployitemspec)
- [DeployItemTemplate](#deployitemtemplate)
- [FailedReconcile](#failedreconcile)
- [SucceededReconcile](#succeededreconcile)



#### Error



Error holds information about an error that occurred.



_Appears in:_
- [DeployItemStatus](#deployitemstatus)
- [ExecutionStatus](#executionstatus)
- [InstallationStatus](#installationstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `operation` _string_ | Operation describes the operator where the error occurred. |  |  |
| `lastTransitionTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#time-v1-meta)_ | Last time the condition transitioned from one status to another. |  |  |
| `lastUpdateTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#time-v1-meta)_ | Last time the condition was updated. |  |  |
| `reason` _string_ | The reason for the condition's last transition. |  |  |
| `message` _string_ | A human readable message indicating details about the transition. |  |  |
| `codes` _[ErrorCode](#errorcode) array_ | Well-defined error codes in case the condition reports a problem. |  |  |


#### ErrorCode

_Underlying type:_ _string_

ErrorCode is a string alias.



_Appears in:_
- [Condition](#condition)
- [Error](#error)



#### Execution



Execution contains the configuration of a execution and deploy item



_Appears in:_
- [ExecutionList](#executionlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ExecutionSpec](#executionspec)_ | Spec defines a execution and its items |  |  |




#### ExecutionPhase

_Underlying type:_ _string_





_Appears in:_
- [ExecutionStatus](#executionstatus)



#### ExecutionSpec



ExecutionSpec defines a execution plan.



_Appears in:_
- [Execution](#execution)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `context` _string_ | Context defines the current context of the execution. |  |  |
| `deployItems` _[DeployItemTemplateList](#deployitemtemplatelist)_ | DeployItems defines all execution items that need to be scheduled. |  |  |
| `deployItemsCompressed` _integer array_ | DeployItemsCompressed as zipped byte array |  |  |




#### ExportDefinition



ExportDefinition defines a exported value



_Appears in:_
- [ExportDefinitionList](#exportdefinitionlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name defines the field name to search for the value and map to exports.<br />Ref: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#selecting-fields |  |  |
| `schema` _[JSONSchemaDefinition](#jsonschemadefinition)_ | Schema defines the imported value as jsonschema. |  |  |
| `targetType` _string_ | TargetType defines the type of the imported target. |  |  |
| `type` _[ExportType](#exporttype)_ | Type specifies which kind of object is being exported.<br />This field should be set and will likely be mandatory in future. |  |  |


#### ExportDefinitionList

_Underlying type:_ _[ExportDefinition](#exportdefinition)_

ExportDefinitionList defines a list of export definitions.



_Appears in:_
- [Blueprint](#blueprint)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name defines the field name to search for the value and map to exports.<br />Ref: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#selecting-fields |  |  |
| `schema` _[JSONSchemaDefinition](#jsonschemadefinition)_ | Schema defines the imported value as jsonschema. |  |  |
| `targetType` _string_ | TargetType defines the type of the imported target. |  |  |
| `type` _[ExportType](#exporttype)_ | Type specifies which kind of object is being exported.<br />This field should be set and will likely be mandatory in future. |  |  |


#### ExportType

_Underlying type:_ _string_

ExportType is a string alias



_Appears in:_
- [ExportDefinition](#exportdefinition)



#### FailedReconcile



FailedReconcile allows to configure automatically repeated reconciliations for failed installations



_Appears in:_
- [AutomaticReconcile](#automaticreconcile)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `numberOfReconciles` _integer_ | NumberOfReconciles specifies the maximal number of automatically repeated reconciliations. If not set, no upper limit exists. |  | Format: int32 <br /> |
| `interval` _[Duration](#duration)_ | Interval specifies the interval between two subsequent repeated reconciliations. If not set, a default of 5 minutes is used. |  | Type: string <br /> |


#### FieldValueDefinition



FieldValueDefinition defines a im- or exported field.
Either schema or target type have to be defined



_Appears in:_
- [ExportDefinition](#exportdefinition)
- [ImportDefinition](#importdefinition)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name defines the field name to search for the value and map to exports.<br />Ref: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#selecting-fields |  |  |
| `schema` _[JSONSchemaDefinition](#jsonschemadefinition)_ | Schema defines the imported value as jsonschema. |  |  |
| `targetType` _string_ | TargetType defines the type of the imported target. |  |  |


#### ImportDefinition



ImportDefinition defines a imported value



_Appears in:_
- [ImportDefinitionList](#importdefinitionlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name defines the field name to search for the value and map to exports.<br />Ref: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#selecting-fields |  |  |
| `schema` _[JSONSchemaDefinition](#jsonschemadefinition)_ | Schema defines the imported value as jsonschema. |  |  |
| `targetType` _string_ | TargetType defines the type of the imported target. |  |  |
| `type` _[ImportType](#importtype)_ | Type specifies which kind of object is being imported.<br />This field should be set and will likely be mandatory in future. |  |  |
| `required` _boolean_ | Required specifies whether the import is required for the component to run.<br />Defaults to true. |  |  |
| `default` _[Default](#default)_ | Default sets a default value for the current import that is used if the key is not set. |  |  |
| `imports` _[ImportDefinitionList](#importdefinitionlist)_ | ConditionalImports are Imports that are only valid if this imports is satisfied.<br />Does only make sense for optional imports. |  |  |


#### ImportDefinitionList

_Underlying type:_ _[ImportDefinition](#importdefinition)_

ImportDefinitionList defines a list of import defiinitions.



_Appears in:_
- [Blueprint](#blueprint)
- [ImportDefinition](#importdefinition)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name defines the field name to search for the value and map to exports.<br />Ref: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#selecting-fields |  |  |
| `schema` _[JSONSchemaDefinition](#jsonschemadefinition)_ | Schema defines the imported value as jsonschema. |  |  |
| `targetType` _string_ | TargetType defines the type of the imported target. |  |  |
| `type` _[ImportType](#importtype)_ | Type specifies which kind of object is being imported.<br />This field should be set and will likely be mandatory in future. |  |  |
| `required` _boolean_ | Required specifies whether the import is required for the component to run.<br />Defaults to true. |  |  |
| `default` _[Default](#default)_ | Default sets a default value for the current import that is used if the key is not set. |  |  |
| `imports` _[ImportDefinitionList](#importdefinitionlist)_ | ConditionalImports are Imports that are only valid if this imports is satisfied.<br />Does only make sense for optional imports. |  |  |


#### ImportType

_Underlying type:_ _string_

ImportType is a string alias



_Appears in:_
- [ImportDefinition](#importdefinition)



#### InlineBlueprint



InlineBlueprint defines a inline blueprint with component descriptor and
filesystem.



_Appears in:_
- [BlueprintDefinition](#blueprintdefinition)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `filesystem` _[AnyJSON](#anyjson)_ | Filesystem defines a inline yaml filesystem with a blueprint. |  | Schemaless: {} <br /> |


#### Installation



Installation contains the configuration of a component



_Appears in:_
- [InstallationList](#installationlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[InstallationSpec](#installationspec)_ | Spec contains the specification for a installation. |  |  |


#### InstallationExports



InstallationExports defines exports of data objects and targets.



_Appears in:_
- [InstallationSpec](#installationspec)
- [InstallationTemplate](#installationtemplate)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `data` _[DataExport](#dataexport) array_ | Data defines all data object exports. |  |  |
| `targets` _[TargetExport](#targetexport) array_ | Targets defines all target exports. |  |  |


#### InstallationImports



InstallationImports defines import of data objects and targets.



_Appears in:_
- [InstallationSpec](#installationspec)
- [InstallationTemplate](#installationtemplate)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `data` _[DataImport](#dataimport) array_ | Data defines all data object imports. |  |  |
| `targets` _[TargetImport](#targetimport) array_ | Targets defines all target imports. |  |  |




#### InstallationPhase

_Underlying type:_ _string_





_Appears in:_
- [InstallationStatus](#installationstatus)



#### InstallationSpec



InstallationSpec defines a component installation.



_Appears in:_
- [Installation](#installation)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `context` _string_ | Context defines the current context of the installation. |  |  |
| `verification` _[Verification](#verification)_ | Verification defines the necessary data to verify the signature of the refered component |  |  |
| `componentDescriptor` _[ComponentDescriptorDefinition](#componentdescriptordefinition)_ | ComponentDescriptor is a reference to the installation's component descriptor |  |  |
| `blueprint` _[BlueprintDefinition](#blueprintdefinition)_ | Blueprint is the resolved reference to the definition. |  |  |
| `imports` _[InstallationImports](#installationimports)_ | Imports define the imported data objects and targets. |  |  |
| `importDataMappings` _object (keys:string, values:[AnyJSON](#anyjson))_ | ImportDataMappings contains a template for restructuring imports.<br />It is expected to contain a key for every blueprint-defined data import.<br />Missing keys will be defaulted to their respective data import.<br />Example: namespace: (( installation.imports.namespace )) |  | Schemaless: {} <br />Type: object <br /> |
| `exports` _[InstallationExports](#installationexports)_ | Exports define the exported data objects and targets. |  |  |
| `exportDataMappings` _object (keys:string, values:[AnyJSON](#anyjson))_ | ExportDataMappings contains a template for restructuring exports.<br />It is expected to contain a key for every blueprint-defined data export.<br />Missing keys will be defaulted to their respective data export.<br />Example: namespace: (( blueprint.exports.namespace )) |  | Schemaless: {} <br />Type: object <br /> |
| `automaticReconcile` _[AutomaticReconcile](#automaticreconcile)_ | AutomaticReconcile allows to configure automatically repeated reconciliations. |  |  |
| `optimization` _[Optimization](#optimization)_ | Optimization contains settings to improve execution performance. |  |  |




#### InstallationTemplate



InstallationTemplate defines a subinstallation in a blueprint.



_Appears in:_
- [InstallationTemplateList](#installationtemplatelist)
- [SubinstallationTemplate](#subinstallationtemplate)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the unique name of the step |  |  |
| `blueprint` _[InstallationTemplateBlueprintDefinition](#installationtemplateblueprintdefinition)_ | Reference defines a reference to a Blueprint.<br />The blueprint can reside in an OCI or other supported location. |  |  |
| `imports` _[InstallationImports](#installationimports)_ | Imports define the imported data objects and targets. |  |  |
| `importDataMappings` _object (keys:string, values:[AnyJSON](#anyjson))_ | ImportDataMappings contains a template for restructuring imports.<br />It is expected to contain a key for every blueprint-defined data import.<br />Missing keys will be defaulted to their respective data import.<br />Example: namespace: (( installation.imports.namespace )) |  | Schemaless: {} <br />Type: object <br /> |
| `exports` _[InstallationExports](#installationexports)_ | Exports define the exported data objects and targets. |  |  |
| `exportDataMappings` _object (keys:string, values:[AnyJSON](#anyjson))_ | ExportDataMappings contains a template for restructuring exports.<br />It is expected to contain a key for every blueprint-defined data export.<br />Missing keys will be defaulted to their respective data export.<br />Example: namespace: (( blueprint.exports.namespace )) |  | Schemaless: {} <br />Type: object <br /> |
| `optimization` _[Optimization](#optimization)_ | Optimization contains settings to improve execution performance. |  |  |


#### InstallationTemplateBlueprintDefinition



InstallationTemplateBlueprintDefinition contains either a reference to a blueprint or an inline definition.



_Appears in:_
- [InstallationTemplate](#installationtemplate)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `ref` _string_ | Ref is a reference to a blueprint.<br />Only blueprints that are defined by the component descriptor of the current blueprint can be referenced here.<br />Example: cd://componentReference/dns/resources/blueprint |  |  |
| `filesystem` _[AnyJSON](#anyjson)_ | Filesystem defines a virtual filesystem with all files needed for a blueprint.<br />The filesystem must be a YAML filesystem. |  | Schemaless: {} <br /> |




#### JSONSchemaDefinition



JSONSchemaDefinition defines a jsonschema.



_Appears in:_
- [Blueprint](#blueprint)
- [ExportDefinition](#exportdefinition)
- [FieldValueDefinition](#fieldvaluedefinition)
- [ImportDefinition](#importdefinition)



#### LocalConfigMapReference

_Underlying type:_ _[struct{Name string "json:\"name\""; Key string "json:\"key\""}](#struct{name-string-"json:\"name\"";-key-string-"json:\"key\""})_

LocalConfigMapReference is a reference to data in a configmap.



_Appears in:_
- [DataImport](#dataimport)



#### LocalSecretReference

_Underlying type:_ _[struct{Name string "json:\"name\""; Key string "json:\"key\""}](#struct{name-string-"json:\"name\"";-key-string-"json:\"key\""})_

LocalSecretReference is a reference to data in a secret.



_Appears in:_
- [DataImport](#dataimport)
- [TargetSpec](#targetspec)
- [TargetSyncSpec](#targetsyncspec)
- [TargetTemplate](#targettemplate)



#### LsHealthCheck



LsHealthCheck is a resource containing information about problems with the landscaper installation



_Appears in:_
- [LsHealthCheckList](#lshealthchecklist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `lastUpdateTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#time-v1-meta)_ | LastUpdateTime contains last time the check was updated. |  |  |
| `description` _string_ | Description contains description of the problem(s) |  |  |








#### ObjectReference



ObjectReference is the reference to a kubernetes object.



_Appears in:_
- [ConfigMapReference](#configmapreference)
- [DeployItemSpec](#deployitemspec)
- [DeployItemStatus](#deployitemstatus)
- [DeployItemTemplate](#deployitemtemplate)
- [ExecutionStatus](#executionstatus)
- [InstallationStatus](#installationstatus)
- [NamedObjectReference](#namedobjectreference)
- [SecretReference](#secretreference)
- [TargetSelector](#targetselector)
- [TypedObjectReference](#typedobjectreference)
- [VersionedObjectReference](#versionedobjectreference)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the name of the kubernetes object. |  |  |
| `namespace` _string_ | Namespace is the namespace of kubernetes object. |  |  |


#### OnDeleteConfig



OnDeleteConfig specifies particular setting when deleting a deploy item



_Appears in:_
- [DeployItemSpec](#deployitemspec)
- [DeployItemTemplate](#deployitemtemplate)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `skipUninstallIfClusterRemoved` _boolean_ | SkipUninstallIfClusterRemoved specifies that uninstall is skipped if the target cluster is already deleted.<br />Works only in the context of an existing target sync object which is used to check the Garden project with<br />the shoot cluster resources |  |  |




#### Optimization



Optimization contains settings to improve execution preformance



_Appears in:_
- [InstallationSpec](#installationspec)
- [InstallationTemplate](#installationtemplate)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `hasNoSiblingImports` _boolean_ | set this on true if the installation does not import data from its siblings or has no siblings at all |  |  |
| `hasNoSiblingExports` _boolean_ | set this on true if the installation does not export data to its siblings or has no siblings at all |  |  |


#### RemoteBlueprintReference



RemoteBlueprintReference describes a reference to a blueprint defined by a component descriptor.



_Appears in:_
- [BlueprintDefinition](#blueprintdefinition)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `resourceName` _string_ | ResourceName is the name of the blueprint as defined by a component descriptor. |  |  |


#### Requirement



Requirement contains values, a key, and an operator that relates the key and values.
The zero value of Requirement is invalid.
Requirement implements both set based match and exact match
Requirement should be initialized via NewRequirement constructor for creating a valid Requirement.



_Appears in:_
- [TargetSelector](#targetselector)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `key` _string_ |  |  |  |
| `operator` _[Operator](#operator)_ |  |  |  |
| `values` _string array_ | In huge majority of cases we have at most one value here.<br />It is generally faster to operate on a single-element slice<br />than on a single-element map, so we have a slice here. |  |  |




#### ResourceReference



ResourceReference defines the reference to a resource defined in a component descriptor.



_Appears in:_
- [VersionedResourceReference](#versionedresourcereference)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `componentName` _string_ | ComponentName defines the unique of the component containing the resource. |  |  |
| `resourceName` _string_ | ResourceName defines the name of the resource. |  |  |


#### SecretLabelSelectorRef



SecretLabelSelectorRef selects secrets with the given label and key.



_Appears in:_
- [StaticDataValueFrom](#staticdatavaluefrom)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `selector` _object (keys:string, values:string)_ | Selector is a map of labels to select specific secrets. |  |  |
| `key` _string_ | The key of the secret to select from.  Must be a valid secret key. |  |  |


#### SecretReference



SecretReference is reference to data in a secret.
The secret can also be in a different namespace.



_Appears in:_
- [VerificationSignature](#verificationsignature)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the name of the kubernetes object. |  |  |
| `namespace` _string_ | Namespace is the namespace of kubernetes object. |  |  |
| `key` _string_ | Key is the name of the key in the secret that holds the data. |  |  |




#### StaticDataValueFrom



StaticDataValueFrom defines a static data that is read from a external resource.



_Appears in:_
- [BlueprintStaticDataSource](#blueprintstaticdatasource)
- [StaticDataSource](#staticdatasource)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretKeyRef` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#secretkeyselector-v1-core)_ | Selects a key of a secret in the installations's namespace |  |  |
| `secretLabelSelector` _[SecretLabelSelectorRef](#secretlabelselectorref)_ | Selects a key from multiple secrets in the installations's namespace<br />that matches the given labels. |  |  |


#### SubInstCache



SubInstCache contains the existing sub installations



_Appears in:_
- [InstallationStatus](#installationstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `activeSubs` _[SubNamePair](#subnamepair) array_ |  |  |  |
| `orphanedSubs` _string array_ |  |  |  |


#### SubNamePair



DiNamePair contains the spec name and the real name of a deploy item



_Appears in:_
- [SubInstCache](#subinstcache)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `specName` _string_ |  |  |  |
| `objectName` _string_ |  |  |  |


#### SubinstallationTemplate



SubinstallationTemplate defines a subinstallation template.



_Appears in:_
- [SubinstallationTemplateList](#subinstallationtemplatelist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `file` _string_ | File references a subinstallation template stored in another file. |  |  |


#### SubinstallationTemplateList

_Underlying type:_ _[SubinstallationTemplate](#subinstallationtemplate)_

SubinstallationTemplateList is a list of installation templates



_Appears in:_
- [Blueprint](#blueprint)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `file` _string_ | File references a subinstallation template stored in another file. |  |  |


#### SucceededReconcile



SucceededReconcile allows to configure automatically repeated reconciliations for succeeded installations



_Appears in:_
- [AutomaticReconcile](#automaticreconcile)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `interval` _[Duration](#duration)_ | Interval specifies the interval between two subsequent repeated reconciliations. If not set, a default of<br />24 hours is used. |  | Type: string <br /> |


#### SyncObject



The SyncObject helps to sync access to deploy items.



_Appears in:_
- [SyncObjectList](#syncobjectlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[SyncObjectSpec](#syncobjectspec)_ | Spec contains the specification |  |  |




#### SyncObjectSpec



SyncObjectSpec contains the specification.



_Appears in:_
- [SyncObject](#syncobject)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `podName` _string_ | PodName describes the name of the pod of the responsible deployer |  |  |
| `kind` _string_ | Kind describes the kind of object that is being locked by this SyncObject |  |  |
| `name` _string_ | Name is the name of the object that is being locked by this SyncObject |  |  |
| `lastUpdateTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#time-v1-meta)_ | LastUpdateTime contains last time the object was updated. |  |  |
| `prefix` _string_ | Prefix is the prefix of the name of the object. |  |  |




#### Target



Target defines a specific data object that defines target environment.
Every deploy item can have a target which is used by the deployer to install the specific application.



_Appears in:_
- [ResolvedTarget](#resolvedtarget)
- [TargetList](#targetlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[TargetSpec](#targetspec)_ |  |  |  |


#### TargetExport



TargetExport is a single target export.



_Appears in:_
- [InstallationExports](#installationexports)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name the internal name of the exported target. |  |  |
| `target` _string_ | Target is the name of the in-cluster target object. |  |  |


#### TargetImport



TargetImport is either a single target or a target list import.



_Appears in:_
- [InstallationImports](#installationimports)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name the internal name of the imported target. |  |  |
| `target` _string_ | Target is the name of the in-cluster target object.<br />Exactly one of Target, Targets, and TargetListReference has to be specified. |  |  |
| `targets` _string array_ | Targets is a list of in-cluster target objects.<br />Exactly one of Target, Targets, and TargetListReference has to be specified. |  |  |
| `targetListRef` _string_ | TargetListReference can (only) be used to import a targetlist that has been imported by the parent installation.<br />Exactly one of Target, Targets, and TargetListReference has to be specified. |  |  |
| `targetMap` _object (keys:string, values:string)_ |  |  |  |
| `targetMapRef` _string_ |  |  |  |






#### TargetSpec



TargetSpec contains the definition of a target.



_Appears in:_
- [Target](#target)
- [TargetTemplate](#targettemplate)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _[TargetType](#targettype)_ | Type is the type of the target that defines its data structure.<br />The actual schema may be defined by a target type crd in the future. |  |  |
| `config` _[AnyJSON](#anyjson)_ | Configuration contains the target type specific configuration.<br />Exactly one of the fields Configuration and SecretRef must be set |  | Schemaless: {} <br /> |
| `secretRef` _[LocalSecretReference](#localsecretreference)_ | Reference to a secret containing the target type specific configuration.<br />Exactly one of the fields Configuration and SecretRef must be set |  |  |


#### TargetSync



The TargetSync is created targets from secrets.



_Appears in:_
- [TargetSyncList](#targetsynclist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[TargetSyncSpec](#targetsyncspec)_ | Spec contains the specification |  |  |




#### TargetSyncSpec



TargetSyncSpec contains the specification for a TargetSync.



_Appears in:_
- [TargetSync](#targetsync)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `sourceNamespace` _string_ | SourceNamespace describes the namespace from where the secrets should be synced |  |  |
| `secretRef` _[LocalSecretReference](#localsecretreference)_ | SecretRef references the secret that contains the kubeconfig to the namespace of the secrets to be synced. |  |  |
| `createTargetToSource` _boolean_ | CreateTargetToSource specifies if set on true, that also a target is created, which references the secret in SecretRef |  |  |
| `targetToSourceName` _string_ | TargetToSourceName is the name of the target referencing the secret defined in SecretRef if CreateTargetToSource<br />is set on true. If TargetToSourceName is empty SourceNamespace is used instead. |  |  |
| `secretNameExpression` _string_ | SecretNameExpression defines the names of the secrets which should be synced via a regular expression according<br />to https://github.com/google/re2/wiki/Syntax with the extension that * is also a valid expression and matches<br />all names.<br />if not set no secrets are synced |  |  |
| `shootNameExpression` _string_ | ShootNameExpression defines the names of shoot clusters for which targets with short living access data<br />to the shoots are created via a regular expression according to https://github.com/google/re2/wiki/Syntax with<br />the extension that * is also a valid expression and matches all names.<br />if not set no targets for the shoots are created |  |  |
| `tokenRotation` _[TokenRotation](#tokenrotation)_ | TokenRotation defines the data to perform an automatic rotation of the token to access the source cluster with the<br />secrets to sync. The token expires after 90 days and will be rotated every 60 days. |  |  |






#### TargetType

_Underlying type:_ _string_

TargetType defines the type of the target.



_Appears in:_
- [TargetSpec](#targetspec)
- [TargetTemplate](#targettemplate)



#### TemplateExecutor



TemplateExecutor describes a templating mechanism and configuration.



_Appears in:_
- [Blueprint](#blueprint)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the unique name of the template |  |  |
| `type` _[TemplateType](#templatetype)_ | Type describes the templating mechanism. |  |  |
| `file` _string_ | File is the path to the template in the blueprint's content. |  |  |
| `template` _[AnyJSON](#anyjson)_ | Template contains an optional inline template.<br />The template has to be of string for go template<br />and either a string or valid yaml/json for spiff. |  |  |


#### TemplateType

_Underlying type:_ _string_

TemplateType describes the template mechanism.



_Appears in:_
- [TemplateExecutor](#templateexecutor)



#### TokenRotation







_Appears in:_
- [TargetSyncSpec](#targetsyncspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | Enabled defines if automatic token is executed |  |  |


#### TransitionTimes







_Appears in:_
- [DeployItemStatus](#deployitemstatus)
- [ExecutionStatus](#executionstatus)
- [InstallationStatus](#installationstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `triggerTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#time-v1-meta)_ | TriggerTime is the time when the jobID is set. |  |  |
| `initTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#time-v1-meta)_ | InitTime is the time when the Init phase starts. |  |  |
| `waitTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#time-v1-meta)_ | WaitTime is the time when the work is done. |  |  |
| `finishedTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#time-v1-meta)_ | FinishedTime is the time when the finished phase is set. |  |  |




#### Verification



Verification defines the necessary data to verify the signature of the refered component



_Appears in:_
- [InstallationSpec](#installationspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `signatureName` _string_ | SignatureName defines the name of the signature that is verified |  |  |


#### VerificationSignature



VerificationSignatures contains the trusted verification information



_Appears in:_
- [Context](#context)
- [ContextConfiguration](#contextconfiguration)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `publicKeySecretReference` _[SecretReference](#secretreference)_ | PublicKeySecretReference contains a secret reference to a public key in PEM format that is used to verify the component signature |  |  |
| `caCertificateSecretReference` _[SecretReference](#secretreference)_ | CaCertificateSecretReference contains a secret reference to one or more certificates in PEM format that are used to verify the compnent signature |  |  |




#### VersionedObjectReference



VersionedObjectReference is a reference to a object with its last observed resource generation.
This struct is used by status fields.



_Appears in:_
- [VersionedNamedObjectReference](#versionednamedobjectreference)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the name of the kubernetes object. |  |  |
| `namespace` _string_ | Namespace is the namespace of kubernetes object. |  |  |
| `observedGeneration` _integer_ | ObservedGeneration defines the last observed generation of the referenced resource. |  |  |




