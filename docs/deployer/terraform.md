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
```

### Status

This section describes the provider specific status of the resource.

This shows the output of `terraform ouput` in `json`.

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
