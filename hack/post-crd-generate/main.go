// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

// Contains a small helper program to post manipulate the generated crds.
// controller-gen only sets type Any for unknown types see https://github.com/kubernetes-sigs/controller-tools/pull/427
// As Any is an unsupported value the type has to be changed from `type: Any` to `x-kubernetes-preserve-unknown-fields`
//
// As Kubernetes does also not support usage of `{}` we have to workaround with x-kubernetes-preserve-unknown-fields
// Therefore this script replaces all occurrences for "items" and "additionalProperties"
//
// It also removes the status field as there are currently issues with the generation see https://github.com/kubernetes-sigs/controller-tools/issues/456

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"
	"sigs.k8s.io/yaml"
)

func main() {
	pflag.Parse()

	if len(pflag.Args()) != 1 {
		printHelp()
		fmt.Println("Path to the crds is missing")
		os.Exit(1)
	}

	err := filepath.Walk(pflag.Arg(0), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		data, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		var crd map[string]interface{}
		if err := yaml.Unmarshal(data, &crd); err != nil {
			return err
		}

		crd = replaceTypesInStruct(crd)

		// remove status field
		delete(crd, "status")

		data, err = yaml.Marshal(crd)
		if err != nil {
			return err
		}

		fmt.Printf("Generated %s\n", path)

		if debug || dryRun {
			fmt.Println(string(data))
		}

		if dryRun {
			return nil
		}

		return ioutil.WriteFile(path, data, info.Mode())
	})
	if err != nil {
		printError(err)
	}

}

func printError(err error) {
	fmt.Println(err.Error())
	os.Exit(1)
}

func printHelp() {
	help := `
go run main.go path/to/crds
`
	fmt.Print(help)
}
