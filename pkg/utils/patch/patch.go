package patch

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type mergePatch struct {
	diff any
}

var _ client.Patch = &mergePatch{}

func NewPatch(diff any) client.Patch {
	return &mergePatch{
		diff: diff,
	}
}

func (m *mergePatch) Type() types.PatchType {
	return types.MergePatchType
}

func (m *mergePatch) Data(_ client.Object) ([]byte, error) {
	return json.Marshal(m.diff)
}
