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

echo "Build minikube target and apply to cluster"
# creates a target.landscaper.gardener.cloud by combining the target.yaml with an indended minikube kubeconfig. Uses <() as process substitution to use command output as a file for kubectl
kubectl apply -f <(cat .devcontainer/target-template.yaml;  cat ~/.kube/kubeconfig--minikube-local.yaml | sed 's/^/      /')

echo "Apply context to cluster"
kubectl apply -f /workspaces/landscaper/.devcontainer/context.yaml