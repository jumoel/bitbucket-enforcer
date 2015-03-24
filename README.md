# Bitbucket Enforcer

Daemon to ensure various defaults when repositories are created on Bitbucket.

`bitbucket-enforcer` relies upon the fact that newly created repositories won't
have any of the settings it can manage already applied to them. Thus, they
should be safe to modify at will.

When `bitbucket-enforcer` has enforced the specified defaults, it will add
`-defaults-enforced` to the comment field of the repository so the repository
won't be changed again.


Plans to support:

  - [X] Landing page
  - [ ] Ensure presence of branches (requires `git` to be installed?)
  - [X] Main branch (only existing branches can be set)
  - [ ] Access management
  - [ ] Branch management
  - [X] Deployment keys (ensures presence of keys the defined content, potentially recreating to enforce names)
  - [ ] Hooks
  - [ ] Issue tracker settings
  - [ ] Overriding enforcement type
  - [X] Forking policy
  - [X] Repository privacy

## Configuration

The `bitbucket-enforcer` tool uses an OAuth consumer key and password to
communicate with the Bitbucket API. These are read from the
`BITBUCKET_ENFORCE_KEY` and `BITBUCKET_ENFORCE_PASS` environment variables.
`bitbucket-enforcer` supports [`.env` files](https://www.github.com/joho/godotenv).

Enforcement policy configuration files should be placed in the `config` folder.

    $ tree config
    configs
    ├── default.json
    └── some-project-type.json

    0 directories, 2 files

Each file in this folder specifies an enforcement type. The files are JSON files
and should contain each of the following settings that are applicable. See
`configs/default.json.example` for details.

If a setting doesn't match the specifications or isn't present, it is ignored.

## Overriding enforcement type

`bitbucket-enforcer` supports tags in the repository description field. This can be
used to override the default behaviour, which is to enforce `default` settings.

  * `-noenforce` in the repository comment field tells `bitbucket-enforce` to
    leave the repository alone
  * `-enforce=some-type` uses the `some-type` settings instead of `default`

In both cases, the tag will be removed from the description field and replaced
with `-defaults-enforced`

## Limitations

Error messages might be bad. They are copied verbatim from bitbucket, and some of
them contain HTML, some of them JSON-strings and some might contain something else
entirely.
