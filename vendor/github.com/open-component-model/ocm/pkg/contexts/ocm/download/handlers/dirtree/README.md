# Directory Tree Downloader

The standard configuration now provides a downloader for resources of type `directoryTree`and the legacy type `filesystem`.

It acts on the mimetypes for an artifact set (`application/vnd.oci.image.manifest.v1+tar+gzip`) and a tar/tgz archive (`application/x-tar`, `application/x-tar+gzip`, `application/x-tgz`). The default configuration extracts the content to a
filesystem folder. If the blob format is an artifact set, for example provided by the access method `ociArtifact`,
the default configuration accepts the image config mimetype (`application/vnd.oci.image.config.v1+json`).
In this case the final filesystem content provided by the image is downloaded by evaluating the layered image file system.

The default behaviour can just be  used with
<pre>
import "github.com/open-component-model/ocm/pkg/contexts/ocm/download"
download.For(octx).Download(printer,resourceAccess,targetdir,vfs)
</pre>

If the resource access describes an appropriate artifact, the new handler is automatically selected.

As usual, the target is always a virtual filesystem.

Like all download handlers. the dirtree download handler can also explicitly be used with

<pre>
import "github.com/open-component-model/ocm/pkg/contexts/ocm/download/handlers/dirtree"
dirtree.New().Download(...)
</pre>

In this mode, the behaviour can be influenced by specifying any list of accepted OCI artifact config mime types.

With `New(mimetypes ...string).SetArchiveMode(true)` it is possible to enable the archive target mode. The content is downloaded to an archive instead of extracted filesystem content.

The handler checks the mimetypes, only, but the default registration is done exclusively for the directory content resource types.
A context can be extended for other resource types with

```
download.For(octx),Register(dirtree.New...(...), download.ForCombi(type, [mimetype]))
```

The handler also provides additional methods, which can be used to execute more specific tasks, for example
methods for an optimized content access providing an internal virtual filesystem or an archive byte stream trying to avoid unnecessary conversions depending on the actual input format,

The localization package has been adapted, accordingly. The `localize.Instantiate` method now prefers to use the
download handlers to get to the filesystem content to be configured, instead of expecting an archive resource.
Therefore, any (potentially own) resource type with any format can be used, as long as there is an appropriate downloader configured for the used OCM context.  An optional additional parameter can be used to restrict the accepted resource types
to an explicitly given set of types.

## Use cases

If you use `dirtree.New(...).Download(...)` you explicitly use the `dirtree` downloader and nothing else.
It only checks the mime types, but not the resource types. So, you can enforce to use it on resources,
regardless of their type to download dirtree-like  resources (with a matching mime type).

If you use `download.For(...).Download(...)` it tries to find a registered downloader with registration
criteria matching the actual resource. This can be used without bothering with the kind of actually used
resource (to just download it, whatever it is in a standard manner). If a matching downloader is found,
it is used, otherwise just the blob is downloaded as provided by the access method. Here, for sure the
`dirtree` downloader is used for the standard scenarios (it is registered for). This is especially the
`directoryTree`  resource type with the tar-like mimetypes and the oci artifact archive mime type. But it
is not used for other scenarios.

So, if you want to use an own resource type (directly expressing the dedicated new meaning of a general
filesystem content), for example `gitOpsTemplate`, which is more expressive than just `directoryTree`. You
can
- either register the `dirtree` handler in advance for your OCM context and for this resource type at the
- registry (then it would automatically be chosen for all downloads using this context)
- or you know what you are doing, and explicitly call the `dirtree` downloader on such a resource.

If, for example `gitOpsTemplate` should be a standard resource type, we should add such a registration as
part of the standard.

Another possible scenario, where you might want to use the explicit `dirtree` usage is to overwrite the
standard behaviour for a special use case. For example, an OCI image is typically downloaded as OCI
artifact with the distribution spec format. But. if you want to access the effective filesystem, you
could explicitly use the `dirtree` downloader for an OCI image, which handles this for you. It would
make less sense to use it on a helm chart OCI artifact, because here the layers have a different meaning
than building a directory tree.

## Registration Handler

It provides a registration handler with the path `ocm/dirtree`. and a config
object with the fields:
- *`asArchive`* *bool*: download as archive (default is directory tree).
- *`ociConfigtypes`* *[]string*: list of accepted OCI manifest config media types. Default is the OCI image config media type.