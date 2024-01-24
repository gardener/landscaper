// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package signutils

import (
	"crypto/x509/pkix"
	"regexp"
	"sort"
	"strings"

	"github.com/open-component-model/ocm/pkg/errors"
)

func CommonName(n string) *pkix.Name {
	return &pkix.Name{CommonName: n}
}

var (
	dnRegexp    = regexp.MustCompile(`[/;,+]([^=]+)=([^/;,+]+)`)
	allDNRegexp = regexp.MustCompile(`^[^=]+=[^/;,+]+([/;,+][^=]+=[^/;,+]+)*$`)
)

func ParseDN(dn string) (*pkix.Name, error) {
	name := pkix.Name{}

	dn = strings.TrimSpace(dn)
	if len(dn) == 0 {
		return nil, errors.ErrInvalid("distinguished name", dn)
	}
	if !allDNRegexp.MatchString(dn) {
		name.CommonName = dn
	} else {
		matches := dnRegexp.FindAllStringSubmatch("+"+dn, -1)
		for _, match := range matches {
			val := match[2]
			if val == "" {
				continue
			}

			switch match[1] {
			case "C":
				name.Country = append(name.Country, val)
			case "O":
				name.Organization = append(name.Organization, val)
			case "OU":
				name.OrganizationalUnit = append(name.OrganizationalUnit, val)
			case "L":
				name.Locality = append(name.Locality, val)
			case "ST":
				name.Province = append(name.Province, val)
			case "STREET":
				name.StreetAddress = append(name.StreetAddress, val)
			case "POSTALCODE":
				name.PostalCode = append(name.PostalCode, val)
			case "SN":
				name.SerialNumber = val
			case "CN":
				name.CommonName = val
			default:
				return nil, errors.ErrInvalid("attribute", match[1])
			}
		}
	}

	return &name, nil
}

func NormalizeDN(dn pkix.Name) string {
	sort.Strings(dn.StreetAddress)
	sort.Strings(dn.Locality)
	sort.Strings(dn.OrganizationalUnit)
	sort.Strings(dn.Organization)
	sort.Strings(dn.Country)
	sort.Strings(dn.Province)
	sort.Strings(dn.PostalCode)
	return DNAsString(dn)
}

func DNAsString(dn pkix.Name) string {
	s := dn.String()
	if len(s) == 3+len(dn.CommonName) {
		return dn.CommonName
	}
	return s
}

func MatchDN(n pkix.Name, p pkix.Name) error {
	if p.CommonName != "" && n.CommonName != p.CommonName {
		return errors.ErrInvalid("common name", n.CommonName)
	}
	if len(p.Country) != 0 {
		if err := containsAll("country", n.Country, p.Country); err != nil {
			return err
		}
	}
	if len(p.Province) != 0 {
		if err := containsAll("province", n.Province, p.Province); err != nil {
			return err
		}
	}
	if len(p.Locality) != 0 {
		if err := containsAll("locality", n.Locality, p.Locality); err != nil {
			return err
		}
	}
	if len(p.PostalCode) != 0 {
		if err := containsAll("postal code", n.PostalCode, p.PostalCode); err != nil {
			return err
		}
	}
	if len(p.StreetAddress) != 0 {
		if err := containsAll("street address", n.StreetAddress, p.StreetAddress); err != nil {
			return err
		}
	}
	if len(p.Organization) != 0 {
		if err := containsAll("organization", n.Organization, p.Organization); err != nil {
			return err
		}
	}
	if len(p.OrganizationalUnit) != 0 {
		if err := containsAll("organizational unit", n.OrganizationalUnit, p.OrganizationalUnit); err != nil {
			return err
		}
	}
	return nil
}

func containsAll(key string, n []string, p []string) error {
loop:
	for _, ps := range p {
		for _, ns := range n {
			if ns == ps {
				continue loop
			}
		}
		return errors.ErrNotFound(key, ps)
	}
	return nil
}
