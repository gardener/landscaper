package cnudie

import "github.com/gardener/landscaper/pkg/components/model"

type Resource struct {
}

var _ model.Resource = &Resource{}

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
