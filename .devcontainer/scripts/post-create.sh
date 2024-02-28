#!/usr/bin/env bash

set -e

# run once on post-create and it will be always restarted automatically when the container restarts
if [ $( curl https://localhost:443 )  ]; then #TODO improve detection of already running container
  echo "docker registry already exist"
else
  echo "docker registry does not exist yet, start registry with ssl..."
  
  docker run -d -p 443:443 --restart always --name registry \
  -v /registry/certs/:/certs \
  -e REGISTRY_HTTP_ADDR=:443 \
  -e REGISTRY_HTTP_TLS_CERTIFICATE=/certs/ociregistry.crt \
  -e REGISTRY_HTTP_TLS_KEY=/certs/ociregistry.key \
  registry:2
fi

minikube start

# apply CRDs to minikube
kubectl apply -f /workspaces/landscaper/.crd/

# create sample ocm componentversion and add it to registry
ocm add components --create --file /workspaces/landscaper/.devcontainer/sample-installation/sample-component /workspaces/landscaper/.devcontainer/sample-installation/components.yaml
ocm transfer ctf /workspaces/landscaper/.devcontainer/sample-installation/sample-component OCIRegistry::localhost:443