// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package deployitems

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/gardener/landscaper/test/framework"
)

// RegisterTests registers all tests of this package
func RegisterTests(f *framework.Framework) {
	PickupTimeoutTests(f)
	ProgressingTimeoutTests(f)
	AbortingTimeoutTests(f)
}

const (
	waitingForDeployItems     = 5 * time.Second  // how long to wait for the landscaper to create deploy items from the installation
	deployItemPickupTimeout   = 10 * time.Second // the landscaper has to be configured accordingly for this test to work!
	deployItemAbortingTimeout = 10 * time.Second // the landscaper has to be configured accordingly for this test to work!
	waitingForReconcile       = 10 * time.Second // how long to wait for the landscaper or the deployer to reconcile and update the deploy item
	resyncTime                = 1 * time.Second  // after which time to check again if the condition was not fulfilled the last time
	retryUpdates              = 5 * time.Second  // for how long update operations are retried if they failed
)

func namespacedName(meta metav1.ObjectMeta) types.NamespacedName {
	return types.NamespacedName{
		Namespace: meta.Namespace,
		Name:      meta.Name,
	}
}
