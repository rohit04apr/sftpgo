# SFTPGo

[![CI Status](https://github.com/drakkan/sftpgo/workflows/CI/badge.svg?branch=main&event=push)](https://github.com/drakkan/sftpgo/workflows/CI/badge.svg?branch=main&event=push)
[![Code Coverage](https://codecov.io/gh/drakkan/sftpgo/branch/main/graph/badge.svg)](https://codecov.io/gh/drakkan/sftpgo/branch/main)
[![License: AGPL-3.0-only](https://img.shields.io/badge/License-AGPLv3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)
[![Docker Pulls](https://img.shields.io/docker/pulls/drakkan/sftpgo)](https://hub.docker.com/r/drakkan/sftpgo)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go)

[English](./README.md) | [简体中文](./README.zh_CN.md)

Fully featured and highly configurable SFTP server with optional HTTP/S, FTP/S and WebDAV support.
Several storage backends are supported: local filesystem, encrypted local filesystem, S3 (compatible) Object Storage, Google Cloud Storage, Azure Blob Storage, SFTP.

## Sponsors

If you find SFTPGo useful please consider supporting this Open Source project.

Maintaining and evolving SFTPGo is a lot of work - easily the equivalent of a full time job - for me.

I'd like to make SFTPGo into a sustainable long term project and would not like to introduce a dual licensing option and limit some features to the proprietary version only.

If you use SFTPGo, it is in your best interest to ensure that the project you rely on stays healthy and well maintained.
This can only happen with your donations and [sponsorships](https://github.com/sponsors/drakkan) :heart:

With sponsorships/donations we establish a channel for reciprocal access, ensuring better outcomes for both you and the project.

If you just take and don't return anything back, the project will die in the long run and you will be forced to pay for a similar proprietary solution.

More [info](https://github.com/drakkan/sftpgo/issues/452).

You can also purchase support plans from the [SFTPGo website](https://sftpgo.com/#pricing).

### Thank you to our sponsors

#### Platinum sponsors

[<img src="./img/Aledade_logo.png" alt="Aledade logo" width="202" height="70">](https://www.aledade.com/)

#### Silver sponsors

[<img src="./img/Dendi_logo.png" alt="Dendi logo" width="212" height="66">](https://dendisoftware.com/)

#### Bronze sponsors

[<img src="https://www.7digital.com/wp-content/themes/sevendigital/images/top_logo.png" alt="7digital logo">](https://www.7digital.com/)

## Support policy

SFTPGo is an Open Source project and you can of course use it for free but please don't ask for free support as well.

We will check the reported issues to see if you are experiencing a bug and if so, it may or may not be fixed, we only provide support to project [sponsors/donors](#sponsors).

If you report an invalid issue or ask for step-by-step support, your issue will remain open with no answer or will be closed as invalid without further explanation. Thanks for understanding.

## Features

- Support for serving local filesystem, encrypted local filesystem, S3 Compatible Object Storage, Google Cloud Storage, Azure Blob Storage or other SFTP accounts over SFTP/SCP/FTP/WebDAV.
- Virtual folders are supported: a virtual folder can use any of the supported storage backends. So you can have, for example, a user with the S3 backend mapping a GCS bucket (or part of it) on a specified path and an encrypted local filesystem on another one. Virtual folders can be private or shared among multiple users, for shared virtual folders you can define different quota limits for each user.
- Configurable [custom commands and/or HTTP hooks](./docs/custom-actions.md) on upload, pre-upload, download, pre-download, delete, pre-delete, rename, mkdir, rmdir on SSH commands and on user add, update and delete.
- Virtual accounts stored within a "data provider".
- SQLite, MySQL, PostgreSQL, CockroachDB, Bolt (key/value store in pure Go) and in-memory data providers are supported.
- Chroot isolation for local accounts. Cloud-based accounts can be restricted to a certain base path.
- Per-user and per-directory virtual permissions, for each path you can allow or deny: directory listing, upload, overwrite, download, delete, rename, create directories, create symlinks, change owner/group/file mode and modification time.
- [REST API](./docs/rest-api.md) for users and folders management, data retention, backup, restore and real time reports of the active connections with possibility of forcibly closing a connection.
- The [Event Manager](./docs/eventmanager.md) allows to define custom workflows based on server events or schedules.
- [Web based administration interface](./docs/web-admin.md) to easily manage users, folders and connections.
- [Web client interface](./docs/web-client.md) so that end users can change their credentials, manage and share their files in the browser.
- Public key and password authentication. Multiple public keys per-user are supported.
- SSH user [certificate authentication](https://cvsweb.openbsd.org/src/usr.bin/ssh/PROTOCOL.certkeys?rev=1.8).
- Keyboard interactive authentication. You can easily setup a customizable multi-factor authentication.
- Partial authentication. You can configure multi-step authentication requiring, for example, the user password after successful public key authentication.
- Per-user authentication methods.
- [Two-factor authentication](./docs/howto/two-factor-authentication.md) based on time-based one time passwords (RFC 6238) which works with Authy, Google Authenticator, Microsoft Authenticator and other compatible apps.
- Simplified user administrations using [groups](./docs/groups.md).
- [Roles](./docs/roles.md) allow to create limited administrators who can only create and manage users with their role.
- Custom authentication via [external programs/HTTP API](./docs/external-auth.md).
- Web Client and Web Admin user interfaces support [OpenID Connect](https://openid.net/connect/) authentication and so they can be integrated with identity providers such as [Keycloak](https://www.keycloak.org/). You can find more details [here](./docs/oidc.md).
- [Data At Rest Encryption](./docs/dare.md).
- Dynamic user modification before login via [external programs/HTTP API](./docs/dynamic-user-mod.md).
- Quota support: accounts can have individual disk quota expressed as max total size and/or max number of files.
- Bandwidth throttling, with separate settings for upload and download and overrides based on the client's IP address.
- Data transfer bandwidth limits, with total limit or separate settings for uploads and downloads and overrides based on the client's IP address. Limits can be reset using the REST API.
- Per-protocol [rate limiting](./docs/rate-limiting.md) is supported and can be optionally connected to the built-in defender to automatically block hosts that repeatedly exceed the configured limit.
- Per-user maximum concurrent sessions.
- Per-user and global IP filters: login can be restricted to specific ranges of IP addresses or to a specific IP address.
- Per-user and per-directory shell like patterns filters: files can be allowed, denied and optionally hidden based on shell like patterns.
- Automatically terminating idle connections.
- Automatic blocklist management using the built-in [defender](./docs/defender.md).
- Geo-IP filtering using a [plugin](https://github.com/sftpgo/sftpgo-plugin-geoipfilter).
- Atomic uploads are configurable.
- Per-user files/folders ownership mapping: you can map all the users to the system account that runs SFTPGo (all platforms are supported) or you can run SFTPGo as root user and map each user or group of users to a different system account (\*NIX only).
- Support for Git repositories over SSH.
- SCP and rsync are supported.
- FTP/S is supported. You can configure the FTP service to require TLS for both control and data connections.
- [WebDAV](./docs/webdav.md) is supported.
- ACME protocol is supported. SFTPGo can obtain and automatically renew TLS certificates for HTTPS, WebDAV and FTPS from `Let's Encrypt` or other ACME compliant certificate authorities, using the `HTTP-01` or `TLS-ALPN-01` [challenge types](https://letsencrypt.org/docs/challenge-types/).
- Two-Way TLS authentication, aka TLS with client certificate authentication, is supported for REST API/Web Admin, FTPS and WebDAV over HTTPS.
- Per-user protocols restrictions. You can configure the allowed protocols (SSH/HTTP/FTP/WebDAV) for each user.
- [Prometheus metrics](./docs/metrics.md) are supported.
- Support for HAProxy PROXY protocol: you can proxy and/or load balance the SFTP/SCP/FTP service without losing the information about the client's address.
- Easy [migration](./examples/convertusers) from Linux system user accounts.
- [Portable mode](./docs/portable-mode.md): a convenient way to share a single directory on demand.
- [SFTP subsystem mode](./docs/sftp-subsystem.md): you can use SFTPGo as OpenSSH's SFTP subsystem.
- Performance analysis using built-in [profiler](./docs/profiling.md).
- Configuration format is at your choice: JSON, TOML, YAML, HCL, envfile are supported.
- Log files are accurate and they are saved in the easily parsable JSON format ([more information](./docs/logs.md)).
- SFTPGo supports a [plugin system](./docs/plugins.md) and therefore can be extended using external plugins.
- Infrastructure as Code (IaC) support using the [Terraform provider](https://registry.terraform.io/providers/drakkan/sftpgo/latest).

## Platforms

SFTPGo is developed and tested on Linux. After each commit, the code is automatically built and tested on Linux, macOS, Windows and FreeBSD. Other *BSD variants should work too.

## Requirements

- Go as build only dependency. We support the Go version(s) used in [continuous integration workflows](./.github/workflows).
- A suitable SQL server to use as data provider:
  - upstream supported versions of PostgreSQL, MySQL and MariaDB.
  - CockroachDB stable.
- The SQL server is optional: you can choose to use an embedded SQLite, bolt or in memory data provider.

## Installation

Binary releases for Linux, macOS, and Windows are available. Please visit the [releases](https://github.com/drakkan/sftpgo/releases "releases") page.

An official Docker image is available. Documentation is [here](./docker/README.md).

<details>

<summary>Some Linux distro packages are available</summary>

- For Arch Linux via AUR:
  - [sftpgo](https://aur.archlinux.org/packages/sftpgo/). This package follows stable releases. It requires `git`, `gcc` and `go` to build.
  - [sftpgo-bin](https://aur.archlinux.org/packages/sftpgo-bin/). This package follows stable releases downloading the prebuilt linux binary from GitHub. It does not require `git`, `gcc` and `go` to build.
  - [sftpgo-git](https://aur.archlinux.org/packages/sftpgo-git/). This package builds and installs the latest git `main` branch. It requires `git`, `gcc` and `go` to build.
- Deb and RPM packages are built after each commit and for each release.
- For Ubuntu a PPA is available [here](https://launchpad.net/~sftpgo/+archive/ubuntu/sftpgo).
- Void Linux provides an [official package](https://github.com/void-linux/void-packages/tree/master/srcpkgs/sftpgo).

</details>

APT and YUM repositories are [available](./docs/repo.md).

SFTPGo is also available on some marketplaces:

- [AWS Marketplace](https://aws.amazon.com/marketplace/seller-profile?id=6e849ab8-70a6-47de-9a43-13c3fa849335)
- Azure Marketplace: [SFTPGo for Linux](https://azuremarketplace.microsoft.com/en-us/marketplace/apps/eliamarzia1667381463185.sftpgo_linux), [SFTPGo for Windows](https://azuremarketplace.microsoft.com/en-us/marketplace/apps/eliamarzia1667381463185.sftpgo_windows)
- [Elest.io](https://elest.io/open-source/sftpgo)

Purchasing from there will help keep SFTPGo a long-term sustainable project.

<details><summary>Windows packages</summary>

- The Windows installer to install and run SFTPGo as a Windows service.
- The portable package to start SFTPGo on demand.
- The [winget](https://docs.microsoft.com/en-us/windows/package-manager/winget/install) package to install and run SFTPGo as a Windows service: `winget install SFTPGo`.
- The [Chocolatey package](https://community.chocolatey.org/packages/sftpgo) to install and run SFTPGo as a Windows service.

</details>

On macOS you can install from the Homebrew [Formula](https://formulae.brew.sh/formula/sftpgo).
On FreeBSD you can install from the [SFTPGo port](https://www.freshports.org/ftp/sftpgo).
On DragonFlyBSD you can install SFTPGo from [DPorts](https://github.com/DragonFlyBSD/DPorts/tree/master/ftp/sftpgo).

You can easily test new features selecting a commit from the [Actions](https://github.com/drakkan/sftpgo/actions) page and downloading the matching build artifacts for Linux, macOS or Windows. GitHub stores artifacts for 90 days.

Alternately, you can [build from source](./docs/build-from-source.md).

[Getting Started Guide for the Impatient](./docs/howto/getting-started.md).

## Configuration

A full explanation of all configuration methods can be found [here](./docs/full-configuration.md).

Please make sure to [initialize the data provider](#data-provider-initialization-and-management) before running the daemon.

To start SFTPGo with the default settings, simply run:

```bash
sftpgo serve
```

Check out [this documentation](./docs/service.md) if you want to run SFTPGo as a service.

### Data provider initialization and management

Before starting the SFTPGo server please ensure that the configured data provider is properly initialized/updated.

For PostgreSQL, MySQL and CockroachDB providers, you need to create the configured database. For SQLite, the configured database will be automatically created at startup. Memory and bolt data providers do not require an initialization but they could require an update to the existing data after upgrading SFTPGo.

SFTPGo will attempt to automatically detect if the data provider is initialized/updated and if not, will attempt to initialize/ update it on startup as needed.

Alternately, you can create/update the required data provider structures yourself using the `initprovider` command.

For example, you can simply execute the following command from the configuration directory:

```bash
sftpgo initprovider
```

Take a look at the CLI usage to learn how to specify a different configuration file:

```bash
sftpgo initprovider --help
```

You can disable automatic data provider checks/updates at startup by setting the `update_mode` configuration key to `1`.

You can also reset your provider by using the `resetprovider` sub-command. Take a look at the CLI usage for more details:

```bash
sftpgo resetprovider --help
```

:warning: Please note that some data providers (e.g. MySQL and CockroachDB) do not support schema changes within a transaction, this means that you may end up with an inconsistent schema if migrations are forcibly aborted. CockroachDB doesn't support database-level locks, so make sure you don't execute migrations concurrently.

## Create the first admin

To start using SFTPGo you need to create an admin user, you can do it in several ways:

- by using the web admin interface. The default URL is [http://127.0.0.1:8080/web/admin](http://127.0.0.1:8080/web/admin)
- by loading initial data
- by enabling `create_default_admin` in your configuration file and setting the environment variables `SFTPGO_DEFAULT_ADMIN_USERNAME` and `SFTPGO_DEFAULT_ADMIN_PASSWORD`

## Upgrading

SFTPGo supports upgrading from the previous release branch to the current one.
Some examples for supported upgrade paths are:

- from 2.1.x to 2.2.x
- from 2.2.x to 2.3.x and so on.

For supported upgrade paths, the data and schema are migrated automatically when SFTPGo starts, alternatively you can use the `initprovider` command before starting SFTPGo.

So if, for example, you want to upgrade from 2.0.x to 2.2.x, you must first install version 2.1.x, update the data provider (automatically, by starting SFTPGo or manually using the `initprovider` command) and finally install the version 2.2.x. It is recommended to always install the latest available minor version, ie do not install 2.1.0 if 2.1.2 is available.

Loading data from a provider independent JSON dump is supported from the previous release branch to the current one too. After upgrading SFTPGo it is advisable to regenerate the JSON dump from the new version.

## Downgrading

If for some reason you want to downgrade SFTPGo, you may need to downgrade your data provider schema and data as well. You can use the `revertprovider` command for this task.

As for upgrading, SFTPGo supports downgrading from the previous release branch to the current one.

So, if you plan to downgrade from 2.3.x to 2.2.x, before uninstalling 2.3.x version, you can prepare your data provider executing the following command from the configuration directory:

```shell
sftpgo revertprovider
```

Take a look at the CLI usage to learn how to specify a configuration file:

```shell
sftpgo revertprovider --help
```

The `revertprovider` command is not supported for the memory provider.

Please note that we only support the current release branch and the current main branch, if you find a bug it is better to report it rather than downgrading to an older unsupported version.

## Users, groups, folders and other resource management

After starting SFTPGo you can manage users, groups, folders and other resources using:

- the [WebAdmin UI](./docs/web-admin.md)
- the [REST API](./docs/rest-api.md)

To support embedded data providers like `bolt` and `SQLite`, which do not support concurrent connections, we can't have a CLI that directly write users and other resources to the data provider, we always have to use the REST API.

Full details for users, groups, folders, admins and other resources are documented in the [OpenAPI](./openapi/openapi.yaml) schema. If you want to render the schema without importing it manually, you can explore it on [Stoplight](https://sftpgo.stoplight.io/docs/sftpgo/openapi.yaml).

:warning: SFTPGo users, groups and folders are virtual and therefore unrelated to the system ones. There is no need to create system-wide users and groups.

## Tutorials

Some step-to-step tutorials can be found inside the source tree [howto](./docs/howto "How-to") directory.

## Authentication options

<details><summary> External Authentication</summary>

Custom authentication methods can easily be added. SFTPGo supports external authentication modules, and writing a new backend can be as simple as a few lines of shell script. More information can be found [here](./docs/external-auth.md).

</details>

<details><summary> Keyboard Interactive Authentication</summary>

Keyboard interactive authentication is, in general, a series of questions asked by the server with responses provided by the client.
This authentication method is typically used for multi-factor authentication.

More information can be found [here](./docs/keyboard-interactive.md).

</details>

## Dynamic user creation or modification

A user can be created or modified by an external program just before the login. More information about this can be found [here](./docs/dynamic-user-mod.md).

## Custom Actions

SFTPGo allows you to configure custom commands and/or HTTP hooks to receive notifications about file uploads, deletions and several other events.

More information about custom actions can be found [here](./docs/custom-actions.md).

## Virtual folders

Directories outside the user home directory or based on a different storage provider can be mapped as virtual folders, more information [here](./docs/virtual-folders.md).

## Other hooks

You can get notified as soon as a new connection is established using the [Post-connect hook](./docs/post-connect-hook.md) and after each login using the [Post-login hook](./docs/post-login-hook.md).
You can use your own hook to [check passwords](./docs/check-password-hook.md).

## Storage backends

### S3/GCP/Azure

Each user can be mapped with a [S3 Compatible Object Storage](./docs/s3.md) /[Google Cloud Storage](./docs/google-cloud-storage.md)/[Azure Blob Storage](./docs/azure-blob-storage.md) bucket or a bucket virtual folder.

### SFTP backend

Each user can be mapped to another SFTP server account or a subfolder of it. More information can be found [here](./docs/sftpfs.md).

### Encrypted backend

Data at-rest encryption is supported via the [cryptfs backend](./docs/dare.md).

### HTTP/S backend

HTTP/S backend allows you to write your own custom storage backend by implementing a REST API. More information can be found [here](./docs/httpfs.md).

### Other Storage backends

Adding new storage backends is quite easy:

- implement the [Fs interface](./internal/vfs/vfs.go#L86 "interface for filesystem backends").
- update the user method `GetFilesystem` to return the new backend
- update the web interface and the REST API CLI
- add the flags for the new storage backed to the `portable` mode

Anyway, some backends require a pay per-use account (or they offer free account for a limited time period only). To be able to add support for such backends or to review pull requests, please provide a test account. The test account must be available for enough time to be able to maintain the backend and do basic tests before each new release.

## Brute force protection

SFTPGo supports a built-in [defender](./docs/defender.md).

Alternately you can use the [connection failed logs](./docs/logs.md) for integration in tools such as [Fail2ban](http://www.fail2ban.org/). Example of [jails](./fail2ban/jails) and [filters](./fail2ban/filters) working with `systemd`/`journald` are available in fail2ban directory.

## Account's configuration properties

Details information about account configuration properties can be found [here](./docs/account.md).

## Performance

SFTPGo can easily saturate a Gigabit connection on low end hardware with no special configuration, this is generally enough for most use cases.

More in-depth analysis of performance can be found [here](./docs/performance.md).

## Release Cadence

SFTPGo releases are feature-driven, we don't have a fixed time based schedule. As a rough estimate, you can expect 1 or 2 new releases per year.

## Acknowledgements

SFTPGo makes use of the third party libraries listed inside [go.mod](./go.mod).

We are very grateful to all the people who contributed with ideas and/or pull requests.

Thank you [ysura](https://www.ysura.com/) for granting me stable access to a test AWS S3 account.

## License

GNU AGPL-3.0-only
