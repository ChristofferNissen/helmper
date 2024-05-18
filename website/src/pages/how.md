---
title: How
---

# How

`helmper` is first of all a Helm Chart Analyser, built for the purpose of addressing a short comming of the metadata attributes in a Helm Chart - missing list of required images needed to deploy the Helm Chart. This is the **core** part of Helmper, and the only part of the functionality that is custom to `helmper`. For the remaining functionality helmper is *standing on the shoulders of giants* to provide additional capabilities right within `helmper`.

`helmper` is utlizing the following projects:
* [Helm](<https://github.com/helm/helm>) for Helm operations
* [Oras](<https://github.com/oras-project/oras-go>) for OCI registry interactions 
* [Trivy](https://github.com/aquasecurity/trivy) for vulnerability scanning
* [Copacetic](https://github.com/project-copacetic/copacetic) for vulnerability patching
    * [Buildkitd](https://github.com/moby/buildkit) container image modification as part of Copacetic
* [Cosign](https://github.com/sigstore/cosign) for container image signing

`helmper` connects via gRPC to Trivy and Buildkit so you can run `helmper` without root privileges whereever you want - as binary or as container in Kubernetes.

## Core

The diagram below demonstrates the core functionality of Helmper - analysing Helm Charts and importing the images into OCI-compliant registries.

![An image from the static](/img/core.svg)

1) Pull Helm Chart(s) from remote registries
2) Analyse charts for image references
3) Check status of images in registries
4) Distribute across registries

## Extended

The diagram below demonstrates the extended functionality of Helmper - extending the core with os level vulnerability scanning, vulnerability patching and signing.

![An image from the static](/img/extended.svg)

1) Pull Helm Chart(s)
2) Analyse charts for image references
3) Check status of images in registries
4) Pre-patch Scan images with Trivy
5) Patch images with Copacetic
6) Post-patch Scan images with Trivy
7) Push images with `oras-go`
8) Sign images with Cosign
