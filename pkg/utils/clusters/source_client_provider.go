// SPDX-FileCopyrightText: 2023 "SAP SE or an SAP affiliate company and Gardener contributors"
//
// SPDX-License-Identifier: Apache-2.0

package clusters

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/pkg/utils/read_write_layer"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

type SourceClientProvider interface {
	GetSourceClient(
		ctx context.Context,
		targetSync *lsv1alpha1.TargetSync,
		targetClient client.Client,
		schema *runtime.Scheme) (client.Client, *rest.Config, error)

	GetSourceShootClient(
		ctx context.Context,
		targetSync *lsv1alpha1.TargetSync,
		targetClient client.Client) (*ShootClient, error)
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

func (p *DefaultSourceClientProvider) GetSourceShootClient(
	ctx context.Context,
	targetSync *lsv1alpha1.TargetSync,
	targetClient client.Client) (*ShootClient, error) {

	gardenKubeconfigBytes, err := p.resolveSecretRef(ctx, targetClient, targetSync.Spec.SecretRef, targetSync.Namespace)
	if err != nil {
		return nil, err
	}

	return NewShootClient(gardenKubeconfigBytes)
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

	if err := read_write_layer.GetSecret(ctx, targetClient, secretKey, secret, read_write_layer.R000055); err != nil {
		return nil, err
	}

	kubeconfigBytes := secret.Data[secretRef.Key]
	if len(kubeconfigBytes) == 0 {
		return nil, fmt.Errorf("no kubeconfig in secret")
	}

	return kubeconfigBytes, nil
}

type TrivialSourceClientProvider struct {
	sourceClient client.Client
	shootClient  *ShootClient
}

var _ SourceClientProvider = &TrivialSourceClientProvider{}

func NewTrivialSourceClientProvider(sourceClient client.Client, shootClient *ShootClient) SourceClientProvider {
	return &TrivialSourceClientProvider{
		sourceClient: sourceClient,
		shootClient:  shootClient,
	}
}

func (p *TrivialSourceClientProvider) GetSourceClient(
	_ context.Context,
	_ *lsv1alpha1.TargetSync,
	_ client.Client,
	_ *runtime.Scheme) (client.Client, *rest.Config, error) {

	return p.sourceClient, nil, nil
}

func (p *TrivialSourceClientProvider) GetSourceShootClient(
	_ context.Context,
	_ *lsv1alpha1.TargetSync,
	_ client.Client) (*ShootClient, error) {

	return p.shootClient, nil
}
