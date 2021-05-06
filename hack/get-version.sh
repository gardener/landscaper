#!/bin/bash
#
# Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# SPDX-License-Identifier: Apache-2.0

set -e

if [ -n "$TM_GIT_REF" ] ; then
  # running e2e test in a release job, use TM_GIT_REF as image tag (is set to git release tag name)
  echo "$TM_GIT_REF"
  exit 0
fi

if [ -n "$EFFECTIVE_VERSION" ] ; then
  # running in the pipeline use the provided EFFECTIVE_VERSION
  echo "$EFFECTIVE_VERSION"
  exit 0
fi

SOURCE_PATH="$(dirname $0)/.."
VERSION="$(cat "${SOURCE_PATH}/VERSION")"

pushd ${SOURCE_PATH} > /dev/null 2>&1

if [ -n "$TM_GIT_SHA" ] ; then
  # running e2e test for a PR, calculate image tag by concatenating VERSION and commit sha.
  echo "$VERSION-$TM_GIT_SHA"
  exit 0
fi

if [[ "$VERSION" = *-dev ]] ; then
  VERSION="$VERSION-$(git rev-parse HEAD)"
fi

popd > /dev/null 2>&1

echo "$VERSION"
