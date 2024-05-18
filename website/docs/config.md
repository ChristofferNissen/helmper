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

## Example configuration

```yaml title="Example config"
k8s_version: 1.27.9
verbose: true
update: false
all: false
import:
  enabled: true
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
    valuesFilePath: /workspace/.in/values/kyverno/values.yaml
    repo:
      name: kyverno
      url: https://kyverno.github.io/kyverno/
  - name: keda
    version: 2.11.2
    repo:
      name: kedacore
      url: https://kedacore.github.io/charts/
  - name: argo-cd
    version: 5.51.4
    repo:
      name: argo
      url: https://argoproj.github.io/argo-helm/
  - name: prometheus
    version: 25.8.0
    valuesFilePath: /workspace/.in/values/prometheus/values.yaml
    repo:
      name: prometheus-community
      url: https://prometheus-community.github.io/helm-charts/
registries:
  - name: registry
    url: 0.0.0.0:5000
    insecure: true
    plainHTTP: true
```

## Configuration options

| Key | Type  | Default | Required | Description |
|-|-|-|-|-|
| `k8s_version` | string       | "1.27.9" | false | Some charts use images eliciting their tag based on the kube-apiserver version. Therefore, tell Helmper which version you run to import the correct version. |
| `verbose`     | bool         | false    |  false | Toggle verbose output |
| `update`      | bool         | false    |  false | Toggle update to latest chart version for each specified chart in `charts` |
| `all`         | bool         | false    |  false | Toggle import of all images regardless if they exist in the registries defined in `registries` |
| `import`      | object       | nil      | false |  If import is enabled, images will be pushed to the defined registries. If copacetic is enabled, images will be patched if possible. Finally, in the import section Cosign can be configured to sign the images after pushing to the registries. See table blow for full configuration options. |
| `import.enabled`   | bool   | false   | false | Enable import of charts and artifacts to registries |
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
| `import.cosign.allowInsecure`     | bool   | false   | false | Disable TLS verification    |
| `import.cosign.allowHTTPRegistry` | bool   | false   | false | Allow HTTP instead of HTTPS |
| `charts`      | list(object) | | true | Defines which charts to target |
| `charts[].name`           | string |         | true | Chart name                                          |
| `charts[].version`        | string |         | true | Semver version of chart                             |
| `charts[].valuesFilePath` | string | ""      | false | Path to custom values.yaml to customize importing   |
| `charts[].repo.name`                     | string |         | true | Name of the repository                             |
| `charts[].repo.url`                      | string |         | true | URL to the repository                              |
| `charts[].repo.username`                 | string | ""      | false | Username to repository for Basic Auth              |
| `charts[].repo.password`                 | string | ""      | false | Password to Username for Basic Auth                |
| `charts[].repo.certFile`                 | string | ""      | false | Path to certificate file for Certificate Auth      |
| `charts[].repo.keyFile`                  | string | ""      | false | Path to key file for Key Auth                      |
| `charts[].repo.caFile`                   | string | ""      | false | Path to custom certificate authority               |
| `charts[].repo.insecure_skip_tls_verify` | bool   | false   | false | Skip TLS verify / Disable SSL                      |
| `charts[].repo.pass_credentials_all`     | bool   | false   | false | Pass credentials to dependency charts repositories |
| `registries`  | list(object) | [] | false | Defines which registries to import to |
| `registries[].name`      | string |         | true | Name of registry                    |
| `registries[].url`       | string |         | true | URL to registry                     |
| `registries[].insecure`  | bool   | false   | false | Disable SSL certificate validation  |
| `registries[].plainHTTP` | bool   | false   | false | Enable use of HTTP instead of HTTPS |

## Buildkit

### addr

Here are the supported formats for `import.copacetic.buildkit.addr` configuration option:

* `unix:///path/to/buildkit.sock` - Connect to buildkit over unix socket.
* `tcp://$BUILDKIT_ADDR:$PORT` - Connect to buildkit over TCP. (not recommended for security reasons)
* `docker://<docker connection spec>` - Connect to docker, currently only unix sockets are supported, e.g. `docker://unix:///var/run/docker.sock` (or just `docker://`).
* `docker-container://my-buildkit-container` - Connect to a buildkitd running in a docker container.
* `buildx://my-builder` - Connect to a buildx builder (or `buildx://` for the currently selected builder). *Note: only container-backed buildx instances are currently supported*
* `nerdctl-container://my-container-name` - Similar to `docker-container` but uses `nerdctl`.
* `podman-container://my-container-name` - Similar to `docker-container` but uses `podman`.
* `ssh://myhost` - Connect to a buildkit instance over SSH. Format of the host spec should mimic the SSH command.
* `kubepod://mypod` - Connect to buildkit running in a Kubernetes pod. Can also specify kubectl context and pod namespace (`kubepod://mypod?context=foo&namespace=notdefault`).

See more details in the [Copacetic Documentation](https://project-copacetic.github.io/copacetic/website/custom-address)

### mTLS

Helmper supports setting required configuration options for enabling mTLS with an expose Buildkit instance over TCP, althrough the following configuration options:

* `import.copacetic.buildkitd.CACertPath`
* `import.copacetic.buildkitd.certPath`
* `import.copacetic.buildkitd.keyPath`

Read more in the official docs by [mobdy/buildkit](https://github.com/moby/buildkit?tab=readme-ov-file#expose-buildkit-as-a-tcp-service).

## Cosign

### keyRef

keyRef as support for local files, through remote protocols `<some provider>://<some key>` or environment variables `env://[ENV_VAR]`.
Read more about all options in the [Cosign Docs](https://docs.sigstore.dev/signing/signing_with_containers/#sign-with-a-key-pair-stored-elsewhere).

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
