// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package envtest

import (
	"bytes"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/gardener/landscaper/pkg/api"
	mock_client "github.com/gardener/landscaper/pkg/utils/kubernetes/mock"
)

// NewFakeClientFromPath reads all landscaper related files from the given path adds them to the controller runtime's fake client.
func NewFakeClientFromPath(path string) (client.Client, *State, error) {
	objects := make([]client.Object, 0)
	state := NewState()
	if len(path) != 0 {
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
	}

	return fake.NewClientBuilder().WithScheme(api.LandscaperScheme).WithObjects(objects...).Build(), state, nil
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
