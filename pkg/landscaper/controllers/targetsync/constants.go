// SPDX-FileCopyrightText: 2022 "SAP SE or an SAP affiliate company and Gardener contributors"
//
// SPDX-License-Identifier: Apache-2.0

package targetsync

import (
	"time"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

const (
	requeueInterval              = 5 * time.Minute
	tokenRotationInterval        = 60 * 24 * 60 * 60 * time.Second
	tokenExpirationSeconds int64 = 90 * 24 * 60 * 60
	labelKeyTargetSync           = lsv1alpha1.LandscaperDomain + "/targetsync"
	labelValueOk                 = "ok"
)
