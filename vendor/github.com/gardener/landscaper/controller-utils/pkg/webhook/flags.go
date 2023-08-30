// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"

	flag "github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/sets"

	utilerrors "github.com/gardener/landscaper/controller-utils/pkg/errors"
)

var kcidSync = &sync.Mutex{}
var knownClusterIDs = map[string]*ClusterID{}

// ClusterID is a helper struct to identify a cluster.
// It should always be created by using the ID function!
// Note that ID and name must both be unique across all ClusterID objects. There must not be two ClusterID resources with the same ID, nor two ClusterID resources with the same name!
type ClusterID struct {
	ID   string
	Name string
}

// ID returns a cluster ID.
// It ensures uniqueness of cluster IDs - basically, calling this function multiple times with the same id will always return a pointer to the same object.
// The given name is only relevant if the function is called for the first time with the given ID.
// This means that the name in the returned object can actually differ from the one given, but only if cluster IDs are used in the wrong way.
func ID(id, name string) *ClusterID {
	kcidSync.Lock()
	defer kcidSync.Unlock()
	c, ok := knownClusterIDs[id]
	if ok {
		return c
	}
	c = &ClusterID{
		ID:   id,
		Name: name,
	}
	knownClusterIDs[id] = c
	return c
}

// WebhookFlags is a helper struct for making the webhook server configuration configurable via command-line flags.
// It should be instantiated using the NewWebhookFlags function.
//
// Accessing cluster-specific configuration:
// Single-cluster example (getting the certificate namespace from a WebhookFlags object named 'wf'):
//
//	wf.CertNamespace (or: wf.MultiWebhookFlags.CertNamespace)
//
// Multi-cluster example (getting the certificate namespace for a cluster with ClusterID 'cid' from a WebhookFlags object named 'wf'):
//
//	wf.MultiCluster[cid].CertNamespace
type WebhookFlags struct {
	*MultiWebhookFlags // contains the configuration if isMulti is false

	Port                int                               // port where the webhook server is running
	DisabledWebhooksRaw string                            // lists disabled webhooks as a comma-separated string
	DisabledWebhooks    sets.Set[string]                  // transformed version of DisabledWebhooksRaw
	MultiCluster        map[*ClusterID]*MultiWebhookFlags // contains the configuration if isMulti is true
}

// MultiWebhookFlags contain flags which need to be added multiple times if multiple clusters need to reach the webhook server.
// It is automatically instantiated with the NewWebhookFlags function as part of the returned WebhookFlags object.
type MultiWebhookFlags struct {
	WebhookServiceNamespaceName string                 // webhook service namespace and name in the format <namespace>/<name>
	WebhookServicePort          int32                  // port of the webhook service
	WebhookService              *WebhookServiceOptions // transformed version of the webhook service flags
	CertNamespace               string                 // the namespace in which the webhook credentials are being created/updated
	WebhookURL                  string                 // URL of the webhook server if running outside of cluster
}

func (wf *WebhookFlags) AddFlags(fs *flag.FlagSet) {
	fs.IntVar(&wf.Port, "port", 9443, "Specify the port of the webhook server")
	fs.StringVar(&wf.DisabledWebhooksRaw, "disable-webhooks", "", "Specify validation webhooks that should be disabled ('all' to disable validation completely)")

	if wf.IsMultiCluster() {
		for cluster, mwf := range wf.MultiCluster {
			mwf.addMultiFlags(fs, cluster)
		}
	} else {
		wf.addMultiFlags(fs, nil)
	}
}

func (mwf *MultiWebhookFlags) addMultiFlags(fs *flag.FlagSet, cluster *ClusterID) {
	prefix := ""
	cphrase := ""
	if cluster != nil {
		prefix = fmt.Sprintf("%s-", cluster.ID)
		cphrase = fmt.Sprintf("%s cluster ", cluster.Name)
	}

	fs.StringVar(&mwf.WebhookServiceNamespaceName, fmt.Sprintf("%swebhook-service", prefix), "", fmt.Sprintf("Specify namespace and name of the %swebhook service (format: <namespace>/<name>)", cphrase))
	fs.Int32Var(&mwf.WebhookServicePort, fmt.Sprintf("%swebhook-service-port", prefix), 9443, fmt.Sprintf("Specify the port of the %swebhook service", cphrase))
	fs.StringVar(&mwf.WebhookURL, fmt.Sprintf("%swebhook-url", prefix), "", fmt.Sprintf("Specify the URL of the external %swebhook service (scheme://host:port)", cphrase))
	fs.StringVar(&mwf.CertNamespace, fmt.Sprintf("%scert-ns", prefix), "", fmt.Sprintf("Specify the namespace in which the %scertificates are being stored", cphrase))
}

