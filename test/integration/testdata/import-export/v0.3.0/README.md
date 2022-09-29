# Import Export Component - Version v0.3.0

This version is almost equal to version `v0.1.0`.
The only difference is in the DeployExecution, which writes a different entry into the deployed ConfigMap
(`fooUpdated: barUpdated` instead of `foo: bar`). In this way we can test an update of a DeployItem.
