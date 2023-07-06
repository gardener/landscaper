// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package realhelmdeployer

import (
	"encoding/json"
	"time"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	lserror "github.com/gardener/landscaper/apis/errors"
)

const (
	defaultTimeout = 5 * time.Minute
)

// installConfiguration defines settings for a helm install operation.
type installConfiguration struct {
	Atomic  bool                 `json:"atomic,omitempty"`
	Timeout *lsv1alpha1.Duration `json:"timeout,omitempty"`
}

func newInstallConfiguration(conf *helmv1alpha1.HelmDeploymentConfiguration) (*installConfiguration, error) {
	currOp := "NewInstallConfiguration"

	installConf := &installConfiguration{}

	if conf != nil && len(conf.Install) > 0 {
		rawConf, err := json.Marshal(conf.Install)
		if err != nil {
			return nil, lserror.NewWrappedError(err, currOp, "MarshalConfig", err.Error())
		}

		if err := json.Unmarshal(rawConf, installConf); err != nil {
			return nil, lserror.NewWrappedError(err, currOp, "UnmarshalConfig", err.Error())
		}
	}

	// set defaults
	if installConf.Timeout == nil {
		installConf.Timeout = &lsv1alpha1.Duration{Duration: defaultTimeout}
	}

	return installConf, nil
}

// upgradeConfiguration defines settings for a helm upgrade operation.
type upgradeConfiguration struct {
	Atomic  bool                 `json:"atomic,omitempty"`
	Timeout *lsv1alpha1.Duration `json:"timeout,omitempty"`
}

func newUpgradeConfiguration(conf *helmv1alpha1.HelmDeploymentConfiguration) (*upgradeConfiguration, error) {
	currOp := "NewUpgradeConfiguration"

	upgradeConf := &upgradeConfiguration{}

	if conf != nil && len(conf.Upgrade) > 0 {
		rawConf, err := json.Marshal(conf.Upgrade)
		if err != nil {
			return nil, lserror.NewWrappedError(err, currOp, "MarshalConfig", err.Error())
		}

		if err := json.Unmarshal(rawConf, upgradeConf); err != nil {
			return nil, lserror.NewWrappedError(err, currOp, "UnmarshalConfig", err.Error())
		}
	}

	// set defaults
	if upgradeConf.Timeout == nil {
		upgradeConf.Timeout = &lsv1alpha1.Duration{Duration: defaultTimeout}
	}

	return upgradeConf, nil
}

// uninstallConfiguration defines settings for a helm uninstall operation.
type uninstallConfiguration struct {
	Timeout *lsv1alpha1.Duration `json:"timeout,omitempty"`
}

func newUninstallConfiguration(conf *helmv1alpha1.HelmDeploymentConfiguration) (*uninstallConfiguration, error) {
	currOp := "NewUninstallConfiguration"

	uninstallConf := &uninstallConfiguration{}

	if conf != nil && len(conf.Uninstall) > 0 {
		rawConf, err := json.Marshal(conf.Uninstall)
		if err != nil {
			return nil, lserror.NewWrappedError(err, currOp, "MarshalConfig", err.Error())
		}

		if err := json.Unmarshal(rawConf, uninstallConf); err != nil {
			return nil, lserror.NewWrappedError(err, currOp, "UnmarshalConfig", err.Error())
		}
	}

	// set defaults
	if uninstallConf.Timeout == nil {
		uninstallConf.Timeout = &lsv1alpha1.Duration{Duration: defaultTimeout}
	}

	return uninstallConf, nil
}
