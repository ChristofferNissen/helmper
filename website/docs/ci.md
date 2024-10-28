---
sidebar_label: 'Pipeline Examples'
sidebar_position: 10
---

# Pipeline examples

In this section you can find example pipelines to quickly include Helmper in your pipelines.


import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

<Tabs>
<TabItem value="yaml" label="Azure DevOps Pipelines">

```yaml
trigger:
- main

pool:
  vmImage: 'ubuntu-latest'

steps:
- task: Bash@3
  displayName: 'Install latest Helmper'
  inputs:
    targetType: 'inline'
    script: |
      VERSION=$(curl -Lso /dev/null -w %{url_effective} https://github.com/christoffernissen/helmper/releases/latest | grep -o '[^/]*$')
      curl -LO https://github.com/christoffernissen/helmper/releases/download/$VERSION/helmper-linux-amd64
      chmod +x helmper-linux-amd64
      mv helmper-linux-amd64 /usr/local/bin/helmper

- task: Bash@3
  displayName: 'Login registry'
  inputs:
    targetType: 'inline'
    script: |
      az acr login -n <YOUR_REGISTRY_NAME>

- task: Bash@3
  displayName: 'Generate sample configuration'
  inputs:
    targetType: 'inline'
    script: |
      cat <<EOF >>helmper.config
      k8s_version: 1.31.1
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
      - name: <YOUR_REGISTRY_NAME>
        url: <YOUR_REGISTRY_URL>
      EOF

- task: Bash@3
  displayName: 'Run Helmper'
  inputs:
    targetType: 'inline'
    script: |
      /usr/local/bin/helmper
```

</TabItem>
</Tabs>