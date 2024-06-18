---
sidebar_position: 1
---

# Intro

Let's discover **Helmper in less than 5 minutes**.

## Getting Started

Get started by **installing helmper** and **creating a new configuration**.

### What you'll need

- Container Runtime for external services (Registries, Trivy, Buildkit)
  - [Podman](https://podman.io/)
  - [Docker](https://www.docker.com/)
    - Make sure to follow post-install steps to run without root

### Install Helmper

Simply download the latest version of Helmper from GitHub Releases

#### Linux

```shell title="bash"
VERSION=$(curl -Lso /dev/null -w %{url_effective} https://github.com/christoffernissen/helmper/releases/latest | grep -o '[^/]*$')
curl -LO https://github.com/christoffernissen/helmper/releases/download/$VERSION/helmper-linux-amd64
chmod +x helmper-linux-amd64
sudo mv helmper-linux-amd64 /usr/local/bin/helmper
```

### Configuration

Create the configuration file

```yaml title="$HOME/.config/helmper/helmper.yaml"
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

## Start local service

### Registry

```shell title="bash"
docker run -d -p 5000:5000 --restart=always --name registry registry:2
```

## Run Helmper

```shell title="Run Helmper"
helmper
```

<p align="center"><img src="https://github.com/ChristofferNissen/helmper/blob/main/docs/gifs/simple.gif?raw=true"/></p>
