// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// ByteMapToRawMessageMap converts a map of bytes to a map of json.RawMessages
func ByteMapToRawMessageMap(m map[string][]byte) (map[string]json.RawMessage, error) {
	n := make(map[string]json.RawMessage, len(m))
	for key, val := range m {
		jsonVal, err := yaml.ToJSON(val)
		if err != nil {
			return nil, err
		}
		n[key] = json.RawMessage(jsonVal)
	}
	return n, nil
}

// StringMapToRawMessageMap converts a map of strings to a map of json.RawMessages
func StringMapToRawMessageMap(m map[string]string) (map[string]json.RawMessage, error) {
	n := make(map[string]json.RawMessage, len(m))
	for key, val := range m {
		jsonVal, err := yaml.ToJSON([]byte(val))
		if err != nil {
			return nil, err
		}
		n[key] = json.RawMessage(jsonVal)
	}
	return n, nil
}

// ResolveSecretReference is an auxiliary function that fetches the content of a secret as specified by the given SecretReference
// The first returned value is the complete secret content, the second one the specified key (if set), the third one is the generation of the secret
func ResolveSecretReference(ctx context.Context, kubeClient client.Client, secretRef *lsv1alpha1.SecretReference) (map[string][]byte, []byte, int64, error) {
	secret := &corev1.Secret{}
	if err := kubeClient.Get(ctx, secretRef.NamespacedName(), secret); err != nil {
		return nil, nil, 0, err
	}
	completeData := secret.Data
	var (
		data   []byte
		ok     bool
		rawMap map[string]json.RawMessage
		err    error
	)
	if len(secretRef.Key) != 0 {
		data, ok = secret.Data[secretRef.Key]
		if !ok {
			return nil, nil, 0, fmt.Errorf("key %s in secret %s does not exist", secretRef.Key, secretRef.NamespacedName().String())
		}
	} else {
		// use the whole secret as map
		rawMap, err = ByteMapToRawMessageMap(secret.Data)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("unable to convert secret data to raw message map: %w", err)
		}
		data, err = json.Marshal(rawMap)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("unable to marshal secret data as map: %w", err)
		}
	}

	return completeData, data, secret.Generation, nil
}

// ResolveConfigMapReference is an auxiliary function that fetches the content of a configmap as specified by the given ConfigMapReference
// The first returned value is the complete configmap content, the second one the specified key (if set), the third one is the generation of the configmap
func ResolveConfigMapReference(ctx context.Context, kubeClient client.Client, configMapRef *lsv1alpha1.ConfigMapReference) (map[string][]byte, []byte, int64, error) {
	cm := &corev1.ConfigMap{}
	if err := kubeClient.Get(ctx, configMapRef.NamespacedName(), cm); err != nil {
		return nil, nil, 0, err
	}
	completeData := cm.BinaryData
	if completeData == nil {
		completeData = map[string][]byte{}
	}
	for k, v := range cm.Data {
		// kubernetes verifies that this doesn't cause any collisions
		completeData[k] = []byte(v)
	}
	var (
		data   []byte
		sdata  string
		rawMap map[string]json.RawMessage
		err    error
	)
	keyFound := len(configMapRef.Key) == 0
	if cm.Data != nil {
		if len(configMapRef.Key) != 0 {
			sdata, keyFound = cm.Data[configMapRef.Key]
			data = []byte(sdata)
		} else {
			// use whole configmap as json object
			rawMap, err := StringMapToRawMessageMap(cm.Data)
			if err != nil {
				return nil, nil, 0, fmt.Errorf("unable to convert configmap data to raw message map: %w", err)
			}
			data, err = json.Marshal(rawMap)
			if err != nil {
				return nil, nil, 0, fmt.Errorf("unable to marshal configmap data as map: %w", err)
			}
		}
	}
	if cm.BinaryData != nil {
		if len(configMapRef.Key) != 0 {
			data, keyFound = cm.BinaryData[configMapRef.Key]
		} else {
			// use whole configmap as json object
			rawMap, err = ByteMapToRawMessageMap(cm.BinaryData)
			if err != nil {
				return nil, nil, 0, fmt.Errorf("unable to convert configmap data to raw message map: %w", err)
			}
			data, err = json.Marshal(rawMap)
			if err != nil {
				return nil, nil, 0, fmt.Errorf("unable to marshal configmap data as map: %w", err)
			}
		}
	}
	if !keyFound {
		return nil, nil, 0, fmt.Errorf("key '%s' in configmap '%s' does not exist", configMapRef.Key, configMapRef.NamespacedName().String())
	}

	return completeData, data, cm.Generation, nil
}
