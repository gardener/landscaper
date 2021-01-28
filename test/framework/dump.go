// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package framework

import (
	"context"
	"fmt"
	"io"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
)

// Dumper is a struct to dump logs and useful information about known object for a state
type Dumper struct {
	kubeClient client.Client
	namespaces sets.String
	writer     io.Writer
}

// NewDumper create a new dumper
func NewDumper(writer io.Writer, kubeClient client.Client, namespaces ...string) *Dumper {
	return &Dumper{
		writer:     writer,
		kubeClient: kubeClient,
		namespaces: sets.NewString(namespaces...),
	}
}

// AddNamespaces adds additional namespaces that should be dumped.
func (d *Dumper) AddNamespaces(namespaces ...string) {
	d.namespaces.Insert(namespaces...)
}

// ClearNamespaces removes all current namespaces
func (d *Dumper) ClearNamespaces() {
	d.namespaces = sets.NewString()
}

// Dump searches for known objects in the given namespaces and dumps useful information about their state.
// Currently information about the main landscaper resources in dumped:
// - Installations
// - DeployItems
// todo: add additional resources
func (d *Dumper) Dump(ctx context.Context) error {
	for ns := range d.namespaces {
		// check if namespace exists
		if err := d.kubeClient.Get(ctx, kutil.ObjectKey(ns, ""), &corev1.Namespace{}); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return err
		}
		if err := d.DumpInstallationsInNamespace(ctx, ns); err != nil {
			return err
		}
		if err := d.DumpDeployItemsInNamespace(ctx, ns); err != nil {
			return err
		}
	}
	return nil
}

func (d *Dumper) DumpInstallationsInNamespace(ctx context.Context, namespace string) error {
	installationList := &lsv1alpha1.InstallationList{}
	if err := d.kubeClient.List(ctx, installationList); err != nil {
		return fmt.Errorf("unable to list installations for namespace %q: %w", namespace, err)
	}
	for _, inst := range installationList.Items {
		if err := DumpInstallation(d.writer, &inst); err != nil {
			return err
		}
	}
	return nil
}

// DumpInstallation dumps information about the installation
func DumpInstallation(writer io.Writer, inst *lsv1alpha1.Installation) error {
	if _, err := fmt.Fprintf(writer, "--- Installation %s\n", inst.Name); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "%s\n", FormatAsYAML(inst.Status, "")); err != nil {
		return err
	}
	return nil
}

// DumpDeployItemsInNamespace dumps information about all deploy items int he given namespace
func (d *Dumper) DumpDeployItemsInNamespace(ctx context.Context, namespace string) error {
	list := &lsv1alpha1.DeployItemList{}
	if err := d.kubeClient.List(ctx, list); err != nil {
		return fmt.Errorf("unable to list deplo items for namespace %q: %w", namespace, err)
	}
	for _, item := range list.Items {
		if err := DumpDeployItems(d.writer, &item); err != nil {
			return err
		}
	}
	return nil
}

// DumpDeployItems dumps information about the deploy items
func DumpDeployItems(writer io.Writer, deployItem *lsv1alpha1.DeployItem) error {
	fmtMsg := `
--- DeployItem %s
Type: %s
Config: %s
`
	if _, err := fmt.Fprintf(writer, fmtMsg,
		deployItem.Name,
		deployItem.Spec.Type,
		FormatAsYAML(deployItem.Spec.Configuration, "  ")); err != nil {
		return err
	}
	fmtMsg = `
Status:
  Phase: %s
  Error: %s
  ProviderConfig: %s
`

	if _, err := fmt.Fprintf(writer, fmtMsg,
		deployItem.Status.Phase,
		FormatLastError(deployItem.Status.LastError, "    "),
		FormatAsYAML(deployItem.Status.ProviderStatus, "    ")); err != nil {
		return err
	}
	return nil
}

// FormatAsYAML formats a object as yaml
func FormatAsYAML(obj interface{}, indent string) string {
	if obj == nil {
		return "none"
	}
	data, err := yaml.Marshal(obj)
	if err != nil {
		return fmt.Sprintf("Error during yaml serialization: %s", err.Error())
	}
	// add indentation
	out := strings.ReplaceAll(string(data), "\n", "\n"+indent)
	// add an additional newline to properly inline
	out = "\n" + indent + out
	return out
}

// FormatLastError formats a error in a human readable format.
func FormatLastError(err *lsv1alpha1.Error, indent string) string {
	if err == nil {
		return "none"
	}
	format := `

Operation: %s
Reason: %s
Message: %s
LastUpdated: %s
`
	format = strings.ReplaceAll(format, "\n", "\n"+indent)
	return fmt.Sprintf(format, err.Operation, err.Reason, err.Message, err.LastUpdateTime.String())
}
