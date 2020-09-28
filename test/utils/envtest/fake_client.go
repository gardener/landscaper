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

package envtest

import (
	"bytes"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	mock_client "github.com/gardener/landscaper/pkg/utils/kubernetes/mock"
)

// NewFakeClientFromPath reads all landscaper related files from the given path adds them to the controller runtime's fake client.
func NewFakeClientFromPath(path string) (client.Client, *State, error) {
	objects := make([]runtime.Object, 0)
	state := &State{
		Installations: make(map[string]*lsv1alpha1.Installation),
		Executions:    make(map[string]*lsv1alpha1.Execution),
		DeployItems:   make(map[string]*lsv1alpha1.DeployItem),
		DataObjects:   make(map[string]*lsv1alpha1.DataObject),
		Targets:       make(map[string]*lsv1alpha1.Target),
		Secrets:       make(map[string]*corev1.Secret),
	}
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
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

		// template files
		tmpl, err := template.New("init").Funcs(templatingFunctions).Parse(string(data))
		if err != nil {
			return err
		}
		buf := bytes.NewBuffer([]byte{})
		if err := tmpl.Execute(buf, map[string]string{"Namespace": state.Namespace}); err != nil {
			return err
		}

		objects, err = decodeAndAppendLSObject(buf.Bytes(), objects, state)
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

// RegisterFakeClientToMock adds fake client calls to a mockclient
func RegisterFakeClientToMock(mockClient *mock_client.MockClient, fakeClient client.Client) error {
	mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(fakeClient.Get)
	mockClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(fakeClient.Create)
	mockClient.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(fakeClient.Update)
	mockClient.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(fakeClient.Patch)
	mockClient.EXPECT().Delete(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(fakeClient.Delete)
	return nil
}
