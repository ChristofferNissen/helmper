<p align="center">
  <a href="https://github.com/ChristofferNissen/helmper">
    <img src="docs/logo.png" alt="Helmper logo">
  </a>

  <p align="center">
    A little helper that reads Helm Charts and pushes the images to your registries.
    <br>
    <a href="https://github.com/ChristofferNissen/helmper/issues/new?template=bug.md">Report bug</a>
    ·
    <a href="https://github.com/ChristofferNissen/helmper/issues/new">Request feature</a>
    ·
    <a href="https://github.com/ChristofferNissen/helmper/releases">Releases</a>
    ·
    <a href="https://github.com/ChristofferNissen/helmper/releases/latest">Latest release</a>
  </p>

[![Go Report Card](https://goreportcard.com/badge/github.com/ChristofferNissen/helmper)](https://goreportcard.com/report/github.com/ChristofferNissen/helmper) 
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/ChristofferNissen/helmper/blob/main/LICENSE)

</p>

## What is Helmper?

_DISCLAIMER: helmper is in beta, so stuff may change._

`helmper` is a go program that reads Helm Charts from remote OCI registries and pushes the charts container images to your registries.

`helmper` is built with [Helm](<https://github.com/helm/helm>), [Oras](<https://github.com/oras-project/oras-go>), [Trivy](https://github.com/aquasecurity/trivy), [Copacetic](https://github.com/project-copacetic/copacetic) ([Buildkitd](https://github.com/moby/buildkitd)) and [Cosign](https://github.com/sigstore/cosign).

Helmper connects via gRPC to Trivy and Buildkitd so you can run `helmper` without root privileges whereever you want. 

`helmper` demonstrates exceptional proficiency in operating within controlled environments that might require Change Management and/or air-gapped networks. This expertise is particularly beneficial in industries subject to stringent regulations, such as Medical and Banking. This is due to `helmper` ensures binary reproducibility of Helm Charts by storing all necessary artifacts in your registries.

### how?

#### Simple

Simply tell `helmper` which charts to analyze and registries to use by creating a `helmper.yaml` file and run helmper from the same folder.

```yaml
k8s_version: 1.27.9
import:
  enabled: true
charts:
- name: prometheus
  version: 25.8.0
  valuesFile: /workspace/in/values/prometheus/values.yaml # (Optional)
  repo:
    name: prometheus-community
    url: https://prometheus-community.github.io/helm-charts/
registries:
- name: registry
  url: 0.0.0.0:5000
  insecure: true
  plainHTTP: true
```

Helmper will then import the charts, the charts listed as dependencies including all images specified through the Helm `values.yaml` file.

<p align="center"><img src="docs/gifs/simple.gif?raw=true"/></p>

**Note** Authentication

Helmper utilizes well known configuration options to interact with registries. 

When using the Helm SDK, Helmper will utilize the file defined by `HELM_REGISTRY_CONFIG` for picking up authentication credentials for registries.

When Helmper is using Oras for interacting with OCI artifacts, Oras utilizes the [Docker credentials helper](https://pkg.go.dev/oras.land/oras-go/v2@v2.5.0/registry/remote/credentials), which will look in the system keychain, `$DOCKER_CONFIG/config.json` (if set) or `$HOME/.docker/config.json` file for picking up authentication credentials for all registries.

If your registries requires authentication, simply login with the services own login command.

fx for Docker:

```bash
docker login -u user -p pass
```

Azure:

```bash
az acr login -n myregistry
```

#### Full

In this example Helmper will also scan with Trivy, patch with Copacetic and sign with Cosign all identified images before pushing with Oras to all registries.

```yaml
k8s_version: 1.27.9
charts:
- name: prometheus
  version: 25.8.0
  valuesFile: /workspace/in/values/prometheus/values.yaml # (Optional)
  repo:
    name: prometheus-community
    url: https://prometheus-community.github.io/helm-charts/
registries:
- name: registry # `Helmper` picks up authentication from the environment automatically.
  url: 0.0.0.0:5000
  insecure: true
  plainHTTP: true
import:
  enabled: true
  copacetic:
    enabled: true
    ignoreErrors: true
    buildkitd:
      addr: tcp://0.0.0.0:8888
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
```

<p align="center"><img src="docs/gifs/full.gif?raw=true"/></p>

### Flow diagram

To understand how the different configuration options works, please study the flow diagram below

<p align="center"><img style="width: 400px;" src="docs/imgs/diagrams/flow.svg?raw=true"/></p>

## Install

Simply pick the binary for your platform from the Release section on GitHub.

## Scope

### In scope

* Helmper operates with OCI compliant artifacts and OCI compliant registries.
* Helmper must remain without dependency on a container runtime daemon to work in containers without root privileges.

### Out of scope

* Helmper does not work with other Kubernetes package formats
* Helmper authenticates with registries with the docker config. There, Helmper will not have any propreitary libraries to facilitate authentication for any cloud providers. Simply use `docker login` or equivalent before running Helmper, and you should be authenticated for 3 hours for each registry.

## Roadmap

* Operator Framework to enable using Helmper with GitOps in management clusters
* Add option to import to registries via pipeline for compliance audit trail retention
* SBOM
* OpenTelemetry

---

## Code of Conduct

This project has adopted the [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/master/code-of-conduct.md). See [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) for further details.

## Credits

The gopher's logo of Helmper is licensed under the Creative Commons 3.0 Attributions license.

The original Go gopher was designed by Renee French.
