#!/usr/bin/env bash

set -e

# TODO check if this is necessary or if it will be automatically started
if [ $( docker ps -a -f name=minikube | wc -l ) -eq 2 ]; then
  echo "minikube already started"
else
  echo "minikube not started yet, creating..."
  minikube start
fi

echo "Exporting minikube config to yaml file"
kubectl config view --raw > ~/.kube/kubeconfig--minikube-local.yaml
