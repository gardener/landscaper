# Generation of the Documentation Index

The [documentation index](../README.md) can be generated automatically. To do so, use `make generate-docs`. This is also part of `make generate`.

The generated index is based on the directories and files in the `docs` folder. 

Each _directory_ will correspond to a header. The name of this header is read from the meta file (see below). Directories without this meta file will be ignored entirely, as will subdirectories.

Each _markdown file_ (identified by the `.md` file extension) in the selected directories will generate a list entry with a link to that file. The name for the link is derived as follows: If the meta file in the markdown file's directory has an overwrite entry for this file, the value from there will be used. Otherwise, the first line of the markdown file will be used, stripping off a potential `#` prefix. If the resulting string is empty, the file will be ignored.

### The Meta File

Each subdirectory of `docs` which should be taken into account when generating the documentation index needs to have a so-called meta file. This file has to be named `.docnames` and it is expected to be in JSON format.

```json
{
  "header": "API Reference",
  "overwrites": {
    "core.md": "API Reference"
  }
}
```

The `header` part is mandatory. It describes the name of the header which will be used for this directory in the documentation index.

The `overwrites` mapping is optional and allows overwriting the entry name for any file in the directory. This is especially useful if some file doesn't start with a descriptive title in the first line. This feature can also be used to ignore a single file by mapping its filename to the empty string. Entries in the `overwrites` mapping which don't have a corresponding file will be ignored.
