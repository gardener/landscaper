## Credential Rotation for Landscaper as a Service (LaaS)

This document describes a proposal how to automatically rotate the credentials used by the LaaS.

Credential rotation could not be done completely automatically. There must be some initial credentials which has to be 
rotated manually. This proposal requires only one kubeconfig as root credentials which have to be rotated 
manually.

Remark: According to the Gardener security recommendations we assume that all involved Clusters (LaaS Clusters, 
Landscaper Instances Clusters, Customer Shoot Clusters) must be switched to non-static secrets. This proposal does
not require static credentials for shoot clusters. 

### Root Credentials

As root credentials, which needs to be rotated manually, we use the kubeconfig of a robots service account with name 
`root-account` for the [LaaS Gardener project](https://dashboard.garden.canary.k8s.ondemand.com/namespace/garden-hubforplay/members), 
with the `Admin` role.  This service account does not have the role `Service Account Manager` to request new  tokens, because 
old token could not be invalidated. If this service account is allowed to request new token and one token leaks, an 
attacker could create new token forever and our rotation would never invalidate these.

You could get the kubeconfig of the service account `root-account` from the dashboard. The contained token has only
a very short validity. To request a new token with a longer validity of 90 day, either at the beginning or later 
when the token must be rotated, you require another service account `root-rotator` with the role `Service Account Manager`.

The new token is requested by the 
[token request API](https://kubernetes.io/docs/reference/kubernetes-api/authentication-resources/token-request-v1/#TokenRequest)
with the following command:

```
kubectl create --kubeconfig=<path to root rotator kubeconfig> \
      --raw "/api/v1/namespaces/garden-hubforplay/serviceaccounts/root-account/token" \
      -f <(echo '{"spec":{"expirationSeconds": 7776000}}') \
      | jq -r .status
```

The kubeconfig of the service account `root-account` with the new or rotated token is stored in cc-config. The token must
be rotated manually about every 90 days by a member of the 
[LaaS Gardener project](https://dashboard.garden.canary.k8s.ondemand.com/namespace/garden-hubforplay/members). 
Before a rotation is executed the user has to delete and recreate the service account `root-account`. This prevents that 
a leaked token for this service account could be replaced by new one forever.

The pipeline jobs have access to the kubeconfig of service account `root-account` and thereby admin access to the
Garden namespace of the laas project.

### GCR Secret Rotation

The creation of shoot clusters requires a secret key for a GCR service account. This is currently stored in 
the cc-config and must be rotated manually
([see](https://github.tools.sap/kubernetes/k8s-lifecycle-management/blob/master/docs/LaaS/credential-rotation.md#gardener-secrets)).

This document proposes the following automatic rotation of this secret key.  First of all we will use service keys of 
a service account in GCR with the right to create and delete service account keys. Furthermore, we do not store the 
service key in the cc-config anymore. It only exists in a secret of the Garden namespace of the laas project.

During every pipeline deployment job we fetch the secret with the GCR service key, create a new one, and replace it in the 
secret. Then the pipeline job deletes/invalidates all other secret keys of the GCR service account.

If a secret key leaks, the next rotation invalidates it, and it is not possible to read with a leaked key, the newly 
created keys stored in the Gardener project namespace.

Perhaps we need to work with 2 secret keys with overlapping live time to prevent some downtime. This should not change the 
principal approach.

### Access to LaaS Core and Target Clusters during pipeline deployment job

During a deployment pipeline job it is required to access the LaaS core clusters to deploy e.g. the LaaS service. Probably 
also some deployments to the target clusters are required. Such access is provided as follows: 

- The pipeline job requests short living token (live time is maximal 24 hours) to the shoots 
  ([see](https://github.com/gardener/gardener/blob/master/docs/usage/shoot_access.md))
  using the kubeconfig of the service account `root-account`.

It is not required anymore to store the kubeconfigs to access shoot clusters in the cc-config anymore.

### Allow Laas to create and access customer shoot cluster (Landscaper resource cluster)

After removing Virtual Garden for the Landscaper instances, the LaaS must be able to create, maintain and delete
shoot clusters in the laas Gardener project. The required credentials are the root credentials, i.e. the 
kubeconfig of the service account `root-account` which is provided during the deployment of LaaS.

Disadvantage: The root credentials must be stored outside cc-config which increases the risk to leak. Fortunately 
the credentials to rotate them are not stored somewhere.

### Access to Target and Customer Clusters during runtime

During runtime, applications requires admin access to another cluster with token of a larger live time
than 24 hours. An example is the LaaS on the core cluster which requires access to the target clusters and the customer 
clusters. 

For simplicity every application which requires admin access to some other cluster X gets the root credentials during
deployment. To get admin access to some other cluster it proceeds as follows:

- Application fetches a short living token to the cluster X with the root credentials
  ([see](https://github.com/gardener/gardener/blob/master/docs/usage/shoot_access.md))
- short living token is used to create admin service account on cluster X
- short living token is used to create/update token/kubeconfig for admin service account on cluster X with live time
  of 90 days using the
  [token request API](https://kubernetes.io/docs/reference/kubernetes-api/authentication-resources/token-request-v1/#TokenRequest).
  The service account on cluster X should not have the right to refresh the token by itself. 

Remark: All other access data to cluster X must be also rotated. Other access data might be additional service accounts
beside the admin service account. Not sure if there is some general concept for this. 

### Questions

- Is it a good idea to have all clusters (dev, canary, live) in one Gardener project? Thereby it is not possible to separate 
the access and we are probably violating security standards. On the other hand, if we use several Gardener projects
we will have several root credentials, which needs to be rotated manually.

- If an admin token to some cluster of the root credentials leaks, an attacker could create its backdoor which is not 
  closed after any rotation? Such a backdoor could consist e.g. of a further admin service account with the right to 
  rotate its own token. Without deleting it, this will provide access forever. If the root credentials are leaking, an 
  attacker can create a backdoor on all currently existing clusters. The only solution for this problem seems to be that
  it is not allowed to create service accounts on any cluster which allows to create their own token. It must only be 
  possible to create a new token for a shoot cluster sarting at the root credentials. If these are rotated, no leaked 
  access is possible anymore.

- Because in the worst case the root credentials were leaked, we require a solution which is immediately replacing and 
  invalidating any access. This seems to be only achievable by replacing all service accounts (delete and create with 
  the same name) on all levels. Hereby it is required that a leaked secret could not be used to create a new access
  during old access invalidation, which is not an atomic operation. 