// NewWebhookFlags returns a new WebhookFlags object.
//
// How this is supposed to work:
//
// If you have only one cluster, from which the webhook server must be reachable, call this function without any arguments.
// In the returned object, the cluster-specific configuration is contained in the embedded MultiWebhookFlags struct.
// Example (getting the certificate namespace from a WebhookFlags object named 'wf'):
//
//	wf.CertNamespace (or: wf.MultiWebhookFlags.CertNamespace)
//
// If the webhook server must be reachable from multiple clusters, create a cluster ID for each cluster via the ID(id, name) function and pass all of them as arguments.
// In the returned object, the cluster-specific configuration is contained in the MultiCluster map, with the previously created ClusterID object serving as key.
// Example (getting the certificate namespace for a cluster with ClusterID 'cid' from a WebhookFlags object named 'wf'):
//
//	wf.MultiCluster[cid].CertNamespace
func NewWebhookFlags(clusters ...*ClusterID) *WebhookFlags {
	res := &WebhookFlags{}

	if len(clusters) > 0 {
		res.MultiCluster = map[*ClusterID]*MultiWebhookFlags{}

		for _, c := range clusters {
			res.MultiCluster[c] = &MultiWebhookFlags{}
		}
	} else {
		res.MultiWebhookFlags = &MultiWebhookFlags{}
	}

	return res
}

// Complete transforms and validates the arguments given via the CLI.
// The registry argument may be nil (or empty), then the values for the disabled webhooks flag cannot be validated.
func (wf *WebhookFlags) Complete(wr WebhookRegistry) error {
	errs := utilerrors.NewErrorList()

	// transform disabled webhooks string into slice
	wf.DisabledWebhooks = sets.New[string]()
	if wf.DisabledWebhooksRaw != "" {
		dws := strings.Split(wf.DisabledWebhooksRaw, ",")
		for _, dw := range dws {
			wf.DisabledWebhooks.Insert(strings.TrimSpace(dw))
		}
	}

	if wf.Port < 0 {
		errs.Append(fmt.Errorf("port must not be negative"))
	}
	if len(wr) != 0 && len(wf.DisabledWebhooks) != 0 {
		allowedDisablesSet := sets.KeySet[string, *Webhook](wr)
		allowedDisablesSet.Insert("all")
		allowedDisablesList := allowedDisablesSet.UnsortedList()
		sort.Strings(allowedDisablesList) // doesn't necessarily need to be sorted, but order should be stable so that the tooltip doesn't change every time
		allowedDisablesString := strings.Join(allowedDisablesList, ", ")
		for dw := range wf.DisabledWebhooks {
			if !allowedDisablesSet.Has(dw) {
				errs.Append(fmt.Errorf("invalid disabled webhook '%s', allowed values are [%s]", dw, allowedDisablesString))
			}
		}
	}

	var cf map[*ClusterID]*MultiWebhookFlags
	if wf.IsMultiCluster() {
		cf = wf.MultiCluster
	} else {
		cf = map[*ClusterID]*MultiWebhookFlags{
			nil: wf.MultiWebhookFlags,
		}
	}
	for cid, fl := range cf {
		idString := ""
		if cid != nil {
			idString = fmt.Sprintf(" for %s cluster", cid.Name)
		}
		if (fl.WebhookServiceNamespaceName != "") == (fl.WebhookURL != "") {
			// either both or none are set
			errs.Append(fmt.Errorf("invalid flags%s: exactly one of url and service must be specified", idString))
		}
		if fl.WebhookServiceNamespaceName != "" {
			svc := strings.Split(fl.WebhookServiceNamespaceName, "/")
			if len(svc) != 2 {
				errs.Append(fmt.Errorf("invalid format of webhook service namespace and name%s: expected format is '<namespace>/<name>', but got '%s'", idString, fl.WebhookServiceNamespaceName))
			} else {
				fl.WebhookService = &WebhookServiceOptions{
					Name:      svc[1],
					Namespace: svc[0],
					Port:      fl.WebhookServicePort,
				}
				if fl.WebhookService.Name == "" {
					errs.Append(fmt.Errorf("webhook service name%s must not be empty", idString))
				}
				if fl.WebhookService.Namespace == "" {
					errs.Append(fmt.Errorf("webhook service namespace%s must not be empty", idString))
				}
			}
		}
		if fl.WebhookServicePort < 0 {
			errs.Append(fmt.Errorf("webhook service port%s '%d' is invalid: port must not be negative", idString, fl.WebhookServicePort))
		}
		if fl.WebhookURL != "" {
			parsedUrl, err := url.Parse(fl.WebhookURL)
			if err != nil {
				errs.Append(fmt.Errorf("invalid webhook url '%s'%s: %w", fl.WebhookURL, idString, err))
			} else if parsedUrl.Host == "" {
				errs.Append(fmt.Errorf("invalid webhook url%s: expected format is '<scheme>://<host>:<port>', but got %s", idString, fl.WebhookURL))
			}
		}
		if fl.CertNamespace == "" {
			if fl.WebhookService != nil {
				fl.CertNamespace = fl.WebhookService.Namespace
			} else {
				errs.Append(fmt.Errorf("certificate namespace%s is empty and cannot be derived from service namespace", idString))
			}
		}
	}

	return errs.Aggregate()
}

