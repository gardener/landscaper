// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/log"
	"github.com/pkg/errors"

	"github.com/open-component-model/ocm/pkg/docker/resolve"
)

var ErrObjectNotRequired = errors.New("object not required")

type TagList struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

type dockerLister struct {
	dockerBase *dockerBase
}

func (r *dockerResolver) Lister(ctx context.Context, ref string) (resolve.Lister, error) {
	base, err := r.resolveDockerBase(ref)
	if err != nil {
		return nil, err
	}
	if base.refspec.Object != "" {
		return nil, ErrObjectNotRequired
	}

	return &dockerLister{
		dockerBase: base,
	}, nil
}

func (r *dockerLister) List(ctx context.Context) ([]string, error) {
	refspec := r.dockerBase.refspec
	base := r.dockerBase
	var (
		firstErr error
		paths    [][]string
		caps     = HostCapabilityPull
	)

	// turns out, we have a valid digest, make a url.
	paths = append(paths, []string{"tags/list"})
	caps |= HostCapabilityResolve

	hosts := base.filterHosts(caps)
	if len(hosts) == 0 {
		return nil, errors.Wrap(errdefs.ErrNotFound, "no list hosts")
	}

	ctx, err := ContextWithRepositoryScope(ctx, refspec, false)
	if err != nil {
		return nil, err
	}

	for _, u := range paths {
		for _, host := range hosts {
			ctxWithLogger := log.WithLogger(ctx, log.G(ctx).WithField("host", host.Host))

			req := base.request(host, http.MethodGet, u...)
			if err := req.addNamespace(base.refspec.Hostname()); err != nil {
				return nil, err
			}

			req.header["Accept"] = []string{"application/json"}

			log.G(ctxWithLogger).Debug("listing")
			resp, err := req.doWithRetries(ctxWithLogger, nil)
			if err != nil {
				if errors.Is(err, ErrInvalidAuthorization) {
					err = errors.Wrapf(err, "pull access denied, repository does not exist or may require authorization")
				}
				// Store the error for referencing later
				if firstErr == nil {
					firstErr = err
				}
				log.G(ctxWithLogger).WithError(err).Info("trying next host")
				continue // try another host
			}

			if resp.StatusCode > 299 {
				resp.Body.Close()
				if resp.StatusCode == http.StatusNotFound {
					log.G(ctxWithLogger).Info("trying next host - response was http.StatusNotFound")
					continue
				}
				if resp.StatusCode > 399 {
					// Set firstErr when encountering the first non-404 status code.
					if firstErr == nil {
						firstErr = errors.Errorf("pulling from host %s failed with status code %v: %v", host.Host, u, resp.Status)
					}
					continue // try another host
				}
				return nil, errors.Errorf("taglist from host %s failed with unexpected status code %v: %v", host.Host, u, resp.Status)
			}

			data, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				return nil, err
			}

			tags := &TagList{}

			err = json.Unmarshal(data, tags)
			if err != nil {
				return nil, err
			}
			return tags.Tags, nil
		}
	}

	// If above loop terminates without return, then there was an error.
	// "firstErr" contains the first non-404 error. That is, "firstErr == nil"
	// means that either no registries were given or each registry returned 404.

	if firstErr == nil {
		firstErr = errors.Wrap(errdefs.ErrNotFound, base.refspec.Locator)
	}

	return nil, firstErr
}
