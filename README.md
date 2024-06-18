<p align="center">
  <a href="https://github.com/ChristofferNissen/helmper">
    <img src="docs/logo/helmper_banner.png" alt="Helmper logo">
  </a>

  <p align="center">
    A little helper that pushes Helm Charts and images to your registries, easily configured with a declarative spec.
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

<img align="right" width="20%" src="docs/logo/helmper.svg" alt="Helmper logo">

`helmper` is a go program that reads Helm Charts from remote OCI registries and pushes the charts container images to your registries with optional OS level vulnerability patching.

`helmper` is built with [Helm](<https://github.com/helm/helm>), [Oras](<https://github.com/oras-project/oras-go>), [Trivy](https://github.com/aquasecurity/trivy), [Copacetic](https://github.com/project-copacetic/copacetic) ([Buildkit](https://github.com/moby/buildkitd)) and [Cosign](https://github.com/sigstore/cosign).


Helmper connects via gRPC to Trivy and Buildkit so you can run `helmper` without root privileges wherever you want. 

`helmper` demonstrates exceptional proficiency in operating within controlled environments that might require Change Management and/or air-gapped networks. This expertise is particularly beneficial in industries subject to stringent regulations, such as Medical and Banking. `helmper` aims to ensure binary reproducibility of Helm Charts by storing all necessary artifacts in your registries.

`helmper` provides an interface to reduce the maintenance burden associated with managing a large collection of Helm Charts by:

- automatically detecting all enabled container images in charts
- providing an easy way to stay up to date on new chart releases
- providing option to only import new images, or all images
- enabling quick patching(and re-patching) of all images
- enabling signing of images as an integrated part of the process
- providing a mechanism to check requirements/dependencies before deploying charts with fx GitOps

### how?

#### Core

Simply tell `helmper` which charts to analyze and registries to use by creating a `helmper.yaml` file and run helmper from the same folder.

```yaml
k8s_version: 1.27.9
import:
  enabled: true
charts:
- name: prometheus
  version: 25.8.0
  valuesFilePath: /workspace/in/values/prometheus/values.yaml # (Optional)
  repo:
    name: prometheus-community
    url: https://prometheus-community.github.io/helm-charts/
registries:
- name: registry
  url: 0.0.0.0:5000
  insecure: true
  plainHTTP: true
```

Helmper will import the charts, the charts listed as dependencies including all images specified through the Helm `values.yaml` file.

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

#### Extended

In this example Helmper will also scan with Trivy, patch with Copacetic and sign with Cosign all identified images before pushing with Oras to all registries.

```yaml
k8s_version: 1.27.9
charts:
- name: prometheus
  version: 25.8.0
  valuesFilePath: /workspace/in/values/prometheus/values.yaml # (Optional)
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

## Documentation

The full documentation for Helmper can be found at [christoffernissen.github.io/helmper](https://christoffernissen.github.io/helmper/).

## Compatibility

Helmper utilizes the Helm SDK to maintain full compatibility with both Helm Repositories and OCI registries for storing Helm Charts.

In practice, Helmper currently pushes charts and images to the same destination registry, so it must be OCI compliant. 

Helmper utilizes `oras-go` to push OCI artifacts. Helmper utilizes the Helm SDK to push Helm Charts, as the Helm SDK sets the correct metadata attributes.

Oras and Helm state support all registries with OCI support, for example:

- [CNCF Distribution](https://oras.land/docs/compatible_oci_registries#cncf-distribution) - local/offline verification
- [Amazon Elastic Container Registry](https://docs.aws.amazon.com/AmazonECR/latest/userguide/push-oci-artifact.html)  
- [Azure Container Registry](https://docs.microsoft.com/azure/container-registry/container-registry-helm-repos#push-chart-to-registry-as-oci-artifact)
- [Google Artifact Registry](https://cloud.google.com/artifact-registry/docs/helm/manage-charts)
- [Docker Hub](https://docs.docker.com/docker-hub/oci-artifacts/)
- [Harbor](https://goharbor.io/docs/main/administration/user-defined-oci-artifact/)
- [Zot Registry](https://zotregistry.dev/)
- [GitHub Packages container registry](https://oras.land/docs/compatible_oci_registries#github-packages-container-registry-ghcr)
- [IBM Cloud Container Registry](https://cloud.ibm.com/docs/Registry?topic=Registry-registry_helm_charts)
- [JFrog Artifactory](https://jfrog.com/help/r/jfrog-artifactory-documentation/helm-oci-repositories)

Sources: [Helm](https://helm.sh/docs/topics/registries/#use-hosted-registries) [Oras](https://oras.land/docs/compatible_oci_registries)

For testing, Helmper is using the [CNCF Distribution]() registry.

## Install

Simply pick the binary for your platform from the Release section on GitHub.

### Linux

```bash
VERSION=$(curl -Lso /dev/null -w %{url_effective} https://github.com/christoffernissen/helmper/releases/latest | grep -o '[^/]*$')
curl -LO https://github.com/christoffernissen/helmper/releases/download/$VERSION/helmper-linux-amd64
chmod +x helmper-linux-amd64
sudo mv helmper-linux-amd64 /usr/local/bin/helmper
```

### Mac OS

```bash
VERSION=$(curl -Lso /dev/null -w %{url_effective} https://github.com/christoffernissen/helmper/releases/latest | grep -o '[^/]*$')
curl -LO https://github.com/christoffernissen/helmper/releases/download/$VERSION/helmper-darwin-amd64
chmod +x helmper-darwin-amd64
sudo mv helmper-darwin-amd64 /usr/local/bin/helmper
```

### Windows

Extract the tar and launch the exe file.

## Scope

### In scope

* Helmper operates with OCI compliant artifacts and OCI compliant registries.
* Helmper must remain without dependency on a container runtime daemon to work in containers without root privileges.

### Out of scope

* Helmper does not work with other Kubernetes package formats
* Helmper authenticates with registries with the docker config. Therefore, Helmper will not have any proprietary libraries to facilitate authentication for any cloud providers. Simply use `docker login` or equivalent before running Helmper, and you should be authenticated for 3 hours for each registry.

## Roadmap

* Operator Framework to enable using Helmper with GitOps in management clusters
* Add option to import to registries via pipeline for compliance audit trail retention
* SBOM
* OpenTelemetry

## Code of Conduct

This project has adopted the [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/main/code-of-conduct.md). See [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) for further details.

## Credits

Helmper logo and banner have been kindly donated to the project by María Ruiz Garrido :heart:

The gopher's logo of Helmper is licensed under the Creative Commons 3.0 Attributions license.

The original Go gopher was designed by Renee French.
