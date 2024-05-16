// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package ocmlib

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/utils"

	v2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/signing"
	"github.com/open-component-model/ocm/pkg/signing/handlers/rsa"
	"github.com/open-component-model/ocm/pkg/signing/signutils"

	"github.com/gardener/landscaper/pkg/components/model/types"

	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/runtime"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/model"
	_ "github.com/gardener/landscaper/pkg/components/ocmlib/repository/inline"
	_ "github.com/gardener/landscaper/pkg/components/ocmlib/repository/local"
)

type RegistryAccess struct {
	octx             ocm.Context
	session          ocm.Session
	inlineSpec       ocm.RepositorySpec
	inlineRepository ocm.Repository
	resolver         ocm.ComponentVersionResolver
}

var _ model.RegistryAccess = (*RegistryAccess)(nil)

func (r *RegistryAccess) NewComponentVersion(cv ocm.ComponentVersionAccess) (model.ComponentVersion, error) {
	if cv == nil {
		return nil, errors.New("component version access cannot be nil during facade component version creation")
	}
	// Get ocm-lib Component Descriptor
	cd := cv.GetDescriptor()

	// TODO: Remove this check
	// this is only included for compatibility reasons as the legacy ocm spec mandated component descriptors to have a
	// repository context
	if len(cd.RepositoryContexts) == 0 {
		return nil, fmt.Errorf("repository context is required")
	}
	data, err := compdesc.Encode(cd, compdesc.SchemaVersion(v2.SchemaVersion))
	if err != nil {
		return nil, err
	}

	// Create Landscaper Component Descriptor from the ocm-lib Component Descriptor
	lscd := types.ComponentDescriptor{}
	err = runtime.DefaultYAMLEncoding.Unmarshal(data, &lscd)
	if err != nil {
		return nil, err
	}

	return &ComponentVersion{
		registryAccess:         r,
		componentVersionAccess: cv,
		componentDescriptorV2:  lscd,
	}, nil
}

func (r *RegistryAccess) VerifySignature(componentVersion model.ComponentVersion, name string, pkeyData []byte, caCertData []byte) error {
	verificationOptions := []signing.Option{
		signing.Recursive(true),
	}

	if pkeyData != nil {
		pubkey, err := rsa.ParsePublicKey(pkeyData)
		if err != nil {
			return fmt.Errorf("failed parsing public key data: %w", err)
		}
		verificationOptions = append(verificationOptions, signing.PublicKey(name, pubkey))
	}

	if caCertData != nil {
		cert, err := signutils.ParseCertificate(caCertData)
		if err != nil {
			return fmt.Errorf("failed parsing ca cert data: %w", err)
		}
		_, certPool, err := signutils.GetCertificate(cert, false)
		if err != nil {
			return fmt.Errorf("failed generating ca cert pool: %w", err)
		}
		verificationOptions = append(verificationOptions, signing.RootCertificates(certPool))
	}

	castedComponentVersion, ok := componentVersion.(*ComponentVersion)
	if !ok {
		return errors.New("failed casting componentVersion interface to ocm.ComponentVersion")
	}
	_, err := signing.VerifyComponentVersion(castedComponentVersion.GetOCMObject(), name, verificationOptions...)
	if err != nil {
		return fmt.Errorf("failed verifying signature: %w", err)
	}
	return nil
}

func (r *RegistryAccess) GetComponentVersion(ctx context.Context, cdRef *lsv1alpha1.ComponentDescriptorReference) (_ model.ComponentVersion, rerr error) {
	logger, _ := logging.FromContextOrNew(ctx, nil)
	if cdRef != nil {
		logger = logger.WithValues("componentRefName", cdRef.ComponentName, "componentRefVersion", cdRef.Version)
	}
	pm := utils.StartPerformanceMeasurement(&logger, "GetComponentVersion")
	defer pm.StopDebug()

	if cdRef == nil {
		return nil, errors.New("component descriptor reference cannot be nil")
	}

	var resolver ocm.ComponentVersionResolver

	if cdRef.RepositoryContext != nil {
		spec, err := r.octx.RepositorySpecForConfig(cdRef.RepositoryContext.Raw, runtime.DefaultYAMLEncoding)
		if err != nil {
			return nil, err
		}

		// check if repository context from inline component descriptor should be used
		if r.inlineRepository != nil && reflect.DeepEqual(spec, r.inlineSpec) {
			// in this case, resolver knows an inline repository as well as the repository specified by the repository
			// context of the inline component descriptor
			resolver = r.resolver
		} else {
			pm1 := utils.StartPerformanceMeasurement(&logger, "GetComponentVersion-LookupRepository")
			// if there is no inline repository or the repository context is different from the one specified in the inline
			// component descriptor, we need to look up the repository specified by the component descriptor reference

			// if rule-a.prio > rule-b.prio, then rule-a is preferred
			// ensure, that this has the highest prio (int(^uint(0)>>1) == MaxInt), since the component version
			// overwrite depends on that
			r.octx.AddResolverRule("", spec, int(^uint(0)>>1))
			resolver = r.octx.GetResolver()
			pm1.StopDebug()
		}
	} else {
		pm1 := utils.StartPerformanceMeasurement(&logger, "GetComponentVersion-LookupRepository")
		resolver = r.octx.GetResolver()
		pm1.StopDebug()
	}

	if resolver == nil {
		return nil, errors.New("no repository or ocm resolvers found")
	}

	pm2 := utils.StartPerformanceMeasurement(&logger, "GetComponentVersion-LookupComponentVersion")
	cv, err := r.session.LookupComponentVersion(resolver, cdRef.ComponentName, cdRef.Version)
	pm2.StopDebug()
	if err != nil {
		return nil, err
	}

	return r.NewComponentVersion(cv)
}

func (r *RegistryAccess) Close() error {
	err := r.session.Close()
	if err != nil {
		return err
	}
	return nil
}
