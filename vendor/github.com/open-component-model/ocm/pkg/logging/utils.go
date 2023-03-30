// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package logging

func ErrorMessage(err error) *string {
	if err == nil {
		return nil
	}
	m := err.Error()
	return &m
}
