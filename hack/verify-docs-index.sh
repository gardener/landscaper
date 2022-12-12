#!/bin/bash
#
# Copyright (c) 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# SPDX-License-Identifier: Apache-2.0

set -e

CURRENT_DIR=$(dirname $0)
PROJECT_ROOT="${CURRENT_DIR}"/..

echo "> Checking if documentation index needs changes"
doc_index_file="$PROJECT_ROOT/docs/README.md"
tmp_compare_file=$(mktemp)
"$CURRENT_DIR/generate-docs-index.sh" "$tmp_compare_file" >/dev/null
if ! cmp -s "$doc_index_file" "$tmp_compare_file"; then
  echo "The documentation index requires changes."
  echo "Please run 'make generate-docs' to update it."
  exit 1
fi
echo "Documentation index is up-to-date."