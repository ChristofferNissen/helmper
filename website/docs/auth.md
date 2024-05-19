---
sidebar_label: 'Authentication'
sidebar_position: 6
---

# Authentication

## Helm

Helmper supports all parameters for defining a Helm Repository. [Read more here](https://helm.sh/docs/helm/helm_repo_add/).

```yaml title="Example chart definition"
...
charts:
- name: prometheus
version: 25.8.0
repo:
    name: prometheus-community
    url: https://prometheus-community.github.io/helm-charts/
...
```

`helmper` will also use the authentication information in the file pointed to by the Helm environment variable `HELM_REGISTRY_CONFIG`.

Simply login with Helm:

```shell title="Example Helm login cmd"
helm registry login [host] [flags]
```

Read mere in the official [Helm Documentation](https://helm.sh/docs/helm/helm_registry_login/).

## Registries

For authenticating against registries, `helmper` utilizes the authentication details present in `~/.docker/config.json`.

Simply login with Docker or similar commands from your cloud provider:

```shell title="Example Docker login cmd"
docker login -u USER -p PASS
```

Read more in the official [Docker Documentation](https://docs.docker.com/reference/cli/docker/login/).

### Cloud provider examples

import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

<Tabs>
<TabItem value="acr" label="Azure Container Registry (ACR)">

```shell Title "Azure Example"
az acr login -n <YOUR_ACR_NAME>
```

Read more in [ACR Documentation](https://learn.microsoft.com/en-us/azure/container-registry/container-registry-authentication?tabs=azure-cli).

</TabItem>

<TabItem value="ecr" label="Elastic Container Registry (ECR)">

```shell Title "Amazon Example"
aws ecr get-login-password | docker login -u AWS --password-stdin "https://$(aws sts get-caller-identity --query 'Account' --output text).dkr.ecr.us-east-1.amazonaws.com"
```

Read more in [ECR Documentation](https://docs.aws.amazon.com/AmazonECR/latest/userguide/registry_auth.html).

</TabItem>


</Tabs>
