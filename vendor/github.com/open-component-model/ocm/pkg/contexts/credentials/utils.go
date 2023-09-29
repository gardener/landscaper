// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package credentials

func GetProvidedConsumerId(obj interface{}, uctx ...UsageContext) ConsumerIdentity {
	if p, ok := obj.(ConsumerIdentityProvider); ok {
		return p.GetConsumerId(uctx...)
	}
	return nil
}

func GetProvidedIdentityMatcher(obj interface{}, uctx ...UsageContext) string {
	if p, ok := obj.(ConsumerIdentityProvider); ok {
		return p.GetIdentityMatcher()
	}
	return ""
}
