#!/usr/bin/env python
#
# Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# SPDX-License-Identifier: Apache-2.0

set -e

import yaml

CURRENT_DIR=$(dirname $0)
PROJECT_ROOT="${CURRENT_DIR}"/..

sed 's/\s*type: Any/ {}/g' -E ${PROJECT_ROOT}/pkg/landscaper/crdmanager/crdresources/* | grep -C 4 additionalItems:
