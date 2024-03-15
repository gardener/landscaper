package verify

import (
	"context"
	"errors"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lutil "github.com/gardener/landscaper/controller-utils/pkg/landscaper"
)

type PublicKeyData []byte

// IsVerifyEnabled returns if verification is enabled.
// The following rules apply:
// 1. if lsConfig.SignatureVerificationEnforcementPolicy is Enforce, always return true
// 2. if lsConfig.SignatureVerificationEnforcementPolicy is Disabled, always return false even if installation.Spec.Verification is true.
// 3. else, if SignatureVerificationEnforcementPolicy is DoNotEnforce return Installation.Spec.Verification.Enabled
// 3. otherwise should not happen, return true as a safe fallback
func IsVerifyEnabled(inst *lsv1alpha1.Installation, lsConfig *config.LandscaperConfiguration) bool {
	switch lsConfig.SignatureVerificationEnforcementPolicy {
	case config.Enforce:
		return true
	case config.Disabled:
		return false
	case config.DoNotEnforce:
		return inst.Spec.Verification != nil && inst.Spec.Verification.Enabled
	default:
		//all cases should be handled above, so return true as failsafe
		return true
	}
}

func ExtractVerifyInfo(ctx context.Context, inst *lsv1alpha1.Installation, installationContext lsv1alpha1.Context, client client.Client) (string, PublicKeyData, error) {
	if inst.Spec.Verification == nil {
		return "", nil, errors.New("installation.Spec.Verification cant be nil")
	}

	signatureName := inst.Spec.Verification.SignatureName
	if signatureName == "" {
		return "", nil, errors.New("installation.Spec.Verification.SignatureName must be set")

	}

	publicKeySecretReference, ok := installationContext.VerificationSignatures[signatureName]
	if !ok {
		return "", nil, fmt.Errorf("context.VerificationSignatures does not contain a key for signature name '%v'", signatureName)
	}

	_, publicKeyData, _, err := lutil.ResolveSecretReference(ctx, client, &publicKeySecretReference.PublicKeySecretReference)
	if err != nil {
		return "", nil, fmt.Errorf("failed resolving public key from reference: %w", err)
	}
	if len(publicKeyData) == 0 {
		return "", nil, errors.New("installation.Spec.Verification.publicKeySecretReference referenced public key is empty")
	}

	return signatureName, publicKeyData, nil
}
