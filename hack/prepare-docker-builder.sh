#!/bin/bash
#
# Copyright (c) 2023 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# SPDX-License-Identifier: Apache-2.0

set -e

DOCKER_BUILDER_NAME=${1:-"ls-multiarch"}

if ! docker buildx ls | grep "$DOCKER_BUILDER_NAME" >/dev/null; then
  echo "Creating docker builder '$DOCKER_BUILDER_NAME' ..."
  docker buildx create --name "$DOCKER_BUILDER_NAME"
fi