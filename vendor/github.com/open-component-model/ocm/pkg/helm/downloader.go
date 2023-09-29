// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"strings"

	"github.com/mandelsoft/filepath/pkg/filepath"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/repo"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/repositories/directcreds"
	"github.com/open-component-model/ocm/pkg/contexts/oci"
	ocihelm "github.com/open-component-model/ocm/pkg/contexts/ocm/download/handlers/helm"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/helm/identity"
	"github.com/open-component-model/ocm/pkg/runtime"
)

type chartDownloader struct {
	*downloader.ChartDownloader
	*chartAccess
	creds   common.Properties
	keyring []byte
}

func DownloadChart(out common.Printer, ctx oci.ContextProvider, ref, version, repourl string, opts ...Option) (ChartAccess, error) {
	repourl = strings.TrimSuffix(repourl, "/")

	acc, err := newTempChartAccess(osfs.New())
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			acc.Close()
		}
	}()

	s := cli.EnvSettings{}

	dl := &chartDownloader{
		ChartDownloader: &downloader.ChartDownloader{
			Out:     out,
			Getters: getter.All(&s),
		},
		chartAccess: acc,
	}
	for _, o := range opts {
		err = o.apply(dl)
		if err != nil {
			return nil, err
		}
	}

	err = dl.complete(ctx, ref, repourl)
	if err != nil {
		return nil, err
	}

	chart := ""
	prov := ""
	aset := ""
	if registry.IsOCI(repourl) {
		fs := osfs.New()
		chart = vfs.Join(fs, dl.root, filepath.Base(ref)+".tgz")
		creds := directcreds.NewCredentials(dl.creds)
		chart, prov, aset, err = ocihelm.Download2(out, ctx.OCIContext(), identity.OCIRepoURL(repourl, ref)+":"+version, chart, osfs.New(), true, creds)
		if prov != "" && dl.Verify > downloader.VerifyNever && dl.Verify != downloader.VerifyLater {
			_, err = downloader.VerifyChart(chart, dl.Keyring)
			if err != nil {
				// Fail always in this case, since it means the verification step
				// failed.
				return nil, err
			}
		}
	} else {
		chart, _, err = dl.DownloadTo("repo/"+ref, version, dl.root)
		prov = chart + ".prov"
	}
	if err != nil {
		return nil, err
	}
	if prov != "" && filepath.Exists(prov) {
		dl.prov = prov
	}
	dl.chart = chart
	dl.aset = aset
	return dl.chartAccess, nil
}

func (d *chartDownloader) complete(ctx oci.ContextProvider, ref, repourl string) error {
	rf := repo.NewFile()

	creds := d.creds
	if d.creds == nil {
		d.creds = identity.GetCredentials(ctx.OCIContext(), repourl, ref)
		if d.creds == nil {
			creds = common.Properties{}
		}
	}

	config := vfs.Join(d.fs, d.root, ".config")
	err := d.fs.MkdirAll(config, 0o700)
	if err != nil {
		return err
	}
	if len(d.keyring) != 0 {
		err = d.writeFile("keyring", config, &d.Keyring, d.keyring, "keyring file")
		if err != nil {
			return err
		}
		d.Verify = downloader.VerifyIfPossible
	}

	if registry.IsOCI(repourl) {
		return nil
	}
	entry := repo.Entry{
		Name:     "repo",
		URL:      repourl,
		Username: creds[identity.ATTR_USERNAME],
		Password: creds[identity.ATTR_PASSWORD],
	}

	cache := vfs.Join(d.fs, d.root, ".cache")
	err = d.fs.MkdirAll(cache, 0o700)
	if err != nil {
		return err
	}

	if len(creds[identity.ATTR_CERTIFICATE_AUTHORITY]) != 0 {
		err = d.writeFile("cacert", config, &entry.CAFile, []byte(creds[identity.ATTR_CERTIFICATE_AUTHORITY]), "CA file")
		if err != nil {
			return err
		}
	}
	if len(creds[identity.ATTR_CERTIFICATE]) != 0 {
		err = d.writeFile("cert", config, &entry.CertFile, []byte(creds[identity.ATTR_CERTIFICATE]), "certificate file")
		if err != nil {
			return err
		}
	}
	if len(creds[identity.ATTR_PRIVATE_KEY]) != 0 {
		err = d.writeFile("private-key", config, &entry.KeyFile, []byte(creds[identity.ATTR_PRIVATE_KEY]), "private key file")
		if err != nil {
			return err
		}
	}
	rf.Add(&entry)

	cr, err := repo.NewChartRepository(&entry, d.Getters)
	if err != nil {
		return errors.Wrapf(err, "cannot get chart repository %q", repourl)
	}

	d.RepositoryCache, cr.CachePath = cache, cache

	_, err = cr.DownloadIndexFile()
	if err != nil {
		return errors.Wrapf(err, "cannot download repository index for %q", repourl)
	}

	data, err := runtime.DefaultYAMLEncoding.Marshal(rf)
	if err != nil {
		return errors.Wrapf(err, "cannot marshal repository file")
	}
	err = d.writeFile("repository", config, &d.RepositoryConfig, data, "repository config")
	if err != nil {
		return err
	}

	return nil
}

func (d *chartDownloader) writeFile(name, root string, path *string, data []byte, desc string) error {
	*path = vfs.Join(d.fs, root, name)
	err := vfs.WriteFile(d.fs, *path, data, 0o600)
	if err != nil {
		return errors.Wrapf(err, "cannot write %s %q", desc, *path)
	}
	return nil
}
