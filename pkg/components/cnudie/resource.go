package cnudie

import (
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/landscaper/pkg/components/model"
)

type Resource struct {
	resource *cdv2.Resource
}

var _ model.Resource = &Resource{}

func newResource(res *cdv2.Resource) *Resource {
	return &Resource{
		resource: res,
	}
}

func (r Resource) GetDescriptor() ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (r Resource) GetBlob() ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (r Resource) GetBlobInfo() (model.BlobInfo, error) {
	//TODO implement me
	panic("implement me")
}
