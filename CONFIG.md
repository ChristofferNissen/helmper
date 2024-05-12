# Configuration parameters overview

The configuration file `helmper.yaml` can be placed in: 

- Current directory (`.`)
- `$HOME/.config/helmper/`
- `/etc/helmper/`

## Configuration options

| Key | Type | Default | Description |
|-|-|-|-|
| `k8s_version` | string       | "1.27.9" |  Some charts use images eliciting their tag based on the kube-apiserver version. Therefore, tell Helmper which version you run to import the correct version. |
| `verbose`     | bool         | false    |  Toggle verbose output |
| `update`      | bool         | false    | Toggle update to latest chart version for each specified chart in `charts` |
| `all`         | bool         | false    |Toggle import of all images regardless if they exist in the registries defined in `registries` |
| `import`      | object       | nil      |  If import is enabled, images will be pushed to the defined registries. If copacetic is enabled, images will be patched if possible. Finally, in the import section Cosign can be configured to sign the images after pushing to the registries. See table blow for full configuration options. |
| `charts`      | list(object) | [] | Defines which charts to target |
| `registries`  | list(object) | [] | Defines which registries to import to |

### import object

| Key         | Type   | Default | Description                                         |
|-------------|--------|---------|-----------------------------------------------------|
| `enabled`   | bool   | false   | Enable import of charts and artifacts to registries |
| `copacetic` | object | nil     | Configuration parameters for Copacetic              |
| `cosign`    | object | nil     | Configuration parameters for Cosign                 |

### charts object

| Key              | Type   | Default | Description                                         |
|------------------|--------|---------|-----------------------------------------------------|
| `name`           | string | ""      | Chart name                                          |
| `version`        | string | ""      | Semver version of chart                             |
| `valuesFilePath` | string | ""      | Path to custom values.yaml to customize importing   |
| `repo`           | object | nil     | Define repository according to Helm Repository Spec |

#### repo object

| Key                        | Type   | Default | Description                                        |
|----------------------------|--------|---------|----------------------------------------------------|
| `name`                     | string |         | Name of the repository                             |
| `url`                      | string |         | URL to the repository                              |
| `username`                 | string | ""      | Username to repository for Basic Auth              |
| `password`                 | string | ""      | Password to Username for Basic Auth                |
| `certFile`                 | string | ""      | Path to certificate file for Certificate Auth      |
| `keyFile`                  | string | ""      | Path to key file for Key Auth                      |
| `caFile`                   | string | ""      | Path to custom certificate authority               |
| `insecure_skip_tls_verify` | bool   | false   | Skip TLS verify / Disable SSL                      |
| `pass_credentials_all`     | bool   | false   | Pass credentials to dependency charts repositories |

### registries object

| Key         | Type   | Default | Description                         |
|-------------|--------|---------|-------------------------------------|
| `name`      | string | ""      | Name of registry                    |
| `url`       | string | ""      | URL to registry                     |
| `insecure`  | bool   | false   | Disable SSL certificate validation  |
| `plainHTTP` | bool   | false   | Enable use of HTTP instead of HTTPS |
