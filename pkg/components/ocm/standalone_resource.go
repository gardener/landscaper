// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package ocm

//
//import (
//	"context"
//	"io"
//
//	"github.com/open-component-model/ocm/pkg/contexts/ocm"
//
//	"github.com/gardener/landscaper/pkg/components/model"
//)
//
//type StandaloneResource struct {
//	compvers   ocm.ComponentVersionAccess
//	accessSpec ocm.AccessSpec
//}
//
//func (r StandaloneResource) GetName() string {
//	return ""
//}
//
//func (r StandaloneResource) GetVersion() string {
//	return r.accessSpec.GetVersion()
//}
//
//func (r StandaloneResource) GetDescriptor(ctx context.Context) ([]byte, error) {
//	return nil, nil
//}
//
//func (r StandaloneResource) GetBlob(ctx context.Context, writer io.Writer) error {
//	meth, err := r.accessSpec.AccessMethod(r.compvers)
//	if err != nil {
//		return err
//	}
//	defer meth.Close()
//
//	blob, err := meth.Get()
//	if err != nil {
//		return err
//	}
//	if _, err := writer.Write(blob); err != nil {
//		return err
//	}
//	return nil
//}
//
//func (r StandaloneResource) GetBlobInfo(ctx context.Context) (*model.BlobInfo, error) {
//	meth, err := r.accessSpec.AccessMethod(r.compvers)
//	if err != nil {
//		return nil, err
//	}
//	defer meth.Close()
//
//	mediatype := meth.MimeType()
//
//	return &model.BlobInfo{
//		MediaType: mediatype,
//	}, nil
//}
