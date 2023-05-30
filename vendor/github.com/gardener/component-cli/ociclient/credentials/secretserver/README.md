# SecretServer 

The secret server auth implements the retrieval of secrets using the gardener secrets server defined in https://github.com/gardener/cc-utils/blob/master/ccc/secrets_server.py#L29 .

### Contract

1. Get Secret Server endpoint from env var `SECRETS_SERVER_ENDPOINT`
2. Get secret key and cipher from the env var `SECRET_KEY` and `SECRET_CIPHER_ALGORITHM`
  - The secret key is base64 encoded
3. Decrypt complete config json using the given secret key and cipher
