#!/bin/bash
#
# Copyright (c) 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

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
