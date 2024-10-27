---
sidebar_position: 11
---

# OCI

Helmper primarily focus on OCI registry compatability, while also supporting Helm Repositories as sources, currently only OCI targets are supported.

## Use OCI registry as source

```yaml
...
charts:
- name: kyverno
  version: "3.2.*"
  plainHTTP: true
  repo:
    url: oci://0.0.0.0:5000/charts/kyverno
...
```

*or* use a public provider:

```yaml
charts:
- name: cert-manager
  version: "1.0.0"
  repo:
    url: "oci://chartproxy.container-registry.com/charts.jetstack.io/cert-manager"
```

## Use OCI registry as destination

Locally you can do

```yaml
...
registries:
- name: registry
  url: oci://0.0.0.0:5001
  insecure: true
  plainHTTP: true
...
```

*or* use a cloud provider:

<!-- TODO: Implement this feature. Optionally specify the part inside the registry you wish to store the OCI Artifacts -->

```yaml
...
registries:
- name: registry
  url: oci://your_registry.azurecr.io
...
```

Helm Charts will be placed under `charts/{chart_name}`, and images will be placed directly in the regsitry root `/`.

Optionally, if you specify `registry[].sourcePrefix=true`, images will be placed under a path obtained from the source registry name, fx

`docker.io/library/hello-world -> oci://reg.azurecr.io/docker/library/hello-world`

```yaml
...
registries:
- name: registry
  url: oci://your_registry.azurecr.io
  sourcePrefix: true
...
```
