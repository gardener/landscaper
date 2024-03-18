#!/bin/bash
#
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

# DOCKERFILE_DIR is the path to the directory containing the Dockerfile.
DOCKERFILE_DIR=$1
# IMAGE_NAME is the name of the container image.
IMAGE_NAME=$2
# IMAGE_VERSION is the version of the container image
IMAGE_VERSION=$3
# OCI_REPO is the OCI repository to which the container image is pushed.
OCI_REPO=$4

function print_usage_and_exit {
  >&2 echo "Wrong number of arguments:"
  >&2 echo "  USAGE: $0 DOCKERFILE_DIR IMAGE_NAME IMAGE_VERSION OCI_REPO"
  exit 1
}

if [ -z "$DOCKERFILE_DIR" ]; then
  print_usage_and_exit
fi

if [ -z "$IMAGE_NAME" ]; then
  print_usage_and_exit
fi

if [ -z "$IMAGE_VERSION" ]; then
  print_usage_and_exit
fi

if [ -z "$OCI_REPO" ]; then
  print_usage_and_exit
fi

docker build -t "${OCI_REPO}/${IMAGE_NAME}:${IMAGE_VERSION}" --platform amd64 $DOCKERFILE_DIR
docker push "${OCI_REPO}/${IMAGE_NAME}:${IMAGE_VERSION}"
