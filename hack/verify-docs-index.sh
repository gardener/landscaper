#!/bin/bash
#
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

PROJECT_ROOT="$(realpath $(dirname $0)/..)"

echo "> Checking if documentation index needs changes"
doc_index_file="$PROJECT_ROOT/docs/README.md"
tmp_compare_file=$(mktemp)
"$PROJECT_ROOT/hack/generate-docs-index.sh" "$tmp_compare_file" >/dev/null
if ! cmp -s "$doc_index_file" "$tmp_compare_file"; then
  echo "The documentation index requires changes."
  echo "Please run 'make generate-docs' to update it."
  exit 1
fi
echo "Documentation index is up-to-date."