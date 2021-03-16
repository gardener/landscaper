# Terraform Deployer

The Terraform deployer is a controller that reconciles
DeployItems of type `landscaper.gardener.cloud/terraform`.

It manages infrastructure resources using terraform.
This means running the `apply` and `destroy` terraform commands.

To run these commands, it relies on the Gardener component
[Terraformer](https://github.com/gardener/terraformer)

### Configuration

This sections describes the provider specific configuration

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: DeployItem
metadata:
  name: s3-buckets
spec:
  type: landscaper.gardener.cloud/terraform

  config:
    apiVersion: terraform.deployer.landscaper.gardener.cloud/v1alpha1
    kind: ProviderConfiguration

    # Main Terraform configuration.
    main.tf: |
      provider "aws" {
        access_key = var.AWS_ACCESS_KEY_ID
        secret_key = var.AWS_SECRET_ACCESS_KEY
        region     = "eu-central-1"
      }
      resource "aws_s3_bucket" "a" {
        bucket = var.bucket_a
        acl    = "private"
      }
      resource "aws_s3_bucket" "b" {
        bucket = var.bucket_b
        acl    = "private"
      }
      output "bucket_names" {
        value = [aws_s3_bucket.a.bucket, aws_s3_bucket.b.bucket]
      }

    # Terraform variables.
    # optional
    variables.tf: |
      variable "bucket_a" {}
      variable "bucket_b" {}
      variable "AWS_ACCESS_KEY_ID" {}
      variable "AWS_SECRET_ACCESS_KEY" {}

    # Terraform input variables.
    # optional
    terraform.tfvars: |
      bucket_a = "bucket-a"
      bucket_b = "bucket-b"
      AWS_ACCESS_KEY_ID     = "AKIAIOSFODNN7EXAMPLE"
      AWS_SECRET_ACCESS_KEY = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
      
    envVars:
    - name: AWS_ACCESS_KEY_ID
      value: "AKIAIOSFODNN7EXAMPLE"
    - name: AWS_SECRET_ACCESS_KEY # see "use targets" for detailed explanation
      fromTarget:
        jsonPath: ".secretAccessKey"
        
    files:
    - name: my-file
      value: "" # base64 encoded value
    - name: my-other-file # see "use targets" for detailed explanation
      fromTarget:
        jsonPath: ".kubeconfig"

    providers:
    - inline: "provider-aws | provider-azure" # will use in-tree providers.
      ref: https://releases.hashicorp.com/terraform-provider-aws/3.32.0/terraform-provider-aws_3.32.0_linux_amd64.zip # url to the zipped provider.
      fromResource: # will fetch the provider from component descriptor resource of type hashicorp.com/terraform-provider
        ref:
          repositoryContext:
            type: ociRegistry
            baseUrl: my-repo
          componentName: github.com/gardener/landscaper
          version: v0.3.0
        resourceName: aws-provider
```

:warning: Note that the terraform image does not include any provider.
In order use specific providers, the needed providers have to be specified as resources.

Inline Providers:
- provider-aws

#### Use targets

Targets mostly include information how a target can be accessed.
In the case of the terraformer any kind of target is possible as terraform also supports different providers.

To provide a generic approach to work with target configuration and make it accessible to the terraform it is possible to specify environment variables or files from targets.

*Env Vars*:
Given a aws target:
```yaml
type: landscaper.gardener.cloud/v1alpha1
kind: Target
spec:
  type: landsacper.gardener.cloud/aws-credentials
  config:
    accessKeyId: "AKIAIOSFODNN7EXAMPLE"
    secretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
```
the secrets can be added to terraform using the environment configuration.
Environment variables can be configured using the `fromTarget` attribut with a json path that can access the target's config attribute.
```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: DeployItem
metadata:
  name: s3-buckets
spec:
  type: landscaper.gardener.cloud/terraform

  config:
    apiVersion: terraform.deployer.landscaper.gardener.cloud/v1alpha1
    kind: ProviderConfiguration

    ...

    envVars:
    - name: AWS_ACCESS_KEY_ID
      fromTarget:
        jsonPath: ".accessKeyId"
    - name: AWS_SECRET_ACCESS_KEY
      fromTarget:
        jsonPath: ".secretAccessKey"
```

*Files*:
Given a kubernetes cluster target:
```yaml
type: landscaper.gardener.cloud/v1alpha1
kind: Target
spec:
  type: landsacper.gardener.cloud/kubernetes-cluster
  config:
    kubeconfig: |
      apiVersion: v1
      kind: Config
      ...
```
the secrets can be added to terraform using the files configuration.
Files can be configured using the `fromTarget` attribut with a json path that can access the target's config attribute.
```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: DeployItem
metadata:
  name: deploy-something
spec:
  type: landscaper.gardener.cloud/terraform

  config:
    apiVersion: terraform.deployer.landscaper.gardener.cloud/v1alpha1
    kind: ProviderConfiguration

    ...

    files:
    - name: kubeconfig # will be mounted to "/ls/files/kubeconfig" - "/ls/files" is the default path for mounted files; "kubeconfig" is the name of the file.
      fromTarget:
        jsonPath: ".kubeconfig"
```

### Status

This section describes the provider specific status of the resource.

This shows the output of `terraform output` in `json`.

```yaml
providerStatus:
  apiVersion: terraform.deployer.landscaper.gardener.cloud/v1alpha1
  kind: ProviderStatus
  output:
    bucket_names:
      type:
      - tuple
      - - string
        - string
      value:
      - bucket-a
      - bucket-b
```


```bash
$ kubectl get deployitem s3-buckets -ojsonpath='{.status.providerStatus}' | jq '.'
{
  "apiVersion": "terraform.deployer.landscaper.gardener.cloud/v1alpha1",
  "kind": "ProviderStatus",
  "output": {
    "bucket_names": {
      "type": [
        "tuple",
        [
          "string",
          "string"
        ]
      ],
      "value": [
        "bucket-a-from-tfvars",
        "bucket-b-from-env-secret"
      ]
    }
  }
}
```
