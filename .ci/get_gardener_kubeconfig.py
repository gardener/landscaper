#!/usr/bin/env python3

# SPDX-FileCopyrightText: 2024 "SAP SE or an SAP affiliate company and Gardener contributors"
#
# SPDX-License-Identifier: Apache-2.0

import utils
import yaml

from util import ctx

factory = ctx().cfg_factory()
landscape_kubeconfig = factory.kubernetes("landscaper-integration-test")

with utils.TempFileAuto(prefix="gardener_serviceaccount_kubeconfig_") as kubeconfig_temp_file:
    kubeconfig_temp_file.write(yaml.safe_dump(landscape_kubeconfig.kubeconfig()))
    landscape_kubeconfig_path = kubeconfig_temp_file.switch()

    print(landscape_kubeconfig_path)
