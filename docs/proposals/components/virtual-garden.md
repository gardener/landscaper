# Virtual Garden Contract

### Import json
```yaml
{
  "backup": {
    "enabled": false,
    "blobstore": {
      "region": "europe-west5",
       "providerConfig": {} # azure needs subscription
    },
    "credentialsRef": "my-cred", 
    "credentials": {
      "my-cred": {
          "type": "gcp",
          "data": {
            "serviceaccount.json": "{\"privKey: ....}"
          }
      },
    }
  },
  "kubeconfig": {
    "apiVersion": "v1",
    "...":  "..."
  },
  "namespace": "default",
  "domain": "api.dev.gardener.cloud",
  "dnsClass": "host",

  "networkpolicies": false,
  
  "virtual": {
    "vpa": {
      "enabled": false,
    }
  },

  "auditlog": {
    "enabled": false,
    "kubeconfig":  {
      "apiVersion": "v1",
      "...":  "..."
    },
    "policy": {} # optional 
  },
  
  # to be discussed
  "identity": { 
    "enabled": false,
    "issuerURL": "my-url",
    "cert": "", # ca of identity
    "api": {
      "endpoint": "", # url to the grpc endpoint of dex
      "clientKey": "" # private key to authenticate against the dex grpc
    }
  }

}
```

### Export json

```yaml
{
  "virtualAdminKubeconfig": {
    "apiVersion": "v1"
  },
  "virtual": {
    "etcd": {
      "endpoints": {
        "main": "url to etcd main",
        "events": "url to etcd events"
      },
      "cert": {
        "ca": "",
        "crt": "",
        "key": "",
      }   
    },

    "apiserver": {
      "endpoints": {
        "internal": "internal url to apiserver"
      },
      "cert": {
        "ca": "",
        "crt": "",
        "key": "",
      }  
    },
  },
}
```
