// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"context"
	"crypto/sha1"
	"encoding/base32"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/api"
)

var StateNotFoundErr = errors.New("state not found")

type GenericStateHandler interface {
	Store(context.Context, string, []byte) error
	Get(context.Context, string) ([]byte, error)
}

// KubernetesStateHandler implements the GenericStateHandler interface
// that stores the stateHdl in a kubernetes cluster.
type KubernetesStateHandler struct {
	KubeClient client.Client
	Inst       *lsv1alpha1.Installation
}

var _ GenericStateHandler = &KubernetesStateHandler{}

func (s KubernetesStateHandler) Store(ctx context.Context, name string, data []byte) error {
	name = s.secretName(name)
	secret, err := s.get(ctx, name)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}

		secret = &corev1.Secret{}
		secret.Name = name
		secret.Namespace = s.Inst.Namespace
		secret.Data = map[string][]byte{
			lsv1alpha1.DataObjectSecretDataKey: data,
		}
		if err := controllerutil.SetControllerReference(s.Inst, secret, api.LandscaperScheme); err != nil {
			return fmt.Errorf("unable to set controller reference: %w", err)
		}
		return s.KubeClient.Create(ctx, secret)
	}

	secret.Data = map[string][]byte{
		lsv1alpha1.DataObjectSecretDataKey: data,
	}
	return s.KubeClient.Update(ctx, secret)
}

func (s KubernetesStateHandler) Get(ctx context.Context, name string) ([]byte, error) {
	name = s.secretName(name)
	secret, err := s.get(ctx, name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, StateNotFoundErr
		}
		return nil, err
	}
	state, ok := secret.Data[lsv1alpha1.DataObjectSecretDataKey]
	if !ok {
		if !apierrors.IsNotFound(err) {
			return nil, StateNotFoundErr
		}
	}
	return state, nil
}

func (s KubernetesStateHandler) get(ctx context.Context, secretName string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	if err := s.KubeClient.Get(ctx, kutil.ObjectKey(secretName, s.Inst.Namespace), secret); err != nil {
		return nil, err
	}
	return secret, nil
}

func (s KubernetesStateHandler) secretName(name string) string {
	doName := fmt.Sprintf("%s/%s", s.Inst.Name, name)
	h := sha1.New()
	_, _ = h.Write([]byte(doName))
	// we need base32 encoding as some base64 (even url safe base64) characters are not supported by k8s
	// see https://kubernetes.io/docs/concepts/overview/working-with-objects/names/
	return base32.NewEncoding(lsv1alpha1helper.Base32EncodeStdLowerCase).WithPadding(base32.NoPadding).EncodeToString(h.Sum(nil))
}

type MemoryStateHandler map[string][]byte

var _ GenericStateHandler = MemoryStateHandler{}

func NewMemoryStateHandler() MemoryStateHandler {
	return MemoryStateHandler{}
}

func (m MemoryStateHandler) Store(_ context.Context, name string, data []byte) error {
	m[name] = data
	return nil
}

func (m MemoryStateHandler) Get(_ context.Context, name string) ([]byte, error) {
	data, ok := m[name]
	if !ok {
		return nil, StateNotFoundErr
	}
	return data, nil
}
