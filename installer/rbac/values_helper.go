package rbac

import (
	"fmt"
	"github.com/gardener/landscaper/installer/shared"
	"maps"
)

const (
	appNameLandscaperRBAC = "landscaper-rbac"
)

type valuesHelper struct {
	values *Values
}

func newValuesHelper(values *Values) (*valuesHelper, error) {
	if values == nil {
		return nil, fmt.Errorf("values must not be nil")
	}

	return &valuesHelper{
		values: values,
	}, nil
}

func (h *valuesHelper) appAndInstance() string {
	return fmt.Sprintf("%s-%s", appNameLandscaperRBAC, h.values.Instance)
}

func (h *valuesHelper) resourceNamespace() string {
	return h.values.Instance.Namespace()
}

func (h *valuesHelper) clusterRoleNameController() string {
	return h.values.Instance.ClusterScopedResourceName("controller")
}

func (h *valuesHelper) clusterRoleNameUser() string {
	return h.values.Instance.ClusterScopedResourceName("user")
}

func (h *valuesHelper) clusterRoleNameWebhooks() string {
	return h.values.Instance.ClusterScopedResourceName("webhooks")
}

func (h *valuesHelper) landscaperLabels() map[string]string {
	labels := map[string]string{
		shared.LabelVersion:   h.values.Version,
		shared.LabelManagedBy: shared.LabelValueManagedBy,
	}
	maps.Copy(labels, h.selectorLabels())
	return labels
}

func (h *valuesHelper) selectorLabels() map[string]string {
	return map[string]string{
		shared.LabelAppName:     appNameLandscaperRBAC,
		shared.LabelAppInstance: h.appAndInstance(),
	}
}

func (h *valuesHelper) isCreateServiceAccount() bool {
	return h.values.ServiceAccount != nil && h.values.ServiceAccount.Create
}
