# Bitbucket Enforcer

Daemon to ensure various defaults when repositories are created on Bitbucket.

`bitbucket-enforcer` relies upon the fact that newly created repositories won't
have any of the settings it can manage already applied to them. Thus, they
should be safe to modify at will.

When `bitbucket-enforcer` has enforced the specified defaults, it will add
`-defaults-enforced` to the comment field of the repository so the repository
won't be changed again.

`bitbucket-enforcer` is not destructive, so it won't remove "extra" data, such as
deploy keys that are present in the repository settings but not in the policy file.


## Planned Features

  - [X] Access management
  - [X] Branch management
  - [X] Deployment keys
  - [X] Hooks
  - [X] Public issue tracker settings
  - [X] Overriding enforcement type
  - [X] Forking policy
  - [X] Repository privacy
  - [ ] New Bitbucket Webhooks

## Configuration

The `bitbucket-enforcer` tool uses an Bitbucket username and API key to
communicate with the Bitbucket API. These are read from the
`BITBUCKET_ENFORCER_USERNAME` and `BITBUCKET_ENFORCER_API_KEY` environment
variables. `bitbucket-enforcer` supports [`.env`
files](https://www.github.com/joho/godotenv).

Enforcement policy configuration files should be placed in the `config` folder.

    $ tree config
    configs
    ├── default.json
    └── some-project-type.json

    0 directories, 2 files

Each file in this folder specifies an enforcement type. The files are JSON files
and should contain each of the following settings that are applicable. See
`configs/default.json` for details.

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

Error messages might be bad. They are copied verbatim from Bitbucket, and some of
them contain HTML, some of them JSON-strings and some might contain something else
entirely.

Main branches are not enforced. `bitbucket-enforcer` is meant to be polling for new
repositories often, so as to enforce policies as soon as a repository is created.
At this point, there will probably be no branches in the repository, which means
that a main branch cannot be set.

Groups that are allowed to push to a branch in a repository are at the moment
assumed to be owned by the repository owner.

The Bitbucket API doesn't seem to support having private issue trackers.
Unfortunately the only settings available are thus public issue tracker or no issue
tracker.
