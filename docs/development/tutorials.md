# Adding new tutorials

Most of the present tutorials provide additional resources that should be provided on Gardeners GCR repository - after all, following through the tutorials can be a lot easier if users can rely on already present component-descriptors and blueprints in GCR that they can simply reuse.

To make uploading the resources simple, reliable and repeatable, we provided the script `hack/upload-tutorial-resources.sh`.

# Using upload-tutorial-resources.sh

## Component descriptors

The script features an array `component_descriptors` which contains the pathes to all component descriptors that should get uploaded to a registry. Elements are seperated by newlines. Since the repository-context and version information are part of the component-descriptor, all it takes in this array is really just the path to the directory containing the `component-descriptor.yaml` file.

If you add a new tutorial resource with a component-descriptor, make sure you add it to this array.

## Blueprints

Like for component-descriptors, the script has an array `blueprints`... which is a bit more complicated this time.

Elements are seperated by newlines, but each element consist of three fields, seperated by semi-cola `;`. The first field is the repository path, the second field is the local path to the directory containing the `blueprint.yaml`. The third field is a version number which will be used if there is no version number in the `blueprint.yaml` itself.

Example:

```
"eu.gcr.io/gardener-project/landscaper/tutorials/blueprints/echo-server;./docs/tutorials/resources/echo-server/blueprint;v0.1.1"
```

This line will upload the Blueprint found in `./docs/tutorials/resources/echo-server/blueprint` as an OCI artifact to `eu.gcr.io/gardener-project/landscaper/tutorials/blueprints/echo-server:v0.1.1`.

### Blueprint version numbers

Blueprints contain no version information - this is explicitely not part of the Blueprint specification. Thus, the version tag must be provided in the array described above or it can be provided in the `blueprint.yaml` file.

Just to make developing tutorial resources easier (i.e. if you change an existing Blueprint, you do not have to change the `upload-tutorial-resources.sh` script), the script will look into the `blueprint.yaml` for a comment `# TUTORIAL_BLUEPRINT_VERSION: v0.2.0` and use this version tag to upload the Blueprint to the OCI registry.

**WARNING:** This is only meant to simplify the development of tutorial resources. It is not part of the Blueprint specification and MUST NOT be used in production.
