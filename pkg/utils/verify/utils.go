package verify

import (
	"context"
	"errors"
	"fmt"

	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lutil "github.com/gardener/landscaper/controller-utils/pkg/landscaper"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PublicKeyData []byte

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

func ExtractVerifyInfo(ctx context.Context, inst *lsv1alpha1.Installation, client client.Client) (string, PublicKeyData, error) {
	if inst.Spec.Verification == nil {
		return "", nil, errors.New("installation.Spec.Verification cant be nil")
	}

	signatureName := inst.Spec.Verification.SignatureName
	if signatureName == "" {
		return "", nil, errors.New("installation.Spec.Verification.SignatureName must be set")

	}

	_, publicKeyData, _, err := lutil.ResolveSecretReference(ctx, client, &inst.Spec.Verification.PublicKeySecretReference)
	if err != nil {
		return "", nil, fmt.Errorf("failed resolving public key from reference: %w", err)
	}
	if len(publicKeyData) == 0 {
		return "", nil, errors.New("installation.Spec.Verification.publicKeySecretReference referenced public key is empty")
	}

	return signatureName, publicKeyData, nil
}
