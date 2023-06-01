// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"net/url"
	"strings"
)

func ParseURL(urlToParse string) (*url.URL, error) {
	const dummyScheme = "dummy://"
	if !strings.Contains(urlToParse, "://") {
		urlToParse = dummyScheme + urlToParse
	}
	parsedURL, err := url.Parse(urlToParse)
	if err != nil {
		return nil, err
	}
	if parsedURL.Scheme == dummyScheme {
		parsedURL.Scheme = ""
	}
	return parsedURL, nil
}
