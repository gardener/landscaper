<p>Packages:</p>
<ul>
<li>
<a href="#landscaper.gardener.cloud%2fv1alpha1">landscaper.gardener.cloud/v1alpha1</a>
</li>
</ul>
<h2 id="landscaper.gardener.cloud/v1alpha1">landscaper.gardener.cloud/v1alpha1</h2>
<p>
<p>Package v1alpha1 is a version of the API.</p>
</p>
Resource Types:
<ul><li>
<a href="#landscaper.gardener.cloud/v1alpha1.Blueprint">Blueprint</a>
</li><li>
<a href="#landscaper.gardener.cloud/v1alpha1.ComponentVersionOverwrites">ComponentVersionOverwrites</a>
</li><li>
<a href="#landscaper.gardener.cloud/v1alpha1.Context">Context</a>
</li><li>
<a href="#landscaper.gardener.cloud/v1alpha1.DataObject">DataObject</a>
</li><li>
<a href="#landscaper.gardener.cloud/v1alpha1.DeployItem">DeployItem</a>
</li><li>
<a href="#landscaper.gardener.cloud/v1alpha1.DeployerRegistration">DeployerRegistration</a>
</li><li>
<a href="#landscaper.gardener.cloud/v1alpha1.Environment">Environment</a>
</li><li>
<a href="#landscaper.gardener.cloud/v1alpha1.Execution">Execution</a>
</li><li>
<a href="#landscaper.gardener.cloud/v1alpha1.Installation">Installation</a>
</li><li>
<a href="#landscaper.gardener.cloud/v1alpha1.InstallationTemplate">InstallationTemplate</a>
</li><li>
<a href="#landscaper.gardener.cloud/v1alpha1.LsHealthCheck">LsHealthCheck</a>
</li><li>
<a href="#landscaper.gardener.cloud/v1alpha1.SyncObject">SyncObject</a>
</li><li>
<a href="#landscaper.gardener.cloud/v1alpha1.Target">Target</a>
</li><li>
<a href="#landscaper.gardener.cloud/v1alpha1.TargetSync">TargetSync</a>
</li></ul>
<h3 id="landscaper.gardener.cloud/v1alpha1.Blueprint">Blueprint
</h3>
<p>
<p>Blueprint contains the configuration of a component</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
landscaper.gardener.cloud/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>Blueprint</code></td>
</tr>
<tr>
<td>
<code>annotations</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Annotations is an unstructured key value map stored with a resource that may be
set by external tools to store and retrieve arbitrary metadata.</p>
</td>
</tr>
<tr>
<td>
<code>jsonSchemaVersion</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>JSONSchemaVersion defines the default jsonschema version of the blueprint.
e.g. &ldquo;<a href="https://json-schema.org/draft/2019-09/schema&quot;">https://json-schema.org/draft/2019-09/schema&rdquo;</a></p>
</td>
</tr>
<tr>
<td>
<code>localTypes</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.JSONSchemaDefinition">
map[string]..JSONSchemaDefinition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>LocalTypes defines additional blueprint local schemas</p>
</td>
</tr>
<tr>
<td>
<code>imports</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ImportDefinitionList">
ImportDefinitionList
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Imports define the import values that are needed for the definition and its sub-definitions.</p>
</td>
</tr>
<tr>
<td>
<code>importExecutions</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.TemplateExecutor">
[]TemplateExecutor
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ImportExecutions defines the templating executors that are sequentially executed by the landscaper.
The templates must return a list of errors</p>
</td>
</tr>
<tr>
<td>
<code>exports</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ExportDefinitionList">
ExportDefinitionList
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Exports define the exported values of the definition and its sub-definitions</p>
</td>
</tr>
<tr>
<td>
<code>subinstallations</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.SubinstallationTemplateList">
SubinstallationTemplateList
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Subinstallations defines an optional list of subinstallations (for aggregating blueprints).</p>
</td>
</tr>
<tr>
<td>
<code>subinstallationExecutions</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.TemplateExecutor">
[]TemplateExecutor
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>SubinstallationExecutions defines the templating executors that are sequentially executed by the landscaper.
The templates must return a list of installation templates.
Both subinstallations and SubinstallationExecutions are valid options and will be merged.</p>
</td>
</tr>
<tr>
<td>
<code>deployExecutions</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.TemplateExecutor">
[]TemplateExecutor
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>DeployExecutions defines the templating executors that are sequentially executed by the landscaper.
The templates must return a list of deploy item templates.</p>
</td>
</tr>
<tr>
<td>
<code>exportExecutions</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.TemplateExecutor">
[]TemplateExecutor
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ExportExecutions defines the templating executors that are used to generate the exports.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.ComponentVersionOverwrites">ComponentVersionOverwrites
</h3>
<p>
<p>ComponentVersionOverwrites contain overwrites for specific (versions of) components.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
landscaper.gardener.cloud/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>ComponentVersionOverwrites</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://v1-22.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>overwrites</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ComponentVersionOverwriteList">
ComponentVersionOverwriteList
</a>
</em>
</td>
<td>
<p>Overwrites defines a list of component overwrites</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.Context">Context
</h3>
<p>
<p>Context is a resource that contains shared information of installations.
This includes information about the repository context like the context itself or secrets to access the oci artifacts.
But it can also contain deployer specific config.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
landscaper.gardener.cloud/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>Context</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://v1-22.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>repositoryContext</code></br>
<em>
<a href="https://pkg.go.dev/github.com/gardener/component-spec/bindings-go/apis/v2#UnstructuredTypedObject">
github.com/gardener/component-spec/bindings-go/apis/v2.UnstructuredTypedObject
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>RepositoryContext defines the context of the component repository to resolve blueprints.</p>
</td>
</tr>
<tr>
<td>
<code>useOCM</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>UseOCM defines whether OCM is used to process installations that reference this context.</p>
</td>
</tr>
<tr>
<td>
<code>registryPullSecrets</code></br>
<em>
<a href="https://v1-22.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#localobjectreference-v1-core">
[]Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>RegistryPullSecrets defines a list of registry credentials that are used to
pull blueprints, component descriptors and jsonschemas from the respective registry.
For more info see: <a href="https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/">https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/</a>
Note that the type information is used to determine the secret key and the type of the secret.</p>
</td>
</tr>
<tr>
<td>
<code>configurations</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.AnyJSON">
map[string]..AnyJSON
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Configurations contains arbitrary configuration information for dedicated purposes given by a string key.
The key should use a dns-like syntax to express the purpose and avoid conflicts.</p>
</td>
</tr>
<tr>
<td>
<code>componentVersionOverwrites</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ComponentVersionOverwritesReference is a reference to a ComponentVersionOverwrites object
The overwrites object has to be in the same namespace as the context.
If the string is empty, no overwrites will be used.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.DataObject">DataObject
</h3>
<p>
<p>DataObject are resources that can hold any kind json or yaml data.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
landscaper.gardener.cloud/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>DataObject</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://v1-22.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>data</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.AnyJSON">
AnyJSON
</a>
</em>
</td>
<td>
<p>Data contains the data of the object as string.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.DeployItem">DeployItem
</h3>
<p>
<p>DeployItem defines a resource that should be processed by a external deployer</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
landscaper.gardener.cloud/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>DeployItem</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://v1-22.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.DeployItemSpec">
DeployItemSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>type</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.DeployItemType">
DeployItemType
</a>
</em>
</td>
<td>
<p>Type is the type of the deployer that should handle the item.</p>
</td>
</tr>
<tr>
<td>
<code>target</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ObjectReference">
ObjectReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Target specifies an optional target of the deploy item.
In most cases it contains the secrets to access a evironment.
It is also used by the deployers to determine the ownernship.</p>
</td>
</tr>
<tr>
<td>
<code>context</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Context defines the current context of the deployitem.</p>
</td>
</tr>
<tr>
<td>
<code>config</code></br>
<em>
<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/runtime#RawExtension">
k8s.io/apimachinery/pkg/runtime.RawExtension
</a>
</em>
</td>
<td>
<p>Configuration contains the deployer type specific configuration.</p>
</td>
</tr>
<tr>
<td>
<code>registryPullSecrets</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ObjectReference">
[]ObjectReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>RegistryPullSecrets defines a list of registry credentials that are used to
pull blueprints, component descriptors and jsonschemas from the respective registry.
For more info see: <a href="https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/">https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/</a>
Note that the type information is used to determine the secret key and the type of the secret.</p>
</td>
</tr>
<tr>
<td>
<code>timeout</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.Duration">
Duration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Timeout specifies how long the deployer may take to apply the deploy item.
When the time is exceeded, the deploy item fails.
Value has to be parsable by time.ParseDuration (or &lsquo;none&rsquo; to deactivate the timeout).
Defaults to ten minutes if not specified.</p>
</td>
</tr>
<tr>
<td>
<code>updateOnChangeOnly</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>UpdateOnChangeOnly specifies if redeployment is executed only if the specification of the deploy item has changed.</p>
</td>
</tr>
<tr>
<td>
<code>onDelete</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.OnDeleteConfig">
OnDeleteConfig
</a>
</em>
</td>
<td>
<p>OnDelete specifies particular setting when deleting a deploy item</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.DeployItemStatus">
DeployItemStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.DeployerRegistration">DeployerRegistration
</h3>
<p>
<p>DeployerRegistration defines a installation template for a deployer.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
landscaper.gardener.cloud/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>DeployerRegistration</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://v1-22.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.DeployerRegistrationSpec">
DeployerRegistrationSpec
</a>
</em>
</td>
<td>
<p>Spec defines the deployer registration configuration.</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>types</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.DeployItemType">
[]DeployItemType
</a>
</em>
</td>
<td>
<p>DeployItemTypes defines the types of deploy items that are handled by the deployer.</p>
</td>
</tr>
<tr>
<td>
<code>installationTemplate</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.DeployerInstallationTemplate">
DeployerInstallationTemplate
</a>
</em>
</td>
<td>
<p>InstallationTemplate defines the installation template for installing a deployer.´</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.DeployerRegistrationStatus">
DeployerRegistrationStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Status contains the current status of the deployer registration.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.Environment">Environment
</h3>
<p>
<p>Environment defines a environment that is created by a agent.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
landscaper.gardener.cloud/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>Environment</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://v1-22.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.EnvironmentSpec">
EnvironmentSpec
</a>
</em>
</td>
<td>
<p>Spec defines the environment.</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>hostTarget</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.TargetTemplate">
TargetTemplate
</a>
</em>
</td>
<td>
<p>HostTarget describes the target that is used for the deployers.</p>
</td>
</tr>
<tr>
<td>
<code>namespace</code></br>
<em>
string
</em>
</td>
<td>
<p>Namespace is the host cluster namespace where the deployers should be installed.</p>
</td>
</tr>
<tr>
<td>
<code>landscaperClusterConfig</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ClusterRestConfig">
ClusterRestConfig
</a>
</em>
</td>
<td>
<p>LandscaperClusterRestConfig describes the connection information to connect to the
landscaper cluster.
This information should be provided by the agent as the access information may differ
when calling from different networking zones.</p>
</td>
</tr>
<tr>
<td>
<code>targetSelectors</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.TargetSelector">
[]TargetSelector
</a>
</em>
</td>
<td>
<p>TargetSelector defines the target selector that is applied to all installed deployers</p>
</td>
</tr>
</table>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.Execution">Execution
</h3>
<p>
<p>Execution contains the configuration of a execution and deploy item</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
landscaper.gardener.cloud/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>Execution</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://v1-22.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ExecutionSpec">
ExecutionSpec
</a>
</em>
</td>
<td>
<p>Spec defines a execution and its items</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>context</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Context defines the current context of the execution.</p>
</td>
</tr>
<tr>
<td>
<code>deployItems</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.DeployItemTemplateList">
DeployItemTemplateList
</a>
</em>
</td>
<td>
<p>DeployItems defines all execution items that need to be scheduled.</p>
</td>
</tr>
<tr>
<td>
<code>deployItemsCompressed</code></br>
<em>
[]byte
</em>
</td>
<td>
<p>DeployItemsCompressed as zipped byte array</p>
</td>
</tr>
<tr>
<td>
<code>registryPullSecrets</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ObjectReference">
[]ObjectReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>RegistryPullSecrets defines a list of registry credentials that are used to
pull blueprints, component descriptors and jsonschemas from the respective registry.
For more info see: <a href="https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/">https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/</a>
Note that the type information is used to determine the secret key and the type of the secret.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ExecutionStatus">
ExecutionStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Status contains the current status of the execution.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.Installation">Installation
</h3>
<p>
<p>Installation contains the configuration of a component</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
landscaper.gardener.cloud/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>Installation</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://v1-22.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.InstallationSpec">
InstallationSpec
</a>
</em>
</td>
<td>
<p>Spec contains the specification for a installation.</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>context</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Context defines the current context of the installation.</p>
</td>
</tr>
<tr>
<td>
<code>componentDescriptor</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ComponentDescriptorDefinition">
ComponentDescriptorDefinition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ComponentDescriptor is a reference to the installation&rsquo;s component descriptor</p>
</td>
</tr>
<tr>
<td>
<code>blueprint</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.BlueprintDefinition">
BlueprintDefinition
</a>
</em>
</td>
<td>
<p>Blueprint is the resolved reference to the definition.</p>
</td>
</tr>
<tr>
<td>
<code>registryPullSecrets</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ObjectReference">
[]ObjectReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>RegistryPullSecrets defines a list of registry credentials that are used to
pull blueprints, component descriptors and jsonschemas from the respective registry.
For more info see: <a href="https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/">https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/</a>
Note that the type information is used to determine the secret key and the type of the secret.</p>
</td>
</tr>
<tr>
<td>
<code>imports</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.InstallationImports">
InstallationImports
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Imports define the imported data objects and targets.</p>
</td>
</tr>
<tr>
<td>
<code>importDataMappings</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.AnyJSON">
map[string]..AnyJSON
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ImportDataMappings contains a template for restructuring imports.
It is expected to contain a key for every blueprint-defined data import.
Missing keys will be defaulted to their respective data import.
Example: namespace: (( installation.imports.namespace ))</p>
</td>
</tr>
<tr>
<td>
<code>exports</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.InstallationExports">
InstallationExports
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Exports define the exported data objects and targets.</p>
</td>
</tr>
<tr>
<td>
<code>exportDataMappings</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.AnyJSON">
map[string]..AnyJSON
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ExportDataMappings contains a template for restructuring exports.
It is expected to contain a key for every blueprint-defined data export.
Missing keys will be defaulted to their respective data export.
Example: namespace: (( blueprint.exports.namespace ))</p>
</td>
</tr>
<tr>
<td>
<code>automaticReconcile</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.AutomaticReconcile">
AutomaticReconcile
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>AutomaticReconcile allows to configure automatically repeated reconciliations.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.InstallationStatus">
InstallationStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Status contains the status of the installation.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.InstallationTemplate">InstallationTemplate
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.SubinstallationTemplate">SubinstallationTemplate</a>)
</p>
<p>
<p>InstallationTemplate defines a subinstallation in a blueprint.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
landscaper.gardener.cloud/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>InstallationTemplate</code></td>
</tr>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name is the unique name of the step</p>
</td>
</tr>
<tr>
<td>
<code>blueprint</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.InstallationTemplateBlueprintDefinition">
InstallationTemplateBlueprintDefinition
</a>
</em>
</td>
<td>
<p>Reference defines a reference to a Blueprint.
The blueprint can reside in an OCI or other supported location.</p>
</td>
</tr>
<tr>
<td>
<code>imports</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.InstallationImports">
InstallationImports
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Imports define the imported data objects and targets.</p>
</td>
</tr>
<tr>
<td>
<code>importDataMappings</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.AnyJSON">
map[string]..AnyJSON
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ImportDataMappings contains a template for restructuring imports.
It is expected to contain a key for every blueprint-defined data import.
Missing keys will be defaulted to their respective data import.
Example: namespace: (( installation.imports.namespace ))</p>
</td>
</tr>
<tr>
<td>
<code>exports</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.InstallationExports">
InstallationExports
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Exports define the exported data objects and targets.</p>
</td>
</tr>
<tr>
<td>
<code>exportDataMappings</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.AnyJSON">
map[string]..AnyJSON
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ExportDataMappings contains a template for restructuring exports.
It is expected to contain a key for every blueprint-defined data export.
Missing keys will be defaulted to their respective data export.
Example: namespace: (( blueprint.exports.namespace ))</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.LsHealthCheck">LsHealthCheck
</h3>
<p>
<p>LsHealthCheck is a resource containing information about problems with the landscaper installation</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
landscaper.gardener.cloud/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>LsHealthCheck</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://v1-22.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.LsHealthCheckStatus">
LsHealthCheckStatus
</a>
</em>
</td>
<td>
<p>Status defines the overall status of the landscaper and its standard deployers.</p>
</td>
</tr>
<tr>
<td>
<code>lastUpdateTime</code></br>
<em>
<a href="https://v1-22.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>LastUpdateTime contains last time the check was updated.</p>
</td>
</tr>
<tr>
<td>
<code>description</code></br>
<em>
string
</em>
</td>
<td>
<p>Description contains description of the problem(s)</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.SyncObject">SyncObject
</h3>
<p>
<p>The SyncObject helps to sync access to deploy items.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
landscaper.gardener.cloud/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>SyncObject</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://v1-22.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.SyncObjectSpec">
SyncObjectSpec
</a>
</em>
</td>
<td>
<p>Spec contains the specification</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>podName</code></br>
<em>
string
</em>
</td>
<td>
<p>PodName describes the name of the pod of the responsible deployer</p>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
<em>
string
</em>
</td>
<td>
<p>Kind describes the kind of object that is being locked by this SyncObject</p>
</td>
</tr>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name is the name of the object that is being locked by this SyncObject</p>
</td>
</tr>
<tr>
<td>
<code>lastUpdateTime</code></br>
<em>
<a href="https://v1-22.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>LastUpdateTime contains last time the object was updated.</p>
</td>
</tr>
<tr>
<td>
<code>prefix</code></br>
<em>
string
</em>
</td>
<td>
<p>Prefix is the prefix of the name of the object.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.SyncObjectStatus">
SyncObjectStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Status contains the status</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.Target">Target
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.ResolvedTarget">ResolvedTarget</a>)
</p>
<p>
<p>Target defines a specific data object that defines target environment.
Every deploy item can have a target which is used by the deployer to install the specific application.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
landscaper.gardener.cloud/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>Target</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://v1-22.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.TargetSpec">
TargetSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>type</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.TargetType">
TargetType
</a>
</em>
</td>
<td>
<p>Type is the type of the target that defines its data structure.
The actual schema may be defined by a target type crd in the future.</p>
</td>
</tr>
<tr>
<td>
<code>config</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.AnyJSON">
AnyJSON
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Configuration contains the target type specific configuration.
Exactly one of the fields Configuration and SecretRef must be set</p>
</td>
</tr>
<tr>
<td>
<code>secretRef</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.LocalSecretReference">
LocalSecretReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Reference to a secret containing the target type specific configuration.
Exactly one of the fields Configuration and SecretRef must be set</p>
</td>
</tr>
</table>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.TargetSync">TargetSync
</h3>
<p>
<p>The TargetSync is created targets from secrets.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
landscaper.gardener.cloud/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>TargetSync</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://v1-22.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.TargetSyncSpec">
TargetSyncSpec
</a>
</em>
</td>
<td>
<p>Spec contains the specification</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>sourceNamespace</code></br>
<em>
string
</em>
</td>
<td>
<p>SourceNamespace describes the namespace from where the secrets should be synced</p>
</td>
</tr>
<tr>
<td>
<code>secretRef</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.LocalSecretReference">
LocalSecretReference
</a>
</em>
</td>
<td>
<p>SecretRef references the secret that contains the kubeconfig to the namespace of the secrets to be synced.</p>
</td>
</tr>
<tr>
<td>
<code>createTargetToSource</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>CreateTargetToSource specifies if set on true, that also a target is created, which references the secret in SecretRef</p>
</td>
</tr>
<tr>
<td>
<code>targetToSourceName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>TargetToSourceName is the name of the target referencing the secret defined in SecretRef if CreateTargetToSource
is set on true. If TargetToSourceName is empty SourceNamespace is used instead.</p>
</td>
</tr>
<tr>
<td>
<code>secretNameExpression</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>SecretNameExpression defines the names of the secrets which should be synced via a regular expression according
to <a href="https://github.com/google/re2/wiki/Syntax">https://github.com/google/re2/wiki/Syntax</a> with the extension that * is also a valid expression and matches
all names.
if not set no secrets are synced</p>
</td>
</tr>
<tr>
<td>
<code>shootNameExpression</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ShootNameExpression defines the names of shoot clusters for which targets with short living access data
to the shoots are created via a regular expression according to <a href="https://github.com/google/re2/wiki/Syntax">https://github.com/google/re2/wiki/Syntax</a> with
the extension that * is also a valid expression and matches all names.
if not set no targets for the shoots are created</p>
</td>
</tr>
<tr>
<td>
<code>tokenRotation</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.TokenRotation">
TokenRotation
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TokenRotation defines the data to perform an automatic rotation of the token to access the source cluster with the
secrets to sync. The token expires after 90 days and will be rotated every 60 days.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.TargetSyncStatus">
TargetSyncStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Status contains the status</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.AnyJSON">AnyJSON
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.Context">Context</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.DataObject">DataObject</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.InstallationTemplate">InstallationTemplate</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.BlueprintStaticDataSource">BlueprintStaticDataSource</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.Default">Default</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.DeployerInstallationTemplate">DeployerInstallationTemplate</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.InlineBlueprint">InlineBlueprint</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.InstallationSpec">InstallationSpec</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.InstallationTemplateBlueprintDefinition">InstallationTemplateBlueprintDefinition</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.StaticDataSource">StaticDataSource</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.TargetSpec">TargetSpec</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.TemplateExecutor">TemplateExecutor</a>)
</p>
<p>
<p>AnyJSON enhances the json.RawMessages with a dedicated openapi definition so that all
it is correctly generated</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>RawMessage</code></br>
<em>
<a href="https://pkg.go.dev/pkg/encoding/json/#RawMessage">
encoding/json.RawMessage
</a>
</em>
</td>
<td>
<p>
(Members of <code>RawMessage</code> are embedded into this type.)
</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.AutomaticReconcile">AutomaticReconcile
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.InstallationSpec">InstallationSpec</a>)
</p>
<p>
<p>AutomaticReconcile allows to configure automatically repeated reconciliations.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>succeededReconcile</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.SucceededReconcile">
SucceededReconcile
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>SucceededReconcile allows to configure automatically repeated reconciliations for succeeded installations.
If not set, no such automatically repeated reconciliations are triggered.</p>
</td>
</tr>
<tr>
<td>
<code>failedReconcile</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.FailedReconcile">
FailedReconcile
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>FailedReconcile allows to configure automatically repeated reconciliations for failed installations.
If not set, no such automatically repeated reconciliations are triggered.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.AutomaticReconcileStatus">AutomaticReconcileStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.InstallationStatus">InstallationStatus</a>)
</p>
<p>
<p>AutomaticReconcileStatus describes the status of automatically triggered reconciles.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>generation</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Generation describes the generation of the installation for which the status holds.</p>
</td>
</tr>
<tr>
<td>
<code>numberOfReconciles</code></br>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
<p>NumberOfReconciles is the number of automatic reconciles for the installation with the stored generation.</p>
</td>
</tr>
<tr>
<td>
<code>lastReconcileTime</code></br>
<em>
<a href="https://v1-22.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>LastReconcileTime is the time of the last automatically triggered reconcile.</p>
</td>
</tr>
<tr>
<td>
<code>onFailed</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>OnFailed is true if the last automatically triggered reconcile was done for a failed installation.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.BlueprintDefinition">BlueprintDefinition
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.DeployerInstallationTemplate">DeployerInstallationTemplate</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.InstallationSpec">InstallationSpec</a>)
</p>
<p>
<p>BlueprintDefinition defines the blueprint that should be used for the installation.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ref</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.RemoteBlueprintReference">
RemoteBlueprintReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Reference defines a remote reference to a blueprint</p>
</td>
</tr>
<tr>
<td>
<code>inline</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.InlineBlueprint">
InlineBlueprint
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Inline defines a inline yaml filesystem with a blueprint.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.BlueprintStaticDataSource">BlueprintStaticDataSource
</h3>
<p>
<p>BlueprintStaticDataSource defines a static data source for a blueprint</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>value</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.AnyJSON">
AnyJSON
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Value defined inline a raw data</p>
</td>
</tr>
<tr>
<td>
<code>valueFrom</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.StaticDataValueFrom">
StaticDataValueFrom
</a>
</em>
</td>
<td>
<p>ValueFrom defines data from an external resource</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.BlueprintStaticDataValueFrom">BlueprintStaticDataValueFrom
</h3>
<p>
<p>BlueprintStaticDataValueFrom defines static data that is read from a external resource.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>localPath</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Selects a key of a secret in the installations&rsquo;s namespace</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.ClusterRestConfig">ClusterRestConfig
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.EnvironmentSpec">EnvironmentSpec</a>)
</p>
<p>
<p>ClusterRestConfig describes parts of a rest.Config
that is used to access the</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>host</code></br>
<em>
string
</em>
</td>
<td>
<p>Host must be a host string, a host:port pair, or a URL to the base of the apiserver.
If a URL is given then the (optional) Path of that URL represents a prefix that must
be appended to all request URIs used to access the apiserver. This allows a frontend
proxy to easily relocate all of the apiserver endpoints.</p>
</td>
</tr>
<tr>
<td>
<code>apiPath</code></br>
<em>
string
</em>
</td>
<td>
<p>APIPath is a sub-path that points to an API root.</p>
</td>
</tr>
<tr>
<td>
<code>TLSClientConfig</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.TLSClientConfig">
TLSClientConfig
</a>
</em>
</td>
<td>
<p>
(Members of <code>TLSClientConfig</code> are embedded into this type.)
</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.ComponentDescriptorDefinition">ComponentDescriptorDefinition
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.DeployerInstallationTemplate">DeployerInstallationTemplate</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.InstallationSpec">InstallationSpec</a>)
</p>
<p>
<p>ComponentDescriptorDefinition defines the component descriptor that should be used
for the installation</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ref</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ComponentDescriptorReference">
ComponentDescriptorReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ComponentDescriptorReference is the reference to a component descriptor</p>
</td>
</tr>
<tr>
<td>
<code>inline</code></br>
<em>
<a href="https://pkg.go.dev/github.com/gardener/component-spec/bindings-go/apis/v2#ComponentDescriptor">
github.com/gardener/component-spec/bindings-go/apis/v2.ComponentDescriptor
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>InlineDescriptorReference defines an inline component descriptor</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.ComponentDescriptorKind">ComponentDescriptorKind
(<code>string</code> alias)</p></h3>
<p>
<p>ComponentDescriptorKind is the kind of a component descriptor.
It can be a component or a resource.</p>
</p>
<h3 id="landscaper.gardener.cloud/v1alpha1.ComponentDescriptorReference">ComponentDescriptorReference
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.ComponentDescriptorDefinition">ComponentDescriptorDefinition</a>)
</p>
<p>
<p>ComponentDescriptorReference is the reference to a component descriptor.
given an optional context.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>repositoryContext</code></br>
<em>
<a href="https://pkg.go.dev/github.com/gardener/component-spec/bindings-go/apis/v2#UnstructuredTypedObject">
github.com/gardener/component-spec/bindings-go/apis/v2.UnstructuredTypedObject
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>RepositoryContext defines the context of the component repository to resolve blueprints.</p>
</td>
</tr>
<tr>
<td>
<code>componentName</code></br>
<em>
string
</em>
</td>
<td>
<p>ComponentName defines the unique of the component containing the resource.</p>
</td>
</tr>
<tr>
<td>
<code>version</code></br>
<em>
string
</em>
</td>
<td>
<p>Version defines the version of the component.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.ComponentVersionOverwrite">ComponentVersionOverwrite
</h3>
<p>
<p>ComponentVersionOverwrite defines an overwrite for a specific component and/or version of a component.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>source</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ComponentVersionOverwriteReference">
ComponentVersionOverwriteReference
</a>
</em>
</td>
<td>
<p>Source defines the component that should be replaced.</p>
</td>
</tr>
<tr>
<td>
<code>substitution</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ComponentVersionOverwriteReference">
ComponentVersionOverwriteReference
</a>
</em>
</td>
<td>
<p>Substitution defines the replacement target for the component or version.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.ComponentVersionOverwriteReference">ComponentVersionOverwriteReference
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.ComponentVersionOverwrite">ComponentVersionOverwrite</a>)
</p>
<p>
<p>ComponentVersionOverwriteReference defines a component reference by</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>repositoryContext</code></br>
<em>
<a href="https://pkg.go.dev/github.com/gardener/component-spec/bindings-go/apis/v2#UnstructuredTypedObject">
github.com/gardener/component-spec/bindings-go/apis/v2.UnstructuredTypedObject
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>RepositoryContext defines the context of the component repository to resolve blueprints.</p>
</td>
</tr>
<tr>
<td>
<code>componentName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ComponentName defines the unique of the component containing the resource.</p>
</td>
</tr>
<tr>
<td>
<code>version</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Version defines the version of the component.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.Condition">Condition
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.DeployItemStatus">DeployItemStatus</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.ExecutionStatus">ExecutionStatus</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.InstallationStatus">InstallationStatus</a>)
</p>
<p>
<p>Condition holds the information about the state of a resource.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>type</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ConditionType">
ConditionType
</a>
</em>
</td>
<td>
<p>DataType of the Shoot condition.</p>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ConditionStatus">
ConditionStatus
</a>
</em>
</td>
<td>
<p>Status of the condition, one of True, False, Unknown.</p>
</td>
</tr>
<tr>
<td>
<code>lastTransitionTime</code></br>
<em>
<a href="https://v1-22.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>Last time the condition transitioned from one status to another.</p>
</td>
</tr>
<tr>
<td>
<code>lastUpdateTime</code></br>
<em>
<a href="https://v1-22.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>Last time the condition was updated.</p>
</td>
</tr>
<tr>
<td>
<code>reason</code></br>
<em>
string
</em>
</td>
<td>
<p>The reason for the condition&rsquo;s last transition.</p>
</td>
</tr>
<tr>
<td>
<code>message</code></br>
<em>
string
</em>
</td>
<td>
<p>A human readable message indicating details about the transition.</p>
</td>
</tr>
<tr>
<td>
<code>codes</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ErrorCode">
[]ErrorCode
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Well-defined error codes in case the condition reports a problem.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.ConditionStatus">ConditionStatus
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.Condition">Condition</a>)
</p>
<p>
<p>ConditionStatus is the status of a condition.</p>
</p>
<h3 id="landscaper.gardener.cloud/v1alpha1.ConditionType">ConditionType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.Condition">Condition</a>)
</p>
<p>
<p>ConditionType is a string alias.</p>
</p>
<h3 id="landscaper.gardener.cloud/v1alpha1.ConfigMapReference">ConfigMapReference
</h3>
<p>
<p>ConfigMapReference is reference to data in a configmap.
The configmap can also be in a different namespace.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ObjectReference</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ObjectReference">
ObjectReference
</a>
</em>
</td>
<td>
<p>
(Members of <code>ObjectReference</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>key</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Key is the name of the key in the configmap that holds the data.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.DataExport">DataExport
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.InstallationExports">InstallationExports</a>)
</p>
<p>
<p>DataExport is a data object export.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name the internal name of the imported/exported data.</p>
</td>
</tr>
<tr>
<td>
<code>dataRef</code></br>
<em>
string
</em>
</td>
<td>
<p>DataRef is the name of the in-cluster data object.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.DataImport">DataImport
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.InstallationImports">InstallationImports</a>)
</p>
<p>
<p>DataImport is a data object import.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name the internal name of the imported/exported data.</p>
</td>
</tr>
<tr>
<td>
<code>dataRef</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>DataRef is the name of the in-cluster data object.
The reference can also be a namespaces name. E.g. &ldquo;default/mydataref&rdquo;</p>
</td>
</tr>
<tr>
<td>
<code>version</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Version specifies the imported data version.
defaults to &ldquo;v1&rdquo;</p>
</td>
</tr>
<tr>
<td>
<code>secretRef</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.LocalSecretReference">
LocalSecretReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>SecretRef defines a data reference from a secret.
This method is not allowed in installation templates.</p>
</td>
</tr>
<tr>
<td>
<code>configMapRef</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.LocalConfigMapReference">
LocalConfigMapReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ConfigMapRef defines a data reference from a configmap.
This method is not allowed in installation templates.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.DataObjectSourceType">DataObjectSourceType
(<code>string</code> alias)</p></h3>
<p>
<p>DataObjectSourceType defines the context of a data object.</p>
</p>
<h3 id="landscaper.gardener.cloud/v1alpha1.Default">Default
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.ImportDefinition">ImportDefinition</a>)
</p>
<p>
<p>Default defines a default value (future idea: also reference?).</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>value</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.AnyJSON">
AnyJSON
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.DependentToTrigger">DependentToTrigger
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.InstallationStatus">InstallationStatus</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name is the name of the dependent installation</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.DeployItemPhase">DeployItemPhase
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.DeployItemStatus">DeployItemStatus</a>)
</p>
<p>
</p>
<h3 id="landscaper.gardener.cloud/v1alpha1.DeployItemSpec">DeployItemSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.DeployItem">DeployItem</a>)
</p>
<p>
<p>DeployItemSpec contains the definition of a deploy item.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>type</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.DeployItemType">
DeployItemType
</a>
</em>
</td>
<td>
<p>Type is the type of the deployer that should handle the item.</p>
</td>
</tr>
<tr>
<td>
<code>target</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ObjectReference">
ObjectReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Target specifies an optional target of the deploy item.
In most cases it contains the secrets to access a evironment.
It is also used by the deployers to determine the ownernship.</p>
</td>
</tr>
<tr>
<td>
<code>context</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Context defines the current context of the deployitem.</p>
</td>
</tr>
<tr>
<td>
<code>config</code></br>
<em>
<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/runtime#RawExtension">
k8s.io/apimachinery/pkg/runtime.RawExtension
</a>
</em>
</td>
<td>
<p>Configuration contains the deployer type specific configuration.</p>
</td>
</tr>
<tr>
<td>
<code>registryPullSecrets</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ObjectReference">
[]ObjectReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>RegistryPullSecrets defines a list of registry credentials that are used to
pull blueprints, component descriptors and jsonschemas from the respective registry.
For more info see: <a href="https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/">https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/</a>
Note that the type information is used to determine the secret key and the type of the secret.</p>
</td>
</tr>
<tr>
<td>
<code>timeout</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.Duration">
Duration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Timeout specifies how long the deployer may take to apply the deploy item.
When the time is exceeded, the deploy item fails.
Value has to be parsable by time.ParseDuration (or &lsquo;none&rsquo; to deactivate the timeout).
Defaults to ten minutes if not specified.</p>
</td>
</tr>
<tr>
<td>
<code>updateOnChangeOnly</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>UpdateOnChangeOnly specifies if redeployment is executed only if the specification of the deploy item has changed.</p>
</td>
</tr>
<tr>
<td>
<code>onDelete</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.OnDeleteConfig">
OnDeleteConfig
</a>
</em>
</td>
<td>
<p>OnDelete specifies particular setting when deleting a deploy item</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.DeployItemStatus">DeployItemStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.DeployItem">DeployItem</a>)
</p>
<p>
<p>DeployItemStatus contains the status of a deploy item.
todo: add operation</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>phase</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.DeployItemPhase">
DeployItemPhase
</a>
</em>
</td>
<td>
<p>Phase is the current phase of the DeployItem</p>
</td>
</tr>
<tr>
<td>
<code>observedGeneration</code></br>
<em>
int64
</em>
</td>
<td>
<p>ObservedGeneration is the most recent generation observed for this DeployItem.
It corresponds to the DeployItem generation, which is updated on mutation by the landscaper.</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.Condition">
[]Condition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Conditions contains the actual condition of a deploy item</p>
</td>
</tr>
<tr>
<td>
<code>lastError</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.Error">
Error
</a>
</em>
</td>
<td>
<p>LastError describes the last error that occurred.</p>
</td>
</tr>
<tr>
<td>
<code>lastErrors</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.*..Error">
[]*..Error
</a>
</em>
</td>
<td>
<p>ErrorHistory describes the last n errors that occurred since JobID was changed the last time.</p>
</td>
</tr>
<tr>
<td>
<code>firstError</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.Error">
Error
</a>
</em>
</td>
<td>
<p>FirstError describes the first error that occurred since JobID was changed the last time.</p>
</td>
</tr>
<tr>
<td>
<code>lastReconcileTime</code></br>
<em>
<a href="https://v1-22.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>LastReconcileTime indicates when the reconciliation of the last change to the deploy item has started</p>
</td>
</tr>
<tr>
<td>
<code>deployer</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.DeployerInformation">
DeployerInformation
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Deployer describes the deployer that has reconciled the deploy item.</p>
</td>
</tr>
<tr>
<td>
<code>providerStatus</code></br>
<em>
<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/runtime#RawExtension">
k8s.io/apimachinery/pkg/runtime.RawExtension
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ProviderStatus contains the provider specific status</p>
</td>
</tr>
<tr>
<td>
<code>exportRef</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ObjectReference">
ObjectReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ExportReference is the reference to the object that contains the exported values.</p>
</td>
</tr>
<tr>
<td>
<code>jobID</code></br>
<em>
string
</em>
</td>
<td>
<p>JobID is the ID of the current working request.</p>
</td>
</tr>
<tr>
<td>
<code>jobIDFinished</code></br>
<em>
string
</em>
</td>
<td>
<p>JobIDFinished is the ID of the finished working request.</p>
</td>
</tr>
<tr>
<td>
<code>jobIDGenerationTime</code></br>
<em>
<a href="https://v1-22.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>JobIDGenerationTime is the timestamp when the JobID was set.</p>
</td>
</tr>
<tr>
<td>
<code>deployItemPhase</code></br>
<em>
string
</em>
</td>
<td>
<p>DeployerPhase is DEPRECATED and will soon be removed.</p>
</td>
</tr>
<tr>
<td>
<code>transitionTimes</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.TransitionTimes">
TransitionTimes
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TransitionTimes contains timestamps of status transitions</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.DeployItemTemplate">DeployItemTemplate
</h3>
<p>
<p>DeployItemTemplate defines a execution element that is translated into a deploy item.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name is the unique name of the execution.</p>
</td>
</tr>
<tr>
<td>
<code>type</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.DeployItemType">
DeployItemType
</a>
</em>
</td>
<td>
<p>DataType is the DeployItem type of the execution.</p>
</td>
</tr>
<tr>
<td>
<code>target</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ObjectReference">
ObjectReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Target is the object reference to the target that the deploy item should deploy to.</p>
</td>
</tr>
<tr>
<td>
<code>labels</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Labels is the map of labels to be added to the deploy item.</p>
</td>
</tr>
<tr>
<td>
<code>config</code></br>
<em>
<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/runtime#RawExtension">
k8s.io/apimachinery/pkg/runtime.RawExtension
</a>
</em>
</td>
<td>
<p>ProviderConfiguration contains the type specific configuration for the execution.</p>
</td>
</tr>
<tr>
<td>
<code>dependsOn</code></br>
<em>
[]string
</em>
</td>
<td>
<p>DependsOn lists deploy items that need to be executed before this one</p>
</td>
</tr>
<tr>
<td>
<code>timeout</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.Duration">
Duration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Timeout specifies how long the deployer may take to apply the deploy item.
When the time is exceeded, the deploy item fails.
Value has to be parsable by time.ParseDuration (or &lsquo;none&rsquo; to deactivate the timeout).
Defaults to ten minutes if not specified.</p>
</td>
</tr>
<tr>
<td>
<code>updateOnChangeOnly</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>UpdateOnChangeOnly specifies if redeployment is executed only if the specification of the deploy item has changed.</p>
</td>
</tr>
<tr>
<td>
<code>onDelete</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.OnDeleteConfig">
OnDeleteConfig
</a>
</em>
</td>
<td>
<p>OnDelete specifies particular setting when deleting a deploy item</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.DeployItemType">DeployItemType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.DeployItemSpec">DeployItemSpec</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.DeployItemTemplate">DeployItemTemplate</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.DeployerRegistrationSpec">DeployerRegistrationSpec</a>)
</p>
<p>
<p>DeployItemType defines the type of the deploy item</p>
</p>
<h3 id="landscaper.gardener.cloud/v1alpha1.DeployerInformation">DeployerInformation
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.DeployItemStatus">DeployItemStatus</a>)
</p>
<p>
<p>DeployerInformation holds additional information about the deployer that
has reconciled or is reconciling the deploy item.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>identity</code></br>
<em>
string
</em>
</td>
<td>
<p>Identity describes the unique identity of the deployer.</p>
</td>
</tr>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name is the name of the deployer.</p>
</td>
</tr>
<tr>
<td>
<code>version</code></br>
<em>
string
</em>
</td>
<td>
<p>Version is the version of the deployer.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.DeployerInstallationTemplate">DeployerInstallationTemplate
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.DeployerRegistrationSpec">DeployerRegistrationSpec</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>componentDescriptor</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ComponentDescriptorDefinition">
ComponentDescriptorDefinition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ComponentDescriptor is a reference to the installation&rsquo;s component descriptor</p>
</td>
</tr>
<tr>
<td>
<code>blueprint</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.BlueprintDefinition">
BlueprintDefinition
</a>
</em>
</td>
<td>
<p>Blueprint is the resolved reference to the definition.</p>
</td>
</tr>
<tr>
<td>
<code>imports</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.InstallationImports">
InstallationImports
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Imports define the imported data objects and targets.</p>
</td>
</tr>
<tr>
<td>
<code>importDataMappings</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.AnyJSON">
map[string]..AnyJSON
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ImportDataMappings contains a template for restructuring imports.
It is expected to contain a key for every blueprint-defined data import.
Missing keys will be defaulted to their respective data import.
Example: namespace: (( installation.imports.namespace ))</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.DeployerRegistrationSpec">DeployerRegistrationSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.DeployerRegistration">DeployerRegistration</a>)
</p>
<p>
<p>DeployerRegistrationSpec defines the configuration of a deployer registration</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>types</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.DeployItemType">
[]DeployItemType
</a>
</em>
</td>
<td>
<p>DeployItemTypes defines the types of deploy items that are handled by the deployer.</p>
</td>
</tr>
<tr>
<td>
<code>installationTemplate</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.DeployerInstallationTemplate">
DeployerInstallationTemplate
</a>
</em>
</td>
<td>
<p>InstallationTemplate defines the installation template for installing a deployer.´</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.DeployerRegistrationStatus">DeployerRegistrationStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.DeployerRegistration">DeployerRegistration</a>)
</p>
<p>
<p>DeployerRegistrationStatus contains the current status of a deployer registration.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>lastError</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.Error">
Error
</a>
</em>
</td>
<td>
<p>LastError describes the last error that occurred.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.Duration">Duration
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.DeployItemSpec">DeployItemSpec</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.DeployItemTemplate">DeployItemTemplate</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.FailedReconcile">FailedReconcile</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.SucceededReconcile">SucceededReconcile</a>)
</p>
<p>
<p>Duration is a wrapper for time.Duration that implements JSON marshalling and openapi scheme.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>Duration</code></br>
<em>
<a href="https://pkg.go.dev/time#Duration">
time.Duration
</a>
</em>
</td>
<td>
<p>
(Members of <code>Duration</code> are embedded into this type.)
</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.EnvironmentSpec">EnvironmentSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.Environment">Environment</a>)
</p>
<p>
<p>EnvironmentSpec defines the environment configuration.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>hostTarget</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.TargetTemplate">
TargetTemplate
</a>
</em>
</td>
<td>
<p>HostTarget describes the target that is used for the deployers.</p>
</td>
</tr>
<tr>
<td>
<code>namespace</code></br>
<em>
string
</em>
</td>
<td>
<p>Namespace is the host cluster namespace where the deployers should be installed.</p>
</td>
</tr>
<tr>
<td>
<code>landscaperClusterConfig</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ClusterRestConfig">
ClusterRestConfig
</a>
</em>
</td>
<td>
<p>LandscaperClusterRestConfig describes the connection information to connect to the
landscaper cluster.
This information should be provided by the agent as the access information may differ
when calling from different networking zones.</p>
</td>
</tr>
<tr>
<td>
<code>targetSelectors</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.TargetSelector">
[]TargetSelector
</a>
</em>
</td>
<td>
<p>TargetSelector defines the target selector that is applied to all installed deployers</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.Error">Error
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.DeployItemStatus">DeployItemStatus</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.DeployerRegistrationStatus">DeployerRegistrationStatus</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.ExecutionStatus">ExecutionStatus</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.InstallationStatus">InstallationStatus</a>)
</p>
<p>
<p>Error holds information about an error that occurred.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>operation</code></br>
<em>
string
</em>
</td>
<td>
<p>Operation describes the operator where the error occurred.</p>
</td>
</tr>
<tr>
<td>
<code>lastTransitionTime</code></br>
<em>
<a href="https://v1-22.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>Last time the condition transitioned from one status to another.</p>
</td>
</tr>
<tr>
<td>
<code>lastUpdateTime</code></br>
<em>
<a href="https://v1-22.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>Last time the condition was updated.</p>
</td>
</tr>
<tr>
<td>
<code>reason</code></br>
<em>
string
</em>
</td>
<td>
<p>The reason for the condition&rsquo;s last transition.</p>
</td>
</tr>
<tr>
<td>
<code>message</code></br>
<em>
string
</em>
</td>
<td>
<p>A human readable message indicating details about the transition.</p>
</td>
</tr>
<tr>
<td>
<code>codes</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ErrorCode">
[]ErrorCode
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Well-defined error codes in case the condition reports a problem.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.ErrorCode">ErrorCode
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.Condition">Condition</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.Error">Error</a>)
</p>
<p>
<p>ErrorCode is a string alias.</p>
</p>
<h3 id="landscaper.gardener.cloud/v1alpha1.ExecutionGeneration">ExecutionGeneration
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.ExecutionStatus">ExecutionStatus</a>)
</p>
<p>
<p>ExecutionGeneration links a deployitem to the generation of the execution when it was applied.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name is the name of the deployitem this generation refers to.</p>
</td>
</tr>
<tr>
<td>
<code>observedGeneration</code></br>
<em>
int64
</em>
</td>
<td>
<p>ObservedGeneration stores the generation which the execution had when it last applied the referenced deployitem.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.ExecutionPhase">ExecutionPhase
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.ExecutionStatus">ExecutionStatus</a>)
</p>
<p>
</p>
<h3 id="landscaper.gardener.cloud/v1alpha1.ExecutionSpec">ExecutionSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.Execution">Execution</a>)
</p>
<p>
<p>ExecutionSpec defines a execution plan.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>context</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Context defines the current context of the execution.</p>
</td>
</tr>
<tr>
<td>
<code>deployItems</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.DeployItemTemplateList">
DeployItemTemplateList
</a>
</em>
</td>
<td>
<p>DeployItems defines all execution items that need to be scheduled.</p>
</td>
</tr>
<tr>
<td>
<code>deployItemsCompressed</code></br>
<em>
[]byte
</em>
</td>
<td>
<p>DeployItemsCompressed as zipped byte array</p>
</td>
</tr>
<tr>
<td>
<code>registryPullSecrets</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ObjectReference">
[]ObjectReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>RegistryPullSecrets defines a list of registry credentials that are used to
pull blueprints, component descriptors and jsonschemas from the respective registry.
For more info see: <a href="https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/">https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/</a>
Note that the type information is used to determine the secret key and the type of the secret.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.ExecutionStatus">ExecutionStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.Execution">Execution</a>)
</p>
<p>
<p>ExecutionStatus contains the current status of a execution.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>observedGeneration</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>ObservedGeneration is the most recent generation observed for this Execution.
It corresponds to the Execution generation, which is updated on mutation by the landscaper.</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.Condition">
[]Condition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Conditions contains the actual condition of a execution</p>
</td>
</tr>
<tr>
<td>
<code>lastError</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.Error">
Error
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>LastError describes the last error that occurred.</p>
</td>
</tr>
<tr>
<td>
<code>exportRef</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ObjectReference">
ObjectReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ExportReference references the object that contains the exported values.
only used for operation purpose.</p>
</td>
</tr>
<tr>
<td>
<code>deployItemRefs</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.VersionedNamedObjectReference">
[]VersionedNamedObjectReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>DeployItemReferences contain the state of all deploy items.</p>
</td>
</tr>
<tr>
<td>
<code>execGenerations</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ExecutionGeneration">
[]ExecutionGeneration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ExecutionGenerations stores which generation the execution had when it last applied a specific deployitem.
So in this case, the observedGeneration refers to the executions generation.</p>
</td>
</tr>
<tr>
<td>
<code>jobID</code></br>
<em>
string
</em>
</td>
<td>
<p>JobID is the ID of the current working request.</p>
</td>
</tr>
<tr>
<td>
<code>jobIDFinished</code></br>
<em>
string
</em>
</td>
<td>
<p>JobIDFinished is the ID of the finished working request.</p>
</td>
</tr>
<tr>
<td>
<code>phase</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ExecutionPhase">
ExecutionPhase
</a>
</em>
</td>
<td>
<p>ExecutionPhase is the current phase of the execution.</p>
</td>
</tr>
<tr>
<td>
<code>phaseTransitionTime</code></br>
<em>
<a href="https://v1-22.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>PhaseTransitionTime is the time when the phase last changed.</p>
</td>
</tr>
<tr>
<td>
<code>transitionTimes</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.TransitionTimes">
TransitionTimes
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TransitionTimes contains timestamps of status transitions</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.ExportDefinition">ExportDefinition
</h3>
<p>
<p>ExportDefinition defines a exported value</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>FieldValueDefinition</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.FieldValueDefinition">
FieldValueDefinition
</a>
</em>
</td>
<td>
<p>
(Members of <code>FieldValueDefinition</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>type</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ExportType">
ExportType
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Type specifies which kind of object is being exported.
This field should be set and will likely be mandatory in future.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.ExportType">ExportType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.ExportDefinition">ExportDefinition</a>)
</p>
<p>
<p>ExportType is a string alias</p>
</p>
<h3 id="landscaper.gardener.cloud/v1alpha1.FailedReconcile">FailedReconcile
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.AutomaticReconcile">AutomaticReconcile</a>)
</p>
<p>
<p>FailedReconcile allows to configure automatically repeated reconciliations for failed installations</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>numberOfReconciles</code></br>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
<p>NumberOfReconciles specifies the maximal number of automatically repeated reconciliations. If not set, no upper
limit exists.</p>
</td>
</tr>
<tr>
<td>
<code>interval</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.Duration">
Duration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Interval specifies the interval between two subsequent repeated reconciliations. If not set, a default
of 5 minutes is used.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.FieldValueDefinition">FieldValueDefinition
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.ExportDefinition">ExportDefinition</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.ImportDefinition">ImportDefinition</a>)
</p>
<p>
<p>FieldValueDefinition defines a im- or exported field.
Either schema or target type have to be defined</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name defines the field name to search for the value and map to exports.
Ref: <a href="https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#selecting-fields">https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#selecting-fields</a></p>
</td>
</tr>
<tr>
<td>
<code>schema</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.JSONSchemaDefinition">
JSONSchemaDefinition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Schema defines the imported value as jsonschema.</p>
</td>
</tr>
<tr>
<td>
<code>targetType</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>TargetType defines the type of the imported target.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.ImportDefinition">ImportDefinition
</h3>
<p>
<p>ImportDefinition defines a imported value</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>FieldValueDefinition</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.FieldValueDefinition">
FieldValueDefinition
</a>
</em>
</td>
<td>
<p>
(Members of <code>FieldValueDefinition</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>type</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ImportType">
ImportType
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Type specifies which kind of object is being imported.
This field should be set and will likely be mandatory in future.</p>
</td>
</tr>
<tr>
<td>
<code>required</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Required specifies whether the import is required for the component to run.
Defaults to true.</p>
</td>
</tr>
<tr>
<td>
<code>default</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.Default">
Default
</a>
</em>
</td>
<td>
<p>Default sets a default value for the current import that is used if the key is not set.</p>
</td>
</tr>
<tr>
<td>
<code>imports</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ImportDefinitionList">
ImportDefinitionList
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ConditionalImports are Imports that are only valid if this imports is satisfied.
Does only make sense for optional imports.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.ImportStatus">ImportStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.InstallationStatus">InstallationStatus</a>)
</p>
<p>
<p>ImportStatus hold the state of a import.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name is the distinct identifier of the import.
Can be either from data or target imports</p>
</td>
</tr>
<tr>
<td>
<code>type</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ImportStatusType">
ImportStatusType
</a>
</em>
</td>
<td>
<p>Type defines the kind of import.
Can be either DataObject, Target, or TargetList</p>
</td>
</tr>
<tr>
<td>
<code>target</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Target is the name of the in-cluster target object.</p>
</td>
</tr>
<tr>
<td>
<code>targetList</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.TargetImportStatus">
[]TargetImportStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TargetList is a list of import statuses for in-cluster target objects.</p>
</td>
</tr>
<tr>
<td>
<code>dataRef</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>DataRef is the name of the in-cluster data object.</p>
</td>
</tr>
<tr>
<td>
<code>secretRef</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>SecretRef is the name of the secret.</p>
</td>
</tr>
<tr>
<td>
<code>configMapRef</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ConfigMapRef is the name of the imported configmap.</p>
</td>
</tr>
<tr>
<td>
<code>sourceRef</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ObjectReference">
ObjectReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>SourceRef is the reference to the installation from where the value is imported</p>
</td>
</tr>
<tr>
<td>
<code>configGeneration</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ConfigGeneration is the generation of the imported value.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.ImportStatusType">ImportStatusType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.ImportStatus">ImportStatus</a>)
</p>
<p>
<p>ImportStatusType defines the type of a import status.</p>
</p>
<h3 id="landscaper.gardener.cloud/v1alpha1.ImportType">ImportType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.ImportDefinition">ImportDefinition</a>)
</p>
<p>
<p>ImportType is a string alias</p>
</p>
<h3 id="landscaper.gardener.cloud/v1alpha1.InlineBlueprint">InlineBlueprint
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.BlueprintDefinition">BlueprintDefinition</a>)
</p>
<p>
<p>InlineBlueprint defines a inline blueprint with component descriptor and
filesystem.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>filesystem</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.AnyJSON">
AnyJSON
</a>
</em>
</td>
<td>
<p>Filesystem defines a inline yaml filesystem with a blueprint.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.InstallationExports">InstallationExports
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.InstallationTemplate">InstallationTemplate</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.InstallationSpec">InstallationSpec</a>)
</p>
<p>
<p>InstallationExports defines exports of data objects and targets.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>data</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.DataExport">
[]DataExport
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Data defines all data object exports.</p>
</td>
</tr>
<tr>
<td>
<code>targets</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.TargetExport">
[]TargetExport
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Targets defines all target exports.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.InstallationImports">InstallationImports
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.InstallationTemplate">InstallationTemplate</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.DeployerInstallationTemplate">DeployerInstallationTemplate</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.InstallationSpec">InstallationSpec</a>)
</p>
<p>
<p>InstallationImports defines import of data objects and targets.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>data</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.DataImport">
[]DataImport
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Data defines all data object imports.</p>
</td>
</tr>
<tr>
<td>
<code>targets</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.TargetImport">
[]TargetImport
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Targets defines all target imports.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.InstallationPhase">InstallationPhase
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.InstallationStatus">InstallationStatus</a>)
</p>
<p>
</p>
<h3 id="landscaper.gardener.cloud/v1alpha1.InstallationSpec">InstallationSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.Installation">Installation</a>)
</p>
<p>
<p>InstallationSpec defines a component installation.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>context</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Context defines the current context of the installation.</p>
</td>
</tr>
<tr>
<td>
<code>componentDescriptor</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ComponentDescriptorDefinition">
ComponentDescriptorDefinition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ComponentDescriptor is a reference to the installation&rsquo;s component descriptor</p>
</td>
</tr>
<tr>
<td>
<code>blueprint</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.BlueprintDefinition">
BlueprintDefinition
</a>
</em>
</td>
<td>
<p>Blueprint is the resolved reference to the definition.</p>
</td>
</tr>
<tr>
<td>
<code>registryPullSecrets</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ObjectReference">
[]ObjectReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>RegistryPullSecrets defines a list of registry credentials that are used to
pull blueprints, component descriptors and jsonschemas from the respective registry.
For more info see: <a href="https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/">https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/</a>
Note that the type information is used to determine the secret key and the type of the secret.</p>
</td>
</tr>
<tr>
<td>
<code>imports</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.InstallationImports">
InstallationImports
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Imports define the imported data objects and targets.</p>
</td>
</tr>
<tr>
<td>
<code>importDataMappings</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.AnyJSON">
map[string]..AnyJSON
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ImportDataMappings contains a template for restructuring imports.
It is expected to contain a key for every blueprint-defined data import.
Missing keys will be defaulted to their respective data import.
Example: namespace: (( installation.imports.namespace ))</p>
</td>
</tr>
<tr>
<td>
<code>exports</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.InstallationExports">
InstallationExports
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Exports define the exported data objects and targets.</p>
</td>
</tr>
<tr>
<td>
<code>exportDataMappings</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.AnyJSON">
map[string]..AnyJSON
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ExportDataMappings contains a template for restructuring exports.
It is expected to contain a key for every blueprint-defined data export.
Missing keys will be defaulted to their respective data export.
Example: namespace: (( blueprint.exports.namespace ))</p>
</td>
</tr>
<tr>
<td>
<code>automaticReconcile</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.AutomaticReconcile">
AutomaticReconcile
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>AutomaticReconcile allows to configure automatically repeated reconciliations.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.InstallationStatus">InstallationStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.Installation">Installation</a>)
</p>
<p>
<p>InstallationStatus contains the current status of a Installation.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>observedGeneration</code></br>
<em>
int64
</em>
</td>
<td>
<p>ObservedGeneration is the most recent generation observed for this ControllerInstallations.
It corresponds to the ControllerInstallations generation, which is updated on mutation by the landscaper.</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.Condition">
[]Condition
</a>
</em>
</td>
<td>
<p>Conditions contains the actual condition of a installation</p>
</td>
</tr>
<tr>
<td>
<code>lastError</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.Error">
Error
</a>
</em>
</td>
<td>
<p>LastError describes the last error that occurred.</p>
</td>
</tr>
<tr>
<td>
<code>configGeneration</code></br>
<em>
string
</em>
</td>
<td>
<p>ConfigGeneration is the generation of the exported values.</p>
</td>
</tr>
<tr>
<td>
<code>imports</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ImportStatus">
[]ImportStatus
</a>
</em>
</td>
<td>
<p>Imports contain the state of the imported values.</p>
</td>
</tr>
<tr>
<td>
<code>installationRefs</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.NamedObjectReference">
[]NamedObjectReference
</a>
</em>
</td>
<td>
<p>InstallationReferences contain all references to sub-components
that are created based on the component definition.</p>
</td>
</tr>
<tr>
<td>
<code>executionRef</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ObjectReference">
ObjectReference
</a>
</em>
</td>
<td>
<p>ExecutionReference is the reference to the execution that schedules the templated execution items.</p>
</td>
</tr>
<tr>
<td>
<code>jobID</code></br>
<em>
string
</em>
</td>
<td>
<p>JobID is the ID of the current working request.</p>
</td>
</tr>
<tr>
<td>
<code>jobIDFinished</code></br>
<em>
string
</em>
</td>
<td>
<p>JobIDFinished is the ID of the finished working request.</p>
</td>
</tr>
<tr>
<td>
<code>phase</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.InstallationPhase">
InstallationPhase
</a>
</em>
</td>
<td>
<p>InstallationPhase is the current phase of the installation.</p>
</td>
</tr>
<tr>
<td>
<code>phaseTransitionTime</code></br>
<em>
<a href="https://v1-22.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>PhaseTransitionTime is the time when the phase last changed.</p>
</td>
</tr>
<tr>
<td>
<code>importsHash</code></br>
<em>
string
</em>
</td>
<td>
<p>ImportsHash is the hash of the import data.</p>
</td>
</tr>
<tr>
<td>
<code>automaticReconcileStatus</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.AutomaticReconcileStatus">
AutomaticReconcileStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>AutomaticReconcileStatus describes the status of automatically triggered reconciles.</p>
</td>
</tr>
<tr>
<td>
<code>dependentsToTrigger</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.DependentToTrigger">
[]DependentToTrigger
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>DependentsToTrigger lists dependent installations to be triggered</p>
</td>
</tr>
<tr>
<td>
<code>transitionTimes</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.TransitionTimes">
TransitionTimes
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TransitionTimes contains timestamps of status transitions</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.InstallationTemplateBlueprintDefinition">InstallationTemplateBlueprintDefinition
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.InstallationTemplate">InstallationTemplate</a>)
</p>
<p>
<p>InstallationTemplateBlueprintDefinition contains either a reference to a blueprint or an inline definition.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ref</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Ref is a reference to a blueprint.
Only blueprints that are defined by the component descriptor of the current blueprint can be referenced here.
Example: cd://componentReference/dns/resources/blueprint</p>
</td>
</tr>
<tr>
<td>
<code>filesystem</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.AnyJSON">
AnyJSON
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Filesystem defines a virtual filesystem with all files needed for a blueprint.
The filesystem must be a YAML filesystem.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.JSONSchemaDefinition">JSONSchemaDefinition
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.Blueprint">Blueprint</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.FieldValueDefinition">FieldValueDefinition</a>)
</p>
<p>
<p>JSONSchemaDefinition defines a jsonschema.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>RawMessage</code></br>
<em>
<a href="https://pkg.go.dev/pkg/encoding/json/#RawMessage">
encoding/json.RawMessage
</a>
</em>
</td>
<td>
<p>
(Members of <code>RawMessage</code> are embedded into this type.)
</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.LocalConfigMapReference">LocalConfigMapReference
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.DataImport">DataImport</a>)
</p>
<p>
<p>LocalConfigMapReference is a reference to data in a configmap.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name is the name of the configmap</p>
</td>
</tr>
<tr>
<td>
<code>key</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Key is the name of the key in the configmap that holds the data.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.LocalSecretReference">LocalSecretReference
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.DataImport">DataImport</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.TargetSpec">TargetSpec</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.TargetSyncSpec">TargetSyncSpec</a>)
</p>
<p>
<p>LocalSecretReference is a reference to data in a secret.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name is the name of the secret</p>
</td>
</tr>
<tr>
<td>
<code>key</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Key is the name of the key in the secret that holds the data.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.LsHealthCheckStatus">LsHealthCheckStatus
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.LsHealthCheck">LsHealthCheck</a>)
</p>
<p>
</p>
<h3 id="landscaper.gardener.cloud/v1alpha1.NamedObjectReference">NamedObjectReference
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.InstallationStatus">InstallationStatus</a>)
</p>
<p>
<p>NamedObjectReference is a named reference to a specific resource.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name is the unique name of the reference.</p>
</td>
</tr>
<tr>
<td>
<code>ref</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ObjectReference">
ObjectReference
</a>
</em>
</td>
<td>
<p>Reference is the reference to an object.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.ObjectReference">ObjectReference
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.ConfigMapReference">ConfigMapReference</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.DeployItemSpec">DeployItemSpec</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.DeployItemStatus">DeployItemStatus</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.DeployItemTemplate">DeployItemTemplate</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.ExecutionSpec">ExecutionSpec</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.ExecutionStatus">ExecutionStatus</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.ImportStatus">ImportStatus</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.InstallationSpec">InstallationSpec</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.InstallationStatus">InstallationStatus</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.NamedObjectReference">NamedObjectReference</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.SecretReference">SecretReference</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.TargetImportStatus">TargetImportStatus</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.TargetSelector">TargetSelector</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.TypedObjectReference">TypedObjectReference</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.VersionedObjectReference">VersionedObjectReference</a>)
</p>
<p>
<p>ObjectReference is the reference to a kubernetes object.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name is the name of the kubernetes object.</p>
</td>
</tr>
<tr>
<td>
<code>namespace</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Namespace is the namespace of kubernetes object.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.OnDeleteConfig">OnDeleteConfig
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.DeployItemSpec">DeployItemSpec</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.DeployItemTemplate">DeployItemTemplate</a>)
</p>
<p>
<p>OnDeleteConfig specifies particular setting when deleting a deploy item</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>skipUninstallIfClusterRemoved</code></br>
<em>
bool
</em>
</td>
<td>
<p>SkipUninstallIfClusterRemoved specifies that uninstall is skipped if the target cluster is already deleted.
Works only in the context of an existing target sync object which is used to check the Garden project with
the shoot cluster resources</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.Operation">Operation
(<code>string</code> alias)</p></h3>
<p>
</p>
<h3 id="landscaper.gardener.cloud/v1alpha1.RemoteBlueprintReference">RemoteBlueprintReference
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.BlueprintDefinition">BlueprintDefinition</a>)
</p>
<p>
<p>RemoteBlueprintReference describes a reference to a blueprint defined by a component descriptor.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>resourceName</code></br>
<em>
string
</em>
</td>
<td>
<p>ResourceName is the name of the blueprint as defined by a component descriptor.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.Requirement">Requirement
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.TargetSelector">TargetSelector</a>)
</p>
<p>
<p>Requirement contains values, a key, and an operator that relates the key and values.
The zero value of Requirement is invalid.
Requirement implements both set based match and exact match
Requirement should be initialized via NewRequirement constructor for creating a valid Requirement.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>key</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>operator</code></br>
<em>
<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/selection#Operator">
k8s.io/apimachinery/pkg/selection.Operator
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>values</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>In huge majority of cases we have at most one value here.
It is generally faster to operate on a single-element slice
than on a single-element map, so we have a slice here.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.ResolvedTarget">ResolvedTarget
</h3>
<p>
<p>ResolvedTarget is a helper struct to store a target together with the content of its resolved secret reference.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>target</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.Target">
Target
</a>
</em>
</td>
<td>
<p>Target contains the original target.</p>
</td>
</tr>
<tr>
<td>
<code>content</code></br>
<em>
string
</em>
</td>
<td>
<p>Content contains the content of the target.
If the target has a secret reference, this field should be filled by a TargetResolver.
Otherwise, the inline configuration of the target is put here.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.ResourceReference">ResourceReference
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.VersionedResourceReference">VersionedResourceReference</a>)
</p>
<p>
<p>ResourceReference defines the reference to a resource defined in a component descriptor.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>componentName</code></br>
<em>
string
</em>
</td>
<td>
<p>ComponentName defines the unique of the component containing the resource.</p>
</td>
</tr>
<tr>
<td>
<code>resourceName</code></br>
<em>
string
</em>
</td>
<td>
<p>ResourceName defines the name of the resource.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.SecretLabelSelectorRef">SecretLabelSelectorRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.StaticDataValueFrom">StaticDataValueFrom</a>)
</p>
<p>
<p>SecretLabelSelectorRef selects secrets with the given label and key.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>selector</code></br>
<em>
map[string]string
</em>
</td>
<td>
<p>Selector is a map of labels to select specific secrets.</p>
</td>
</tr>
<tr>
<td>
<code>key</code></br>
<em>
string
</em>
</td>
<td>
<p>The key of the secret to select from.  Must be a valid secret key.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.SecretReference">SecretReference
</h3>
<p>
<p>SecretReference is reference to data in a secret.
The secret can also be in a different namespace.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ObjectReference</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ObjectReference">
ObjectReference
</a>
</em>
</td>
<td>
<p>
(Members of <code>ObjectReference</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>key</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Key is the name of the key in the secret that holds the data.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.StaticDataSource">StaticDataSource
</h3>
<p>
<p>StaticDataSource defines a static data source</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>value</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.AnyJSON">
AnyJSON
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Value defined inline a raw data</p>
</td>
</tr>
<tr>
<td>
<code>valueFrom</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.StaticDataValueFrom">
StaticDataValueFrom
</a>
</em>
</td>
<td>
<p>ValueFrom defines data from an external resource</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.StaticDataValueFrom">StaticDataValueFrom
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.BlueprintStaticDataSource">BlueprintStaticDataSource</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.StaticDataSource">StaticDataSource</a>)
</p>
<p>
<p>StaticDataValueFrom defines a static data that is read from a external resource.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>secretKeyRef</code></br>
<em>
<a href="https://v1-22.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#secretkeyselector-v1-core">
Kubernetes core/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Selects a key of a secret in the installations&rsquo;s namespace</p>
</td>
</tr>
<tr>
<td>
<code>secretLabelSelector</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.SecretLabelSelectorRef">
SecretLabelSelectorRef
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Selects a key from multiple secrets in the installations&rsquo;s namespace
that matches the given labels.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.SubinstallationTemplate">SubinstallationTemplate
</h3>
<p>
<p>SubinstallationTemplate defines a subinstallation template.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>file</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>File references a subinstallation template stored in another file.</p>
</td>
</tr>
<tr>
<td>
<code>InstallationTemplate</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.InstallationTemplate">
InstallationTemplate
</a>
</em>
</td>
<td>
<p>
(Members of <code>InstallationTemplate</code> are embedded into this type.)
</p>
<em>(Optional)</em>
<p>An inline subinstallation template.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.SucceededReconcile">SucceededReconcile
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.AutomaticReconcile">AutomaticReconcile</a>)
</p>
<p>
<p>SucceededReconcile allows to configure automatically repeated reconciliations for succeeded installations</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>interval</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.Duration">
Duration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Interval specifies the interval between two subsequent repeated reconciliations. If not set, a default of
24 hours is used.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.SyncObjectSpec">SyncObjectSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.SyncObject">SyncObject</a>)
</p>
<p>
<p>SyncObjectSpec contains the specification.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>podName</code></br>
<em>
string
</em>
</td>
<td>
<p>PodName describes the name of the pod of the responsible deployer</p>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
<em>
string
</em>
</td>
<td>
<p>Kind describes the kind of object that is being locked by this SyncObject</p>
</td>
</tr>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name is the name of the object that is being locked by this SyncObject</p>
</td>
</tr>
<tr>
<td>
<code>lastUpdateTime</code></br>
<em>
<a href="https://v1-22.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>LastUpdateTime contains last time the object was updated.</p>
</td>
</tr>
<tr>
<td>
<code>prefix</code></br>
<em>
string
</em>
</td>
<td>
<p>Prefix is the prefix of the name of the object.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.SyncObjectStatus">SyncObjectStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.SyncObject">SyncObject</a>)
</p>
<p>
<p>SyncObjectStatus contains the status.</p>
</p>
<h3 id="landscaper.gardener.cloud/v1alpha1.TLSClientConfig">TLSClientConfig
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.ClusterRestConfig">ClusterRestConfig</a>)
</p>
<p>
<p>TLSClientConfig contains settings to enable transport layer security</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>insecure</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Server should be accessed without verifying the TLS certificate. For testing only.</p>
</td>
</tr>
<tr>
<td>
<code>serverName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ServerName is passed to the server for SNI and is used in the client to check server
ceritificates against. If ServerName is empty, the hostname used to contact the
server is used.</p>
</td>
</tr>
<tr>
<td>
<code>caData</code></br>
<em>
[]byte
</em>
</td>
<td>
<em>(Optional)</em>
<p>CAData holds PEM-encoded bytes (typically read from a root certificates bundle).
CAData takes precedence over CAFile</p>
</td>
</tr>
<tr>
<td>
<code>nextProtos</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>NextProtos is a list of supported application level protocols, in order of preference.
Used to populate tls.Config.NextProtos.
To indicate to the server http/1.1 is preferred over http/2, set to <a href="though the server is free to ignore that preference">&ldquo;http/1.1&rdquo;, &ldquo;h2&rdquo;</a>.
To use only http/1.1, set to [&ldquo;http/1.1&rdquo;].</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.TargetExport">TargetExport
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.InstallationExports">InstallationExports</a>)
</p>
<p>
<p>TargetExport is a single target export.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name the internal name of the exported target.</p>
</td>
</tr>
<tr>
<td>
<code>target</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Target is the name of the in-cluster target object.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.TargetImport">TargetImport
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.InstallationImports">InstallationImports</a>)
</p>
<p>
<p>TargetImport is either a single target or a target list import.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name the internal name of the imported target.</p>
</td>
</tr>
<tr>
<td>
<code>target</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Target is the name of the in-cluster target object.
Exactly one of Target, Targets, and TargetListReference has to be specified.</p>
</td>
</tr>
<tr>
<td>
<code>targets</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Targets is a list of in-cluster target objects.
Exactly one of Target, Targets, and TargetListReference has to be specified.</p>
</td>
</tr>
<tr>
<td>
<code>targetListRef</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>TargetListReference can (only) be used to import a targetlist that has been imported by the parent installation.
Exactly one of Target, Targets, and TargetListReference has to be specified.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.TargetImportStatus">TargetImportStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.ImportStatus">ImportStatus</a>)
</p>
<p>
<p>TargetImportStatus</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>target</code></br>
<em>
string
</em>
</td>
<td>
<p>Target is the name of the in-cluster target object.</p>
</td>
</tr>
<tr>
<td>
<code>sourceRef</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ObjectReference">
ObjectReference
</a>
</em>
</td>
<td>
<p>SourceRef is the reference to the installation from where the value is imported</p>
</td>
</tr>
<tr>
<td>
<code>configGeneration</code></br>
<em>
string
</em>
</td>
<td>
<p>ConfigGeneration is the generation of the imported value.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.TargetSelector">TargetSelector
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.EnvironmentSpec">EnvironmentSpec</a>)
</p>
<p>
<p>TargetSelector describes a selector that matches specific targets.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>targets</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ObjectReference">
[]ObjectReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Targets defines a list of specific targets (name and namespace)
that should be reconciled.</p>
</td>
</tr>
<tr>
<td>
<code>annotations</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.Requirement">
[]Requirement
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Annotations matches a target based on annotations.</p>
</td>
</tr>
<tr>
<td>
<code>labels</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.Requirement">
[]Requirement
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Labels matches a target based on its labels.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.TargetSpec">TargetSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.Target">Target</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.TargetTemplate">TargetTemplate</a>)
</p>
<p>
<p>TargetSpec contains the definition of a target.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>type</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.TargetType">
TargetType
</a>
</em>
</td>
<td>
<p>Type is the type of the target that defines its data structure.
The actual schema may be defined by a target type crd in the future.</p>
</td>
</tr>
<tr>
<td>
<code>config</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.AnyJSON">
AnyJSON
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Configuration contains the target type specific configuration.
Exactly one of the fields Configuration and SecretRef must be set</p>
</td>
</tr>
<tr>
<td>
<code>secretRef</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.LocalSecretReference">
LocalSecretReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Reference to a secret containing the target type specific configuration.
Exactly one of the fields Configuration and SecretRef must be set</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.TargetSyncSpec">TargetSyncSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.TargetSync">TargetSync</a>)
</p>
<p>
<p>TargetSyncSpec contains the specification for a TargetSync.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>sourceNamespace</code></br>
<em>
string
</em>
</td>
<td>
<p>SourceNamespace describes the namespace from where the secrets should be synced</p>
</td>
</tr>
<tr>
<td>
<code>secretRef</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.LocalSecretReference">
LocalSecretReference
</a>
</em>
</td>
<td>
<p>SecretRef references the secret that contains the kubeconfig to the namespace of the secrets to be synced.</p>
</td>
</tr>
<tr>
<td>
<code>createTargetToSource</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>CreateTargetToSource specifies if set on true, that also a target is created, which references the secret in SecretRef</p>
</td>
</tr>
<tr>
<td>
<code>targetToSourceName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>TargetToSourceName is the name of the target referencing the secret defined in SecretRef if CreateTargetToSource
is set on true. If TargetToSourceName is empty SourceNamespace is used instead.</p>
</td>
</tr>
<tr>
<td>
<code>secretNameExpression</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>SecretNameExpression defines the names of the secrets which should be synced via a regular expression according
to <a href="https://github.com/google/re2/wiki/Syntax">https://github.com/google/re2/wiki/Syntax</a> with the extension that * is also a valid expression and matches
all names.
if not set no secrets are synced</p>
</td>
</tr>
<tr>
<td>
<code>shootNameExpression</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ShootNameExpression defines the names of shoot clusters for which targets with short living access data
to the shoots are created via a regular expression according to <a href="https://github.com/google/re2/wiki/Syntax">https://github.com/google/re2/wiki/Syntax</a> with
the extension that * is also a valid expression and matches all names.
if not set no targets for the shoots are created</p>
</td>
</tr>
<tr>
<td>
<code>tokenRotation</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.TokenRotation">
TokenRotation
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TokenRotation defines the data to perform an automatic rotation of the token to access the source cluster with the
secrets to sync. The token expires after 90 days and will be rotated every 60 days.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.TargetSyncStatus">TargetSyncStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.TargetSync">TargetSync</a>)
</p>
<p>
<p>TargetSyncStatus contains the status of a TargetSync.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>observedGeneration</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>ObservedGeneration is the most recent generation observed.</p>
</td>
</tr>
<tr>
<td>
<code>lastUpdateTime</code></br>
<em>
<a href="https://v1-22.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Last time the status was updated</p>
</td>
</tr>
<tr>
<td>
<code>lastErrors</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>LastErrors describe the last errors</p>
</td>
</tr>
<tr>
<td>
<code>lastTokenRotationTime</code></br>
<em>
<a href="https://v1-22.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Last time the token was rotated</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.TargetTemplate">TargetTemplate
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.EnvironmentSpec">EnvironmentSpec</a>)
</p>
<p>
<p>TargetTemplate exposes specific parts of a target that are used in the exports
to export a target</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>TargetSpec</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.TargetSpec">
TargetSpec
</a>
</em>
</td>
<td>
<p>
(Members of <code>TargetSpec</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>labels</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Map of string keys and values that can be used to organize and categorize
(scope and select) objects. May match selectors of replication controllers
and services.
More info: <a href="http://kubernetes.io/docs/user-guide/labels">http://kubernetes.io/docs/user-guide/labels</a></p>
</td>
</tr>
<tr>
<td>
<code>annotations</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Annotations is an unstructured key value map stored with a resource that may be
set by external tools to store and retrieve arbitrary metadata. They are not
queryable and should be preserved when modifying objects.
More info: <a href="http://kubernetes.io/docs/user-guide/annotations">http://kubernetes.io/docs/user-guide/annotations</a></p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.TargetType">TargetType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.TargetSpec">TargetSpec</a>)
</p>
<p>
<p>TargetType defines the type of the target.</p>
</p>
<h3 id="landscaper.gardener.cloud/v1alpha1.TemplateExecutor">TemplateExecutor
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.Blueprint">Blueprint</a>)
</p>
<p>
<p>TemplateExecutor describes a templating mechanism and configuration.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name is the unique name of the template</p>
</td>
</tr>
<tr>
<td>
<code>type</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.TemplateType">
TemplateType
</a>
</em>
</td>
<td>
<p>Type describes the templating mechanism.</p>
</td>
</tr>
<tr>
<td>
<code>file</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>File is the path to the template in the blueprint&rsquo;s content.</p>
</td>
</tr>
<tr>
<td>
<code>template</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.AnyJSON">
AnyJSON
</a>
</em>
</td>
<td>
<p>Template contains an optional inline template.
The template has to be of string for go template
and either a string or valid yaml/json for spiff.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.TemplateType">TemplateType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.TemplateExecutor">TemplateExecutor</a>)
</p>
<p>
<p>TemplateType describes the template mechanism.</p>
</p>
<h3 id="landscaper.gardener.cloud/v1alpha1.TokenRotation">TokenRotation
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.TargetSyncSpec">TargetSyncSpec</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>enabled</code></br>
<em>
bool
</em>
</td>
<td>
<p>Enabled defines if automatic token is executed</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.TransitionTimes">TransitionTimes
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.DeployItemStatus">DeployItemStatus</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.ExecutionStatus">ExecutionStatus</a>, 
<a href="#landscaper.gardener.cloud/v1alpha1.InstallationStatus">InstallationStatus</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>triggerTime</code></br>
<em>
<a href="https://v1-22.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TriggerTime is the time when the jobID is set.</p>
</td>
</tr>
<tr>
<td>
<code>initTime</code></br>
<em>
<a href="https://v1-22.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>InitTime is the time when the Init phase starts.</p>
</td>
</tr>
<tr>
<td>
<code>waitTime</code></br>
<em>
<a href="https://v1-22.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>WaitTime is the time when the work is done.</p>
</td>
</tr>
<tr>
<td>
<code>finishedTime</code></br>
<em>
<a href="https://v1-22.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>FinishedTime is the time when the finished phase is set.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.TypedObjectReference">TypedObjectReference
</h3>
<p>
<p>TypedObjectReference is a reference to a typed kubernetes object.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
<em>
string
</em>
</td>
<td>
<p>APIVersion is the group and version for the resource being referenced.
If APIVersion is not specified, the specified Kind must be in the core API group.
For any other third-party types, APIVersion is required.</p>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
<em>
string
</em>
</td>
<td>
<p>Kind is the type of resource being referenced</p>
</td>
</tr>
<tr>
<td>
<code>ObjectReference</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ObjectReference">
ObjectReference
</a>
</em>
</td>
<td>
<p>
(Members of <code>ObjectReference</code> are embedded into this type.)
</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.VersionedNamedObjectReference">VersionedNamedObjectReference
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.ExecutionStatus">ExecutionStatus</a>)
</p>
<p>
<p>VersionedNamedObjectReference is a named reference to a object with its last observed resource generation.
This struct is used by status fields.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name is the unique name of the reference.</p>
</td>
</tr>
<tr>
<td>
<code>ref</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.VersionedObjectReference">
VersionedObjectReference
</a>
</em>
</td>
<td>
<p>Reference is the reference to an object.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.VersionedObjectReference">VersionedObjectReference
</h3>
<p>
(<em>Appears on:</em>
<a href="#landscaper.gardener.cloud/v1alpha1.VersionedNamedObjectReference">VersionedNamedObjectReference</a>)
</p>
<p>
<p>VersionedObjectReference is a reference to a object with its last observed resource generation.
This struct is used by status fields.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ObjectReference</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ObjectReference">
ObjectReference
</a>
</em>
</td>
<td>
<p>
(Members of <code>ObjectReference</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>observedGeneration</code></br>
<em>
int64
</em>
</td>
<td>
<p>ObservedGeneration defines the last observed generation of the referenced resource.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="landscaper.gardener.cloud/v1alpha1.VersionedResourceReference">VersionedResourceReference
</h3>
<p>
<p>VersionedResourceReference defines the reference to a resource with its version.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ResourceReference</code></br>
<em>
<a href="#landscaper.gardener.cloud/v1alpha1.ResourceReference">
ResourceReference
</a>
</em>
</td>
<td>
<p>
(Members of <code>ResourceReference</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>version</code></br>
<em>
string
</em>
</td>
<td>
<p>Version defines the version of the component.</p>
</td>
</tr>
</tbody>
</table>
<hr/>