// ForSingleCluster returns the webhook flags for a single cluster.
// Cluster id may be nil if and only if wf.IsMultiCluster() == false.
// If the given cluster id is nil, the receiver object is returned as is.
// Otherwise, a new WebhookFlags object is created with IsMultiCluster set to 'false' and the embedded MultiWebhookFlags object's fields set to the values from MultiCluster[cid].
// Returns an error if cid and wf.IsMultiCluster() don't fit together, or when no matching configuration for the given cluster id is found.
func (wf *WebhookFlags) ForSingleCluster(cid *ClusterID) (*WebhookFlags, error) {
	if cid == nil {
		if wf.IsMultiCluster() {
			return nil, fmt.Errorf("WebhookFlags object contains multi-cluster configuration, calling ForSingleCluster with nil argument is not allowed")
		}
		return wf, nil
	}
	if !wf.IsMultiCluster() {
		return nil, fmt.Errorf("WebhookFlags object contains single-cluster configuration, calling ForSingleCluster with non-nil argument is not allowed")
	}
	mc, ok := wf.MultiCluster[cid]
	if !ok {
		return nil, fmt.Errorf("no matching multi-cluster configuration found for cluster id '%s'", cid.ID)
	}
	res := &WebhookFlags{
		Port:                wf.Port,
		DisabledWebhooksRaw: wf.DisabledWebhooksRaw,
		MultiWebhookFlags: &MultiWebhookFlags{
			WebhookServiceNamespaceName: mc.WebhookServiceNamespaceName,
			WebhookServicePort:          mc.WebhookServicePort,
			CertNamespace:               mc.CertNamespace,
			WebhookURL:                  mc.WebhookURL,
		},
	}
	if wf.DisabledWebhooks != nil {
		res.DisabledWebhooks = sets.New[string](wf.DisabledWebhooks.UnsortedList()...)
	}
	if mc.WebhookService != nil {
		res.WebhookService = &WebhookServiceOptions{
			Name:      mc.WebhookService.Name,
			Namespace: mc.WebhookService.Namespace,
			Port:      mc.WebhookService.Port,
		}
	}
	return res, nil
}

// IsMultiCluster returns true if the webhook flags object contains configuration for multiple clusters (or modified the object accordingly afterwards).
func (wf *WebhookFlags) IsMultiCluster() bool {
	return wf.MultiCluster != nil
}
