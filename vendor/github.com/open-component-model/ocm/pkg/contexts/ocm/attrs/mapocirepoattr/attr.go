// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package mapocirepoattr

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/exp/maps"

	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/contexts/oci/grammar"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/runtime"
	"github.com/open-component-model/ocm/pkg/utils"
)

const (
	ATTR_KEY   = "github.com/mandelsoft/ocm/mapocirepo"
	ATTR_SHORT = "mapocirepo"
)

func init() {
	datacontext.RegisterAttributeType(ATTR_KEY, AttributeType{}, ATTR_SHORT)
}

type AttributeType struct{}

func (a AttributeType) Name() string {
	return ATTR_KEY
}

func (a AttributeType) Description() string {
	return `
*bool|YAML*
When uploading an OCI artifact blob to an OCI based OCM repository and the
artifact is uploaded as OCI artifact, the repository path part is shortened,
either by hashing all but the last repository name part or by executing
some prefix based name mappings.

If a boolean is given the short hash or none mode is enabled.
The YAML flavor uses the following fields:
- *<code>mode</code>* *string*: <code>hash</code>, <code>shortHash</code>, <code>prefixMapping</code>
  or <code>none</code>. If unset, no mapping is done.
- *<code>prefixMappings</code>*: *map[string]string* repository path prefix mapping.
- *<code>prefix</code>*: *string* repository prefix to use (replaces potential sub path of OCM repo).
  or <code>none</code>.
- *<code>prefixMapping</code>*: *map[string]string* repository path prefix mapping.

Notes:

- The mapping only occurs in transfer commands and only when transferring to OCI registries (e.g.
  when transferring to a CTF archive this option will be ignored).
- The mapping only happens for local resources. When external image references are transferred (with
  option --copy-resources) the mapping will be ignored.
- The mapping in mode <code>prefixMapping</code> requires a full prefix of the composed final name.
  Partial matches are not supported. The host name of the target will be skipped.
- The artifact name of the component-descriptor is not mapped.
- If the mapping is provided on the command line it must be JSON format and needs to be properly
  escaped (see example below).

Example:

Assume a component named <code>github.com/my_org/myexamplewithalongname</code> and a chart name
<code>echo</code> in the <code>Charts.yaml</code> of the chart archive. The following input to a
<code>resource.yaml</code> creates a component version:

<pre>
name: mychart
type: helmChart
input:
  type: helm
  path: charts/mychart.tgz
---
name: myimage
type: ociImage
version: 0.1.0
input:
  type: ociImage
  repository: ocm/ocm.software/ocmcli/ocmcli-image
  path: ghcr.io/acme/ocm/ocm.software/ocmcli/ocmcli-image:0.1.0
</pre>

The following command:

<pre>
ocm "-X mapocirepo={\"mode\":\"mapping\",\"prefixMappings\":{\"acme/github.com/my_org/myexamplewithalongname/ocm/ocm.software/ocmcli\":\"acme/cli\", \"acme/github.com/my_org/myexamplewithalongnameabc123\":\"acme/mychart\"}}" transfer ctf -f --copy-resources ./ctf ghcr.io/acme
</pre>

will result in the following artifacts in <code>ghcr.io/my_org</code>:

<pre>
mychart/echo
cli/ocmcli-image
</pre>

Note that the host name part of the transfer target <code>ghcr.io/acme</code> is excluded from the
prefix but the path <code>acme</code> is considered.

The same using a config file <code>.ocmconfig</code>:
<pre>
type: generic.config.ocm.software/v1
configurations:
...
- type: attributes.config.ocm.software
  attributes:
	...
	mapocirepo:
	  mode: mapping
	  prefixMappings:
	    acme/github.com/my\_org/myexamplewithalongname/ocm/ocm.software/ocmcli: acme/cli
		acme/github.com/my\_org/myexamplewithalongnameabc123: acme/mychart
</pre>

<pre>
ocm transfer ca -f --copy-resources ./ca ghcr.io/acme
</pre>
`
}

func (a AttributeType) Encode(v interface{}, marshaller runtime.Marshaler) ([]byte, error) {
	if _, ok := v.(bool); ok {
		return marshaller.Marshal(&Attribute{Mode: ShortHashMode})
	}

	if _, ok := v.(*Attribute); ok {
		return marshaller.Marshal(v)
	}

	return nil, fmt.Errorf("boolean or attribute struct required")
}

func (a AttributeType) Decode(data []byte, unmarshaller runtime.Unmarshaler) (interface{}, error) {
	var value bool
	attr := &Attribute{}

	err := unmarshaller.Unmarshal(data, attr)
	if err == nil {
		switch attr.Mode {
		case "":
		case NoneMode:
		case HashMode:
		case ShortHashMode:
		case MappingMode:
		default:
			return nil, errors.ErrInvalid("mode", attr.Mode)
		}
		return attr, nil
	}

	err = unmarshaller.Unmarshal(data, &value)
	if err == nil {
		if value {
			attr.Mode = ShortHashMode
		} else {
			attr.Mode = NoneMode
		}
		attr.PrefixMappings = map[string]string{}
		return attr, nil
	}

	return value, err
}

////////////////////////////////////////////////////////////////////////////////

const (
	NoneMode      = "none"
	HashMode      = "hash"
	ShortHashMode = "shortHash"
	MappingMode   = "mapping"
)

type Attribute struct {
	Mode           string            `json:"mode"`
	Always         bool              `json:"always,omitempty"`
	Prefix         *string           `json:"prefix,omitempty"`
	PrefixMappings map[string]string `json:"prefixMappings,omitempty"`
}

func (a *Attribute) Map(name string) string {
	if len(a.PrefixMappings) == 0 {
		a.PrefixMappings = map[string]string{}
	}
	switch a.Mode {
	case "", NoneMode:
		return name
	case HashMode, ShortHashMode:
		return a.Hash(name, a.Mode == ShortHashMode)
	case MappingMode:
		return a.MapPrefix(name)
	}
	return name
}

func (a *Attribute) MapPrefix(name string) string {
	keys := utils.StringMapKeys(a.PrefixMappings)
	for i := range keys {
		k := keys[len(keys)-i-1]
		if strings.HasPrefix(name, k+grammar.RepositorySeparator) {
			name = a.PrefixMappings[k] + name[len(k):]
			break
		}
	}
	return name
}

func (a *Attribute) Hash(name string, short bool) string {
	if idx := strings.LastIndex(name, grammar.RepositorySeparator); idx > 0 {
		prefix := name[:idx]
		sum := sha256.Sum256([]byte(prefix))
		n := hex.EncodeToString(sum[:])
		if short {
			n = n[:8]
		}
		n += grammar.RepositorySeparator + name[idx+1:]
		if a.Always || len(n) < len(name) {
			name = n
		}
	}
	return name
}

func (a *Attribute) Copy() *Attribute {
	n := *a
	n.PrefixMappings = maps.Clone(n.PrefixMappings)
	return &n
}

////////////////////////////////////////////////////////////////////////////////

func Get(ctx datacontext.Context) *Attribute {
	a := ctx.GetAttributes().GetAttribute(ATTR_KEY)
	if a == nil {
		return &Attribute{Mode: NoneMode}
	}
	if b, ok := a.(bool); ok {
		if b {
			return &Attribute{Mode: ShortHashMode}
		} else {
			return &Attribute{Mode: NoneMode}
		}
	}
	return a.(*Attribute).Copy()
}

func Set(ctx datacontext.Context, a *Attribute) error {
	return ctx.GetAttributes().SetAttribute(ATTR_KEY, a)
}
