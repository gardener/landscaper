package verify

import (
	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// IsVerifyEnabled returns if verification is enabled.
// The following rules apply:
// 1. if LandscaperConfiguration.EnforceSignatureVerification is true, always return true
// 2. else, if Installation.Spec.Verification.Enabled is true, return true.
// 3. otherwise, verify is not enabled and therefore return false
func IsVerifyEnabled(inst *lsv1alpha1.Installation, config *config.LandscaperConfiguration) bool {
	if config.EnforceSignatureVerification {
		return true
	}
	if inst.Spec.Verification != nil && inst.Spec.Verification.Enabled {
		return true
	}
	return false
}
