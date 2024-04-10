#!/bin/bash
#
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0


# Prerequisite: you must login with the following command
# gcloud auth login

set -o errexit

component_dir="$(dirname $0)/.."
temp_dir=$(mktemp -d)

echo "Packaging helm chart"
helm package "${component_dir}/chart/hello-world" -d "${temp_dir}"

echo "Login"
gcloud auth print-access-token | helm registry login -u oauth2accesstoken --password-stdin https://eu.gcr.io

echo "Pushing helm chart"
helm push "${temp_dir}/hello-world-1.0.0.tgz" oci://eu.gcr.io/gardener-project/landscaper/examples/charts
