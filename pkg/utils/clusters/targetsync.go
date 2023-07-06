package clusters

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
)

const (
	LabelKeyTargetSync     = lsv1alpha1.LandscaperDomain + "/targetsync"
	LabelValueTargetSyncOk = "ok"
)

// HasTargetSyncLabel returns whether an object has the targetsync label.
// The targets and secrets that are managed by the targetsync are marked with this label.
// Moreover, the setting "skipUninstallIfClusterRemoved" of deployitems is only supported for deployitems whose
// target is managed by the targetsync.
func HasTargetSyncLabel(obj metav1.Object) bool {
	return kubernetes.HasLabelWithValue(obj, LabelKeyTargetSync, LabelValueTargetSyncOk)
}
