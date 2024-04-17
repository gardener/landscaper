# Signature Verification

Landscaper can verify the signature of a component version and only continue with the installation if the signature is valid and matches the content.

> **IMPORTANT**: In the current state, the landscaper can only verify the signature and content when the installation is created/modified. Later controllers such as deployer will access the resources WITHOUT further verification. This implies that a potential malicious manipulation between creating the installation (and verifying the signature) and subsequent resource access can NOT be detected/prevented. Therefore, it is advisable to use this feature with a trusted repository only.

## Requirement

Verification only works when using OCM lib and OCM components. This is enabled as useOCMLib in the `landscaper-config.yaml`:
```yaml
useOCMLib: true
```

## Enable Verification of Component Versions

### Landscaper Config Signature Verification Enforcement Policy

In the landscaper config, a verification enforcement policy can be specified:
```yaml
useOCMLib: true
signatureVerificationEnforcementPolicy: DoNotEnforce # DoNotEnforce(DEFAULT)|Enforce|Disabled
```
The following values are possible:
- **DoNotEnforce** (DEFAULT): does not enforce a landscaper-wide policy. Each installation can specify a `spec.verification` key to enable the verification for it.
- **Enforce**: enforces all installations to be verified and therefore all installations need the `spec.verification` key with provided verification options.
- **Disabled**: explicitly disables all verifications for all installations even if an installation provides verification information with `spec.verification` key. **It is advised to use this option for debugging only.**

## Configure Verification

Necessary verification information is provided in the installation and the context.
In the installation, `spec.verification.signatureName` needs to be set to the name of the signature in the context that is also the sginature name in the component version:
```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: hello-world
  namespace: default

spec:
  context: default-context
# ...
  verification: 
    signatureName: signature-name
```
Depending on the global signatureVerificationEnforcementPolicy configuration in the landscaper-config, this only enables the verification if the signatureVerificationEnforcementPolicy is `DoNotEnforce` or `Enforce`. In case of `Disabled`, specifying the `spec.verification` key with signature name will not lead to signature verification.

The provided signatureName is defined in the Context to specify a public key or a certificate-authority certificate as a secret reference:
```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Context
metadata:
  name: default-context
  namespace: default
# ...

useOCM: true

verificationSignatures:
  acme-sig:
    caCertificateSecretReference:
      name: secret-cacert
      namespace: default
      key: key
# OR
    publicKeySecretReference:
      name: secret-publickey
      namespace: default
      key: key
```
The referenced secret contains the PEM encoded public key or ca certificate. Those can be generated using openssl.

Which option to use depends on how the component version was signed. The `publicKeySecretReference` maps to the `--public-key` of the ocm cli command `ocm verify` and the `caCertificateSecretReference` maps to the `ca-cert` option.
If the signature field of the component version also contains a certificate of the signer, the `caCertificateSecretReference` with the issuer certificate of the signer. 

Example secret containing a public key:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: secret-publickey
  namespace: default
type: Opaque
stringData:
  key: |
    -----BEGIN RSA PUBLIC KEY-----
    ...
    -----END RSA PUBLIC KEY-----

```