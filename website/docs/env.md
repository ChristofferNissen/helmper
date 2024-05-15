---
sidebar_label: 'Development Environment'
sidebar_position: 6
---

# Development Environment

The project provides a devcontainer with a docker-compose.yml defining all required services.

### Docker

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
--name buildkitd \
--entrypoint buildkitd \ 
"moby/buildkit:$BUILDKIT_VERSION" --addr tcp://0.0.0.0:$BUILDKIT_PORT
```

#### Trivy

```shell title="bash"
docker run -d -p 8887:8887 --name trivy \
aquasec/trivy:0.50.4 server --listen=0.0.0.0:8887
```
