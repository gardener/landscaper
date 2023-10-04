// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

//go:generate go-bindata -nometadata -pkg jsonscheme ../../../../../../../../resources/component-descriptor-ocm-v3-schema.yaml
//go:generate gofmt -s -w bindata.go

package jsonscheme

import (
	"errors"
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/xeipuuv/gojsonschema"
)

var Schema *gojsonschema.Schema

func init() {
	dataBytes, err := ResourcesComponentDescriptorOcmV3SchemaYamlBytes()
	if err != nil {
		panic(err)
	}

	data, err := yaml.YAMLToJSON(dataBytes)
	if err != nil {
		panic(err)
	}

	Schema, err = gojsonschema.NewSchema(gojsonschema.NewBytesLoader(data))
	if err != nil {
		panic(err)
	}
}

// Validate validates the given data against the component descriptor v2 jsonscheme.
func Validate(src []byte) error {
	data, err := yaml.YAMLToJSON(src)
	if err != nil {
		return err
	}
	documentLoader := gojsonschema.NewBytesLoader(data)
	res, err := Schema.Validate(documentLoader)
	if err != nil {
		return err
	}

	if !res.Valid() {
		errs := res.Errors()
		errMsg := errs[0].String()
		for i := 1; i < len(errs); i++ {
			errMsg = fmt.Sprintf("%s;%s", errMsg, errs[i].String())
		}
		return errors.New(errMsg)
	}

	return nil
}
