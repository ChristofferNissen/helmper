---
sidebar_label: 'Configuration Options'
sidebar_position: 4
---

# Configuration

## Location

The configuration file `helmper.yaml` can be placed in:

- Current directory (`.`)
- `$HOME/.config/helmper/`
- `/etc/helmper/`

### Override configuration location with `--f` flag

Helmper supports a single flag `--f` to specify the configuration file. When using the flag it takes precedence over the default location and name of the configuration file. The configuration file `--f` can be any format (JSON, TOML, YAML, HCL, envfile and Java properties config files, see [viper](https://github.com/spf13/viper?tab=readme-ov-file#what-is-viper)).

## Example configuration

```yaml title="Example config"
k8s_version: 1.31.1
verbose: true
update: false
all: false
parser:
  useCustomValues: false
import:
  enabled: true
  architecture: "linux/amd64"
  copacetic:
    enabled: true
    ignoreErrors: true
    buildkitd:
      addr: tcp://0.0.0.0:8888
      CACertPath: ""
      certPath: ""
      keyPath: ""
    trivy:
      addr: http://0.0.0.0:8887
      insecure: true
      ignoreUnfixed: true
    output:
      tars:
        folder: /workspace/.out/tars
        clean: true
      reports:
        folder: /workspace/.out/reports
        clean: true
  cosign:
    enabled: true
    keyRef: /workspace/.devcontainer/cosign.key
    KeyRefPass: ""
    allowInsecure: true
    allowHTTPRegistry: true
charts:
  - name: loki
    version: 5.38.0
    valuesFilePath: /workspace/.in/values/loki/values.yaml
    repo:
      name: grafana
      url: https://grafana.github.io/helm-charts/
      username: ""
      password: ""
      certFile: ""
      keyFile: ""
      caFile: ""
      insecure_skip_tls_verify: false
      pass_credentials_all: false
  - name: kyverno
    version: 3.1.1
    values:
      namespaceOverride: system
      registry: customer-registry.io/
    repo:
      name: kyverno
      url: https://kyverno.github.io/kyverno/
  - name: keda
    version: 2.11.2
    repo:
      name: kedacore
      url: https://kedacore.github.io/charts/
  - name: argo-cd
    version: ">5.0.0 <7.0.0"
    repo:
      name: argo
      url: https://argoproj.github.io/argo-helm/
  - name: cilium
    version: "1.15.6"
    repo:
      name: cilium
      url: https://helm.cilium.io/
    images:
      exclude:
      - ref: quay.io/cilium/cilium-envoy
      excludeCopacetic:
      - ref: quay.io/cilium/startup-script
      modify:
      - fromValuePath: operator.image.repository
        to: quay.io/cilium/operator-generic
  - name: prometheus
    version: 25.8.0
    valuesFilePath: /workspace/.in/values/prometheus/values.yaml
    repo:
      name: prometheus-community
      url: https://prometheus-community.github.io/helm-charts/
images:
- ref: docker.io/library/busybox:latest@sha256:7cc4b5aefd1d0cadf8d97d4350462ba51c694ebca145b08d7d41b41acc8db5aa
  patch: false
registries:
  - name: registry
    url: 0.0.0.0:5000
    insecure: true
    plainHTTP: true
```

## Configuration options

| Key | Type  | Default | Required | Description |
|-|-|-|-|-|
| `k8s_version` | string       | "1.31.1" | false | Some charts use images eliciting their tag based on the kube-apiserver version. Therefore, tell Helmper which version you run to import the correct version. |
| `verbose`     | bool         | false    |  false | Toggle verbose output |
| `update`      | bool         | false    |  false | Toggle update to latest chart version for each specified chart in `charts` |
| `all`         | bool         | false    |  false | Toggle import of all images regardless if they exist in the registries defined in `registries` |
| `parser`                          | object       | nil    |  false | Adjust how Helmper parses charts |
| `parser.disableImageDetection`    | bool         | false  |  false | Disable Image detection |
| `parser.useCustomValues`          | bool         | false  |  false | Use user defined values for image parsing |
| `import`      | object       | nil      | false |  If import is enabled, images will be pushed to the defined registries. If copacetic is enabled, images will be patched if possible. Finally, in the import section Cosign can be configured to sign the images after pushing to the registries. See table blow for full configuration options. |
| `import.enabled`   | bool   | false   | false | Enable import of charts and artifacts to registries |
| `import.replaceRegistryReferences`   | bool   | false   | false | Replace occurrences of old registry with import target registry |
| `import.architecture`   | *string   | nil   | false | Specify desired container image architecture |
| `import.copacetic.enabled`      | bool   | false   |  false | Enable Copacetic                            |
| `import.copacetic.ignoreErrors` | bool   | true    |  false | Ignore errors during Copacetic patching     |
| `import.copacetic.buildkitd.addr`       | string |         | true | Address to Buildkit                                   |
| `import.copacetic.buildkitd.CACertPath` | string | ""      | false | Path to certificate authority used for authentication |
| `import.copacetic.buildkitd.certPath`   | string | ""      | false | Path to certificate used for authentication           |
| `import.copacetic.buildkitd.keyPath`    | string | ""      | false | Path to key used for authentication                   |
| `import.copacetic.trivy.addr`          | string |         | true | Address to Trivy               |
| `import.copacetic.trivy.insecure`      | bool   | false   | false | Disable TLS verification       |
| `import.copacetic.trivy.ignoreUnfixed` | bool   | false   | false | Ignore unfixed vulnerabilities |
| `import.copacetic.output.tars.folder` | string |         | true | Path to output folder                  |
| `import.copacetic.output.tars.clean`  | bool   | true    | false | Remove artifacts after running Helmper |
| `import.copacetic.output.reports.folder` | string |         | true | Path to output folder                  |
| `import.copacetic.output.reports.clean`  | bool   | true    | false | Remove artifacts after running Helmper |
| `import.cosign.enabled`           | bool   | false   | false | Enables signing with Cosign |
| `import.cosign.keyRef`            | string |         | true | Path to Cosign private key  |
| `import.cosign.keyRefPass`        | string |         | true | Cosign private key password |
| `import.cosign.pubKeyRef`         | string |         | false | Path to Cosign public key  |
| `import.cosign.allowInsecure`     | bool   | false   | false | Disable TLS verification    |
| `import.cosign.allowHTTPRegistry` | bool   | false   | false | Allow HTTP instead of HTTPS |
| `charts`      | list(object) | [] | false | Defines which charts to target |
| `charts[].name`           | string |         | true | Chart name                                          |
| `charts[].version`        | string |         | true | Desired version of chart. Supports semver literal or semver ranges (semantic version spec 2.0) |
| `charts[].plainHTTP`        | bool | false   | false | Use HTTP instead of HTTPS for repository protocol |
| `charts[].valuesFilePath` | string | ""      | false | Path to custom values.yaml to customize importing   |
| `charts[].values` | object | nil      | false | Inline values to customize importing (cannot be used together with `charts[].valuesFilePath`)   |
| `charts[].images`                         | object        | nil    | false | Customization options for images in chart  |
| `charts[].images.exclude`                 | list(object)  | []     | false | Defines which images to exclude from processing |
| `charts[].images.exclude[].ref`           | string        | ""     | false | Container Image reference |
| `charts[].images.excludeCopacetic`        | list(object)  | []     | false | Defines which images to exclude from copacetic patching if copa is enabled |
| `charts[].images.excludeCopacetic[].ref`  | string        | ""     | false | Container Image reference |
| `charts[].images.modify`                  | list(object)  | []     | false | Defines which image references to modify before import |
| `charts[].images.modify[].from`           | string        | ""     | false | Defines which image reference should be replaced with `to` |
| `charts[].images.modify[].fromValuesPath` | string        | ""     | false | Defines which path in the charts default Helm Values to override with `to`|
| `charts[].images.modify[].to`             | string  Name of the repository      | ""     | false | Defines new value to be inserted |
| `charts[].repo`                          | object |         | true  | Helm Repository spec                             |
| `charts[].repo.name`                     | string |         | true  | Name of the repository                             |
| `charts[].repo.url`                      | string |         | true  | URL to the repository                              |
| `charts[].repo.username`                 | string | ""      | false | Username to repository for Basic Auth              |
| `charts[].repo.password`                 | string | ""      | false | Password to Username for Basic Auth                |
| `charts[].repo.certFile`                 | string | ""      | false | Path to certificate file for Certificate Auth      |
| `charts[].repo.keyFile`                  | string | ""      | false | Path to key file for Key Auth                      |
| `charts[].repo.caFile`                   | string | ""      | false | Path to custom certificate authority               |
| `charts[].repo.insecure_skip_tls_verify` | bool   | false   | false | Skip TLS verify / Disable SSL                      |
| `charts[].repo.pass_credentials_all`     | bool   | false   | false | Pass credentials to dependency charts repositories |
| `images`     | list(object)   | [] | false | Additional container images to include in import |
| `images.ref` | string  | | true | Container image reference |
| `images.patch` | *bool  | nil | false | Define if container image should be patched with Trivy/Copacetic |
| `registries`  | list(object) | [] | false | Defines which registries to import to |
| `registries[].name`      | string |         | true | Name of registry                    |
| `registries[].url`       | string |         | true | URL to registry                     |
| `registries[].insecure`  | bool   | false   | false | Disable SSL certificate validation  |
| `registries[].plainHTTP` | bool   | false   | false | Enable use of HTTP instead of HTTPS |
| `registries[].sourcePrefix` | bool   | false   | false | Append source registry name to source image repository |
| `mirrors` | list(object)   | []   | false | Enable use of registry mirrors |
| `mirrors.registry` | string   | "" | true | Registry to configure mirror for fx docker.io |
| `mirrors.mirror` | string   | "" | true | Registry Mirror URL |

## Charts

The `charts` configuration option defines which charts to import.

| Key | Type  | Default | Required | Description |
|-|-|-|-|-|
| `charts[].name`                           | string        |        | true  | Chart name                                          |
| `charts[].version`                        | string        |        | true  | Desired version of chart. Supports semver literal or semver ranges (semantic version spec 2.0)   |
| `charts[].valuesFilePath`                 | string        | ""     | false | Path to custom values.yaml to customize importing   |
| `charts[].values`                         | object        | nil    | false | Inline values to customize importing (cannot be used together with `charts[].valuesFilePath`)   |
| `charts[].images`                         | object        | nil    | false | Customization options for images in chart  |
| `charts[].images.exclude`                 | list(object)  | []     | false | Defines which images to exclude from processing |
| `charts[].images.exclude.ref`             | string        | ""     | false | Container Image reference |
| `charts[].images.excludeCopacetic`        | list(object)  | []     | false | Defines which images to exclude from copacetic patching if copa is enabled |
| `charts[].images.excludeCopacetic.ref`    | string        | ""     | false | Container Image reference |
| `charts[].images.modify`                  | list(object)  | []     | false | Defines which image references to modify before import |
| `charts[].images.modify[].from`           | string        | ""     | false | Defines which image reference should be replaced with `to` |
| `charts[].images.modify[].fromValuesPath` | string        | ""     | false | Defines which path in the charts default Helm Values to override with `to`|
| `charts[].images.modify[].to`             | string        | ""     | false | Defines new value to be inserted |

The `version` supports [Semantic Versioning 2.0.0](https://semver.org/) format versions as [Helm](https://helm.sh/docs/chart_best_practices/conventions/#version-numbers).

[Semver cheatsheet](https://devhints.io/semver)

### Chart sources

**Helm Repository**

Helmper supports all configuration options for Helm Repositories available in the [Helm CLI](https://helm.sh/docs/helm/helm_repo/) `helm repo add`:

| Key | Type  | Default | Required | Description |
|-|-|-|-|-|
| `charts[].repo.name`                     | string |         | true | Name of the repository                             |
| `charts[].repo.url`                      | string |         | true | URL to the repository                              |
| `charts[].repo.username`                 | string | ""      | false | Username to repository for Basic Auth              |
| `charts[].repo.password`                 | string | ""      | false | Password to Username for Basic Auth                |
| `charts[].repo.certFile`                 | string | ""      | false | Path to certificate file for Certificate Auth      |
| `charts[].repo.keyFile`                  | string | ""      | false | Path to key file for Key Auth                      |
| `charts[].repo.caFile`                   | string | ""      | false | Path to custom certificate authority               |
| `charts[].repo.insecure_skip_tls_verify` | bool   | false   | false | Skip TLS verify / Disable SSL                      |
| `charts[].repo.pass_credentials_all`     | bool   | false   | false | Pass credentials to dependency charts repositories |

**OCI Registry**

Not implemented yet. Coming soon.

## Images

Helmper provides the option to include additional images in the import flow not extracted from one of the defined Helm Charts.
Simply define the additional images in the `images` configuration option.

## Buildkit

### addr

Here are the supported formats for `import.copacetic.buildkit.addr` configuration option:

- `unix:///path/to/buildkit.sock` - Connect to buildkit over unix socket.
- `tcp://$BUILDKIT_ADDR:$PORT` - Connect to buildkit over TCP. (not recommended for security reasons)
- `docker://<docker connection spec>` - Connect to docker, currently only unix sockets are supported, e.g. `docker://unix:///var/run/docker.sock` (or just `docker://`).
- `docker-container://my-buildkit-container` - Connect to a buildkitd running in a docker container.
- `buildx://my-builder` - Connect to a buildx builder (or `buildx://` for the currently selected builder). *Note: only container-backed buildx instances are currently supported*
- `nerdctl-container://my-container-name` - Similar to `docker-container` but uses `nerdctl`.
- `podman-container://my-container-name` - Similar to `docker-container` but uses `podman`.
- `ssh://myhost` - Connect to a buildkit instance over SSH. Format of the host spec should mimic the SSH command.
- `kubepod://mypod` - Connect to buildkit running in a Kubernetes pod. Can also specify kubectl context and pod namespace (`kubepod://mypod?context=foo&namespace=notdefault`).

See more details in the [Copacetic Documentation](https://project-copacetic.github.io/copacetic/website/custom-address)

### mTLS

Helmper supports setting required configuration options for enabling mTLS with an expose Buildkit instance over TCP, although the following configuration options:

- `import.copacetic.buildkitd.CACertPath`
- `import.copacetic.buildkitd.certPath`
- `import.copacetic.buildkitd.keyPath`

Read more in the official docs by [moby/buildkit](https://github.com/moby/buildkit?tab=readme-ov-file#expose-buildkit-as-a-tcp-service).

## Cosign

### keyRef

`keyRef` as support for local files, through remote protocols `<some provider>://<some key>` or environment variables `env://[ENV_VAR]`.
Read more about all options in the [Cosign Docs](https://docs.sigstore.dev/signing/signing_with_containers/#sign-with-a-key-pair-stored-elsewhere).

### pubKeyRef

`pubKeyRef` defines the path to the public key used to verify chart and image signatures.
`pubKeyRef` can be omitted when using remote protocol for keyRef as remote KMS protocols usually works with key-pairs.
If `keyRef` is a path to a local file, and `pubKeyRef` is not define, pubKeyRef will be set to the same path as `keyRef`, with `.pub` instead of `.key`, fx `/home/you/keypair/cosign.key` becomes `/home/you/keypair/cosign.pub`.

#### local

```text title="local file"
cosign.key
```

#### remote

##### Kubernetes Secret

```text title="Kubernetes Secret"
k8s://[NAMESPACE]/[KEY]
```

##### Azure Key Vault

```text title="Azure Key vault"
azurekms://[VAULT_NAME][VAULT_URI]/[KEY]
```

### keyRefPass

Helmper supports specifying the password directly in the helmper.yaml as `keyRefPass`. Alternatively you can use the `COSIGN_PASSWORD` environment variable to specify the password.

If you use any of the remote options for `keyRef` you can leave the keyRefPass unspecified.
