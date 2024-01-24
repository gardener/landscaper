## Signing Service

The type `rsa-signingservice` forwards the signing to a 
signing server. The calculated digest is sent as signing request together with
the used hash algorithm

The URL of the signing service is passed YAML document instead of a
private key.

It must has the field `url` with the desired server address.

The required credentials are taken from the credentials context
using the consumer id `Signingserver.gardener.cloud`.
If uses a hostpath matcher using the identity attrutes `scheme`, `hostname`,
`port` and `pathprefix` derived from the given server URL.

The expected credential properties are:
- **`clientCert`**: the client certificate used as TLS certificate and 
  to authenticate the caller.
- **`privateKey`**: the private key for the client certificate.
- **`caCerts`**: the CA used to validate the identity of the signining server.