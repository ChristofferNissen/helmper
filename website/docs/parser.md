---
sidebar_position: 10
---

# Extracting Images from Helm Charts

On this page you can read about the how `helmper` extracts container image references from Helm Charts `values.yaml` file.  

## Image Format

Helmper considers the following elements of an image reference string

1. registry
2. repository
3. name
4. tag
5. digest/sha

These values will be combined to one of the following strings:

```txt
registry/repository/name:tag
```

```txt
registry/repository/name@digest
```

The combined strings are parsed with [distribution/reference](https://github.com/distribution/reference) library to check image validity.

## Supported sections in `values.yaml`


```yaml
image: "reference:tag"
```

```yaml
image: "reference@digest"
```

```yaml
...
image:
    repository: "docker.io/library/hello-world"
    tag: "latest"
...
```

```yaml
...
image:
    registry: "docker.io"
    repository: "library/hello-world"
    tag: "latest"
...
```

```yaml
...
image:
    registry: "docker.io"
    repository: "library/hello-world"
    tag: "latest"
    digest: "sha256:266b191e926f65542fa8daaec01a192c4d292bff79426f47300a046e1bc576fd"
    useDigest: true
...
```

**note** *some charts use sha instead of digest. Helmper consider those two to both refer to the digest of the image*

### Ignored sections 

Helmper will ignore the following sections: 

```yaml
...
global:
    image:
        registry: ""
        repository: ""
        tag: ""
...
```

## Tested against following Helm Charts

- Prometheus
- Promtail
- Loki
- Mimir-Distributed
- Grafana
- Cilium
- Cert-Manager
- Ingress-Nginx
- Reflector
- Velero
- Kured
- Keda
- Trivy-Operator
- Kubescape-Operator
- ArgoCD

See more in the [test file](/internal/program_test.go)
