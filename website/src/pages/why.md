---
title: Why
---

# Why

`helmper` demonstrates exceptional proficiency in operating within controlled environments that might require Change Management and/or air-gapped networks. This expertise is particularly beneficial in industries subject to stringent regulations, such as Medical and Banking. 

## Security

`helmper` aims to provide a sane opt-in security process by scanning images for vulnerabilities and patching fixable OS level vulnerabilities in container images retrieved from public sources, before distribution to your registries.

## Operations

`helmper` aims to ensure binary reproducibility of Helm Charts by storing all necessary artifacts in your registries, reducing risk associated with disappearing upstream dependencies.


`helmper` aims to reduce maintenance efforts of onboarding new Helm Chart versions to use within an regulated organization by providing a robust engine for extracting images from charts, patching and distributing images in a fast, reliable and repeatable way. 


*In most industries it might be acceptable to use registries as pull-through caches that will automatically find missing images and store them in your registries. For regulated industries this presents a challenge, as it is not worth the risk to start a deployment to production and wait for resources to be fetched.*

