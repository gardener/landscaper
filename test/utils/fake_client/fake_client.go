// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fake_client

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	mock_client "github.com/gardener/landscaper/pkg/utils/mocks/client"
)

// State contains the state of initialized fake client
type State struct {
	DataTypes     map[string]*lsv1alpha1.DataType
	Installations map[string]*lsv1alpha1.ComponentInstallation
	Secrets       map[string]*corev1.Secret
}

// NewFakeClientFromPath reads all landscaper related files from the given path adds them to the controller runtime's fake client.
func NewFakeClientFromPath(path string) (client.Client, *State, error) {
	objects := []runtime.Object{}
	state := &State{
		DataTypes:     make(map[string]*lsv1alpha1.DataType),
		Installations: make(map[string]*lsv1alpha1.ComponentInstallation),
		Secrets:       make(map[string]*corev1.Secret),
	}
	err := filepath.Walk("./testdata/state", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		data, err := ioutil.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "unable to read file %s", path)
		}

		objects, err = decodeAndAppendLSObject(data, objects, state)
		if err != nil {
			return errors.Wrapf(err, "unable to decode file %s", path)
		}

		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return fake.NewFakeClientWithScheme(kubernetes.LandscaperScheme, objects...), state, nil
}

func decodeAndAppendLSObject(data []byte, objects []runtime.Object, state *State) ([]runtime.Object, error) {
	var allErrors *multierror.Error
	decoder := serializer.NewCodecFactory(kubernetes.LandscaperScheme).UniversalDecoder()

	inst := &lsv1alpha1.ComponentInstallation{}
	_, _, err := decoder.Decode(data, nil, inst)
	if err == nil {
		state.Installations[types.NamespacedName{Name: inst.Name, Namespace: inst.Namespace}.String()] = inst
		return append(objects, inst), nil
	}
	allErrors = multierror.Append(allErrors, errors.Wrap(err, "unable to decode file"))

	dt := &lsv1alpha1.DataType{}
	_, _, err = decoder.Decode(data, nil, dt)
	if err == nil {
		state.DataTypes[types.NamespacedName{Name: dt.Name, Namespace: dt.Namespace}.String()] = dt
		return append(objects, dt), nil
	}
	allErrors = multierror.Append(allErrors, errors.Wrap(err, "unable to decode file"))

	secret := &corev1.Secret{}
	_, _, err = decoder.Decode(data, nil, secret)
	if err == nil {

		// add stringdata and data
		if secret.Data == nil {
			secret.Data = make(map[string][]byte)
		}
		for key, data := range secret.StringData {
			secret.Data[key] = []byte(data)
		}

		state.Secrets[types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}.String()] = secret
		return append(objects, secret), nil
	}
	allErrors = multierror.Append(allErrors, errors.Wrap(err, "unable to decode file"))

	return nil, allErrors
}

// RegisterFakeClientToMock adds fake client calls to a mockclient
func RegisterFakeClientToMock(mockClient *mock_client.MockClient, fakeClient client.Client) error {
	mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(fakeClient.Get)
	mockClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(fakeClient.Create)
	mockClient.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(fakeClient.Update)
	mockClient.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(fakeClient.Patch)
	mockClient.EXPECT().Delete(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(fakeClient.Delete)
	return nil
}
