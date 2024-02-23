#!/usr/bin/env bash

set -e

# run once on post-create and it will be always restarted automatically when the container restarts
if [ $( docker ps -a -f name=registry | wc -l ) -eq 2 ]; then
  echo "docker registry already exist"
else
  echo "docker registry does not exist yet, creating..."
  docker run -d -p 5000:5000 --restart always --name registry registry:2
fi

minikube start

# apply CRDs to minikube
kubectl apply -f /workspaces/landscaper/.crd/

