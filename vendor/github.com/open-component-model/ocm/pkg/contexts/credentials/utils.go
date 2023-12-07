// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package credentials

import (
	"github.com/open-component-model/ocm/pkg/utils"
)

func GetProvidedConsumerId(obj interface{}, uctx ...UsageContext) ConsumerIdentity {
	return utils.UnwrappingCall(obj, func(provider ConsumerIdentityProvider) ConsumerIdentity {
		return provider.GetConsumerId()
	})
}

func GetProvidedIdentityMatcher(obj interface{}) string {
	return utils.UnwrappingCall(obj, func(provider ConsumerIdentityProvider) string {
		return provider.GetIdentityMatcher()
	})
}
