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
  name: s3-bucket
spec:
  type: landscaper.gardener.cloud/terraform

  target: # has to be of type landscaper.gardener.cloud/kubernetes-cluster
    name: my-cluster
    namespace: test

  config:
    apiVersion: terraform.deployer.landscaper.gardener.cloud/v1alpha1
    kind: ProviderConfiguration

    # Use a specific image for the Terraformer
    # optional
    terraformerImage: eu.gcr.io/gardener-project/gardener/terraformer:v1.5.0

    # base64 encoded kubeconfig pointing to the cluster to install the chart
    # optional if Target is defined.
    kubeconfig: <redacted>

    # Namespace where the terraformer pod will run and where it will
    # store the main.tf, variables.tf as ConfigMap and terraform.tfvars as Secret.
    # The secrets below must exist in this namespace.
    # option - default to default namespace.
    namespace: default

    # Secrets names containing environment variables which will be
    # added to the Terraformer pod.
    # optional
    envSecrets:
      - aws          # infrastructure secret with AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY
      - application  # application secret with TF_VAR_bucket_b

    # Main Terraform configuration.
    main.tf: |
      provider "aws" {
        region = "eu-central-1"
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

    # Terraform input variables.
    # optional
    terraform.tfvars: |
      bucket_a = "bucket-a-from-tfvars"
```

The `DeployItem` uses the two following secrets:

```yaml
apiVersion: v1
data:
  AWS_ACCESS_KEY_ID: QUtJQUlPU0ZPRE5ON0VYQU1QTEUK
  AWS_SECRET_ACCESS_KEY: d0phbHJYVXRuRkVNSS9LN01ERU5HL2JQeFJmaUNZRVhBTVBMRUtFWQo=
kind: Secret
metadata:
  name: aws
---
apiVersion: v1
data:
  TF_VAR_bucket_b: YnVja2V0LWItZnJvbS1lbnYtc2VjcmV0
kind: Secret
metadata:
  name: application
```

Which results by the terraformer pod to have these available through `env`:

```yaml
env:
- name: AWS_ACCESS_KEY_ID
  valueFrom:
    secretKeyRef:
      key: AWS_ACCESS_KEY_ID
      name: aws
- name: AWS_SECRET_ACCESS_KEY
  valueFrom:
    secretKeyRef:
      key: AWS_SECRET_ACCESS_KEY
      name: aws
- name: TF_VAR_bucket_b
  valueFrom:
    secretKeyRef:
      key: TF_VAR_bucket_b
      name: application
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
      - bucket-a-from-tfvars
      - bucket-b-from-env-secret
```


```bash
$ kubectl get deployitem s3-bucket -ojsonpath='{.status.providerStatus}' | jq '.'
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
