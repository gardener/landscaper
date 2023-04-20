package ocm

//
//import (
//	"github.com/open-component-model/ocm/pkg/contexts/ocm"
//	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
//	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
//)
//
//type FakeComponentVersionAccess struct {
//	context ocm.Context
//}
//
//func (f FakeComponentVersionAccess) GetName() string {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (f FakeComponentVersionAccess) GetVersion() string {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (f FakeComponentVersionAccess) Repository() ocm.Repository {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (f FakeComponentVersionAccess) GetContext() ocm.Context {
//	return f.context
//}
//
//func (f FakeComponentVersionAccess) GetDescriptor() *compdesc.ComponentDescriptor {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (f FakeComponentVersionAccess) GetResources() []ocm.ResourceAccess {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (f FakeComponentVersionAccess) GetResource(meta metav1.Identity) (ocm.ResourceAccess, error) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (f FakeComponentVersionAccess) GetResourceByIndex(i int) (ocm.ResourceAccess, error) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (f FakeComponentVersionAccess) GetResourcesByName(name string, selectors ...compdesc.IdentitySelector) ([]ocm.ResourceAccess, error) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (f FakeComponentVersionAccess) GetSources() []ocm.SourceAccess {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (f FakeComponentVersionAccess) GetSource(meta metav1.Identity) (ocm.SourceAccess, error) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (f FakeComponentVersionAccess) GetSourceByIndex(i int) (ocm.SourceAccess, error) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (f FakeComponentVersionAccess) GetReference(meta metav1.Identity) (ocm.ComponentReference, error) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (f FakeComponentVersionAccess) GetReferenceByIndex(i int) (ocm.ComponentReference, error) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (f FakeComponentVersionAccess) AccessMethod(spec ocm.AccessSpec) (ocm.AccessMethod, error) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (f FakeComponentVersionAccess) AddBlob(blob ocm.BlobAccess, artType, refName string, global ocm.AccessSpec) (ocm.AccessSpec, error) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (f FakeComponentVersionAccess) SetResourceBlob(meta *ocm.ResourceMeta, blob ocm.BlobAccess, refname string, global ocm.AccessSpec) error {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (f FakeComponentVersionAccess) SetResource(meta *ocm.ResourceMeta, spec compdesc.AccessSpec) error {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (f FakeComponentVersionAccess) AdjustResourceAccess(meta *ocm.ResourceMeta, acc compdesc.AccessSpec) error {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (f FakeComponentVersionAccess) SetSourceBlob(meta *ocm.SourceMeta, blob ocm.BlobAccess, refname string, global ocm.AccessSpec) error {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (f FakeComponentVersionAccess) SetSource(meta *ocm.SourceMeta, spec compdesc.AccessSpec) error {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (f FakeComponentVersionAccess) SetReference(ref *ocm.ComponentReference) error {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (f FakeComponentVersionAccess) DiscardChanges() {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (f FakeComponentVersionAccess) Dup() (ocm.ComponentVersionAccess, error) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (f FakeComponentVersionAccess) Close() error {
//	//TODO implement me
//	panic("implement me")
//}
//
//type fakeBaseAccess struct {
//}
