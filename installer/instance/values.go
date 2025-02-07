package instance

import (
	"github.com/gardener/landscaper/installer/helmdeployer"
	"github.com/gardener/landscaper/installer/landscaper"
	"github.com/gardener/landscaper/installer/manifestdeployer"
	"github.com/gardener/landscaper/installer/rbac"
	"github.com/gardener/landscaper/installer/shared"
)

type Values struct {
	Instance               shared.Instance
	RBACValues             *rbac.Values
	LandscaperValues       *landscaper.Values
	ManifestDeployerValues *manifestdeployer.Values
	HelmDeployerValues     *helmdeployer.Values
}
