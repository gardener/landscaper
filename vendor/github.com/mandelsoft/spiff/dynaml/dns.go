/*
Copyright 2014 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package dynaml

import (
	"fmt"
	"regexp"
)

// MaxLenError returns a string explanation of a "string too long" validation
// failure.
func MaxLenError(length int) string {
	return fmt.Sprintf("must be no more than %d characters", length)
}

// RegexError returns a string explanation of a regex validation failure.
func RegexError(msg string, fmt string, examples ...string) string {
	if len(examples) == 0 {
		return msg + " (regex used for validation is '" + fmt + "')"
	}
	msg += " (e.g. "
	for i := range examples {
		if i > 0 {
			msg += " or "
		}
		msg += "'" + examples[i] + "', "
	}
	msg += "regex used for validation is '" + fmt + "')"
	return msg
}

// EmptyError returns a string explanation of a "must not be empty" validation
// failure.
func EmptyError() string {
	return "must be non-empty"
}

const dns1123LabelFmt string = "[a-z0-9]([-a-z0-9]*[a-z0-9])?"
const dns1123LabelErrMsg string = "a DNS-1123 label must consist of lower case alphanumeric characters or '-', and must start and end with an alphanumeric character"
const DNS1123LabelMaxLength int = 63

var dns1123LabelRegexp = regexp.MustCompile("^" + dns1123LabelFmt + "$")

// IsDNS1123Label tests for a string that conforms to the definition of a label in
// DNS (RFC 1123).
func IsDNS1123Label(value string) []string {
	var errs []string
	if len(value) > DNS1123LabelMaxLength {
		errs = append(errs, MaxLenError(DNS1123LabelMaxLength))
	}
	if !dns1123LabelRegexp.MatchString(value) {
		errs = append(errs, RegexError(dns1123LabelErrMsg, dns1123LabelFmt, "my-name", "123-abc"))
	}
	return errs
}

const dns1123SubdomainFmt string = dns1123LabelFmt + "(\\." + dns1123LabelFmt + ")*"
const dns1123SubdomainErrorMsg string = "a DNS-1123 subdomain must consist of lower case alphanumeric characters, '-' or '.', and must start and end with an alphanumeric character"
const DNS1123SubdomainMaxLength int = 253

var dns1123SubdomainRegexp = regexp.MustCompile("^" + dns1123SubdomainFmt + "$")

// IsDNS1123Subdomain tests for a string that conforms to the definition of a
// subdomain in DNS (RFC 1123).
func IsDNS1123Subdomain(value string) []string {
	var errs []string
	if len(value) > DNS1123SubdomainMaxLength {
		errs = append(errs, MaxLenError(DNS1123SubdomainMaxLength))
	}
	if !dns1123SubdomainRegexp.MatchString(value) {
		errs = append(errs, RegexError(dns1123SubdomainErrorMsg, dns1123SubdomainFmt, "example.com"))
	}
	return errs
}

const dns1035LabelFmt string = "[a-z]([-a-z0-9]*[a-z0-9])?"
const dns1035LabelErrMsg string = "a DNS-1035 label must consist of lower case alphanumeric characters or '-', start with an alphabetic character, and end with an alphanumeric character"
const DNS1035LabelMaxLength int = 63

var dns1035LabelRegexp = regexp.MustCompile("^" + dns1035LabelFmt + "$")

// IsDNS1035Label tests for a string that conforms to the definition of a label in
// DNS (RFC 1035).
func IsDNS1035Label(value string) []string {
	var errs []string
	if len(value) > DNS1035LabelMaxLength {
		errs = append(errs, MaxLenError(DNS1035LabelMaxLength))
	}
	if !dns1035LabelRegexp.MatchString(value) {
		errs = append(errs, RegexError(dns1035LabelErrMsg, dns1035LabelFmt, "my-name", "abc-123"))
	}
	return errs
}

// wildcard definition - RFC 1034 section 4.3.3.
// examples:
// - valid: *.bar.com, *.foo.bar.com
// - invalid: *.*.bar.com, *.foo.*.com, *bar.com, f*.bar.com, *
const wildcardDNS1123SubdomainFmt = "\\*\\." + dns1123SubdomainFmt
const wildcardDNS1123SubdomainErrMsg = "a wildcard DNS-1123 subdomain must start with '*.', followed by a valid DNS subdomain, which must consist of lower case alphanumeric characters, '-' or '.' and end with an alphanumeric character"

// IsWildcardDNS1123Subdomain tests for a string that conforms to the definition of a
// wildcard subdomain in DNS (RFC 1034 section 4.3.3).
func IsWildcardDNS1123Subdomain(value string) []string {
	wildcardDNS1123SubdomainRegexp := regexp.MustCompile("^" + wildcardDNS1123SubdomainFmt + "$")

	var errs []string
	if len(value) > DNS1123SubdomainMaxLength {
		errs = append(errs, MaxLenError(DNS1123SubdomainMaxLength))
	}
	if !wildcardDNS1123SubdomainRegexp.MatchString(value) {
		errs = append(errs, RegexError(wildcardDNS1123SubdomainErrMsg, wildcardDNS1123SubdomainFmt, "*.example.com"))
	}
	return errs
}
