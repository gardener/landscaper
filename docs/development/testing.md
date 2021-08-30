# Testing

This document describes how tests in the Landscaper project can be executed, how they can be written and how the execution in the pipelines work.

**Index:**
- [Unit tests](#unit-tests)
- [Integration tests](#integration-tests)

### Unit Tests

#### execute
Unit tests are completely written in golang and therefore can all be executed and debugged using the standard go tools.

The `make test` command should be used to run all unit tests as the integration tests are also written and go and would be executed if someone would just run `go test ./...`.

#### write tests

Unit tests are written in golang using the [ginkgo](https://onsi.github.io/ginkgo/) and [gomega](https://onsi.github.io/gomega/) testframework.

**Kubernetes Tests**<br>
Kubernetes specific tests can either be written using the [controller-runtime testframework](../../test/utils/envtest) or using the Landscaper integration tests.

Tests written with the controller-runtime framework can be executed locally without a running kubernetes. With the disadvantage that controller-runtime only offers a k8s api-server and an etcd. If other Kubernetes capabilities (especially the controllers or cloud capabilities) are needed, the test has to be written as Landscaper integration test.
But the controller-runtime framework based tests should be always favored over real integration tests due to their simplicity in execution and debugging.

Util functions can be found in [test/utils]() and new utils should be also added there.
The difference to the default landscaper utils package is that here also `ginkgo` or `gomega` functions can be used which are not allowed in the default utils package.

### Integration tests

Integration tests are tests that use a k8s-conformance-compliant cluster and optional a real oci-compliant registry.

#### execution

The tests are by default executed on every head-update of the main git branch and can be optionally executed on a PR by commenting that PR with `/test`.
This will trigger the TestMachinery bot and will execute the integration tests for the PR in the TestMachinery.

The TestMachinery itself works with 
- TestDefinitions that define a single test step (find them in [.test-defs](../../.test-defs))
- and Testruns that define the step execution order and configures the tests (find the one testrun in [.ci/testruns/integration-test](../../.ci/testruns/integration-test/templates/testrun.yaml))

![TestMachinery test setup](../images/TestMachineryITSetup.png)

**local execution**
The integration tests can also be executed locally by providing a kubernetes cluster and export the kubeconfig as `export KUBECONFIG=<path to kubeconfig>`.
Then just run the tests with `make integration-test`.

By default the integration tests that require an oci registry executed with the make command are skipped. (These tests are also flagged as `Require OCI Registry` see https://github.com/gardener/landscaper/blob/master/test/integration/core/registry.go#L37)

Test that require an oci registry can be executed locally by
1. Provide an oci-registry.<br>
   The landscaper project offers you to deploy a local registry by running `make setup-local-registry` (a running k8s cluster is needed). Please follow all instructions printed by the command.
   > Note: The registry configuration is written to `./tmp/local-docker.config`.<br>
   > The registry can be deleted with `make remove-local-registry`
2. running the integration tests with the oci registry configuration as `make integration-test REGISTRY_CONFIG=<path to config>`

#### write tests

The integration tests are all combined in a single ginkgo test suite that can be found in [test/integration/suite_test.go](../../test/integration/suite_test.go).
That suite initializes the integration test framework (includes logging, other environment information) and then registers the different tests from different test files.

All tests are located in [test/integration/](../../test/integration/) where the tests can be described using the default ginkgo's `ginkgo.Describe|Context|IT` syntax.
Each `Describe` block should also first call the frameworks `Setup()` function to register logging and cleanup steps.
> Note: Ginkgo is also here used as executor so all ginkgo features like `Focus` or other should work as in other tests.

With that an integration test will look like the following snippet.
For a real example see [test/integration/tutorial/simple-import.go](../../test/integration/tutorial/simple-import.go) where also other helper functions like the `State` struct and others can be found.
```
func SimpleTest(f *framework.Framework) {
	_ = ginkgo.Describe("SimpleTest", func() {
		state := f.Register()
		
		ginkgo.It("some test", func(){
		 // test code ...
		})
	})
}
```
