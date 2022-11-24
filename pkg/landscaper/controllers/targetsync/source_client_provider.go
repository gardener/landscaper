// SPDX-FileCopyrightText: 2022 "SAP SE or an SAP affiliate company and Gardener contributors"
//
// SPDX-License-Identifier: Apache-2.0

package targetsync

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

type SourceClientProvider interface {
	GetSourceClient(
		ctx context.Context,
		targetSync *lsv1alpha1.TargetSync,
		targetClient client.Client,
		schema *runtime.Scheme) (client.Client, *rest.Config, error)

	GetUnstructuredSourceClient(
		ctx context.Context,
		targetSync *lsv1alpha1.TargetSync,
		targetClient client.Client,
		groupVersionResource schema.GroupVersionResource) (dynamic.ResourceInterface, error)
}

type DefaultSourceClientProvider struct{}

var _ SourceClientProvider = &DefaultSourceClientProvider{}

func NewDefaultSourceClientProvider() SourceClientProvider {
	return &DefaultSourceClientProvider{}
}

func (p *DefaultSourceClientProvider) GetSourceClient(
	ctx context.Context,
	targetSync *lsv1alpha1.TargetSync,
	targetClient client.Client,
	schema *runtime.Scheme) (client.Client, *rest.Config, error) {

	restConfig, err := p.getSourceRestConfig(ctx, targetSync, targetClient)
	if err != nil {
		return nil, nil, err
	}

	cl, err := client.New(restConfig, client.Options{
		Scheme: schema,
	})
	if err != nil {
		return nil, nil, err
	}

	return cl, restConfig, nil
}

func (p *DefaultSourceClientProvider) GetUnstructuredSourceClient(
	ctx context.Context,
	targetSync *lsv1alpha1.TargetSync,
	targetClient client.Client,
	groupVersionResource schema.GroupVersionResource) (dynamic.ResourceInterface, error) {

	restConfig, err := p.getSourceRestConfig(ctx, targetSync, targetClient)
	if err != nil {
		return nil, err
	}

	cl, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create unstructured client for target sync: %w", err)
	}

	typedClient := cl.Resource(groupVersionResource).Namespace(targetSync.Spec.SourceNamespace)

	return typedClient, nil
}

func (p *DefaultSourceClientProvider) getSourceRestConfig(ctx context.Context, targetSync *lsv1alpha1.TargetSync, targetClient client.Client) (*rest.Config, error) {
	kubeconfigBytes, err := p.resolveSecretRef(ctx, targetClient, targetSync.Spec.SecretRef, targetSync.Namespace)
	if err != nil {
		return nil, err
	}

	clientConfig, err := clientcmd.NewClientConfigFromBytes(kubeconfigBytes)
	if err != nil {
		return nil, err
	}

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	return restConfig, nil
}

func (p *DefaultSourceClientProvider) resolveSecretRef(ctx context.Context, targetClient client.Client,
	secretRef lsv1alpha1.LocalSecretReference, namespace string) ([]byte, error) {

	secret := &corev1.Secret{}
	secretKey := client.ObjectKey{
		Namespace: namespace,
		Name:      secretRef.Name,
	}

	if err := targetClient.Get(ctx, secretKey, secret); err != nil {
		return nil, err
	}

	kubeconfigBytes := secret.Data[secretRef.Key]
	if len(kubeconfigBytes) == 0 {
		return nil, fmt.Errorf("no kubeconfig in secret")
	}

	return kubeconfigBytes, nil
}

type TrivialSourceClientProvider struct {
	sourceClient             client.Client
	unstructuredSourceClient dynamic.ResourceInterface
}

var _ SourceClientProvider = &TrivialSourceClientProvider{}

func NewTrivialSourceClientProvider(
	sourceClient client.Client,
	unstructuredSourceClient dynamic.ResourceInterface) SourceClientProvider {

	return &TrivialSourceClientProvider{
		sourceClient:             sourceClient,
		unstructuredSourceClient: unstructuredSourceClient,
	}
}

func (p *TrivialSourceClientProvider) GetSourceClient(
	_ context.Context,
	_ *lsv1alpha1.TargetSync,
	_ client.Client,
	_ *runtime.Scheme) (client.Client, *rest.Config, error) {

	return p.sourceClient, nil, nil
}

func (p *TrivialSourceClientProvider) GetUnstructuredSourceClient(
	ctx context.Context,
	targetSync *lsv1alpha1.TargetSync,
	targetClient client.Client,
	groupVersionResource schema.GroupVersionResource) (dynamic.ResourceInterface, error) {

	return p.unstructuredSourceClient, nil
}
