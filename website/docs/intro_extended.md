---
sidebar_position: 2
---

# Intro Extended

Let's try all features of **Helmper in less than 5 minutes**.

In this tutorial demonstrates the full functionality of Helmper, from identifying images 
in the Helm Chart to patching and signing the images.

## Getting Started

Get started by **setting up local services**. These services are required for scanning and patching the images.
Then proceed by **creating the local filesystem structure**, populate one of the folders by **generating keys for cosign**. 
Finally **change the configuration** to included the newly created resources.

### Start local services

#### Registry

```shell title="bash"
docker run -d -p 5000:5000 --restart=always --name registry registry:2
```

#### Buildkitd

```shell title="bash"
export BUILDKIT_VERSION=v0.12.4
export BUILDKIT_PORT=8888
docker run --detach --rm --privileged \
-p 127.0.0.1:$BUILDKIT_PORT:$BUILDKIT_PORT/tcp \
--name buildkitd --entrypoint buildkitd "moby/buildkit:$BUILDKIT_VERSION" --addr tcp://0.0.0.0:$BUILDKIT_PORT
```

#### Trivy

```shell title="bash"
docker run -d -p 8887:8887 --name trivy aquasec/trivy:0.50.4 server --listen=0.0.0.0:8887
```

### Create output folders

```shell title="bash"
mkdir -p $HOME/.config/helmper/out/tars
mkdir -p $HOME/.config/helmper/out/reports
mkdir -p $HOME/.config/helmper/in
```

### Setup cosign keys

```shell title="bash"
docker run -it --name cosign bitnami/cosign generate-key-pair 
docker cp cosign:/cosign-keys $HOME/.config/helmper/in/cosign-keys
```

### Configuration

Change the configuration file

:::tip

Remember to change the user

:::

```yaml title="$HOME/.config/helmper/helmper.yaml"
k8s_version: 1.27.9
charts:
- name: prometheus
  version: 25.8.0
  plainHTTP: false
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
        folder: /home/<YOUR_USER>/.config/helmper/out/tars
        clean: true
      reports:
        folder: /home/<YOUR_USER>/.config/helmper/out/reports
        clean: true
  cosign:
    enabled: true
    keyRef: /home/<YOUR_USER>/.config/helmper/in/cosign-keys/cosign.key
    KeyRefPass: ""
    allowInsecure: true
    allowHTTPRegistry: true
```

## Run Helmper

```shell title="Run Helmper"
helmper
```

<p align="center"><img src="https://github.com/ChristofferNissen/helmper/blob/main/docs/gifs/full.gif?raw=true"/></p>
