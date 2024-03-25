#!/usr/bin/env python3

# SPDX-FileCopyrightText: 2024 "SAP SE or an SAP affiliate company and Gardener contributors"
#
# SPDX-License-Identifier: Apache-2.0

import tempfile
import yaml

from util import ctx

factory = ctx().cfg_factory()
landscape_kubeconfig = factory.kubernetes("landscaper-integration-test")

with tempfile.NamedTemporaryFile(mode="w+", prefix="gardener_serviceaccount_kubeconfig_", suffix=".yaml", delete=False) as kubeconfig_temp_file:
    kubeconfig_temp_file.write(yaml.safe_dump(landscape_kubeconfig.kubeconfig()))
    landscape_kubeconfig_path = kubeconfig_temp_file.name

    print(landscape_kubeconfig_path)
