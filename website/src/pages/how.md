---
title: How
---

# How

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
