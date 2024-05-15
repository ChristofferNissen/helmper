---
title: What
---

<!-- <img align="right" width="30%" src="/static/img/helmper.svg" alt="Helmper logo"/> -->

`helmper` is a go program that reads Helm Charts from remote OCI registries and pushes the charts container images to your registries with optional OS level vulnerability patching.

`helmper` provides an interface to reduce the maintenance burden associated with managing a large collection of Helm Charts by:

- automatically detecting all enabled container images in charts
- providing an easy way to stay up to date on new chart releases
- providing option to only import new images, or all images
- enabling quick patching(and re-patching) of all images
- enabling signing of images was an integrated part of the process
- providing a mechanism to check requirements/dependencies before deploying charts with fx GitOps

`helmper` is built with [Helm](<https://github.com/helm/helm>), [Oras](<https://github.com/oras-project/oras-go>), [Trivy](https://github.com/aquasecurity/trivy), [Copacetic](https://github.com/project-copacetic/copacetic) ([Buildkitd](https://github.com/moby/buildkit)) and [Cosign](https://github.com/sigstore/cosign).


Helmper connects via gRPC to Trivy and Buildkit so you can run `helmper` without root privileges whereever you want. 
