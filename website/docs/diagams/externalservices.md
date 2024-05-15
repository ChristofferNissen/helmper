---
sidebar_label: 'External Services Diagram'
sidebar_position: 2
---

# External services

This diagram illustrates the external services Helmper communicates with. Helmper will always be interacting with OCI registries and Chart repositories. If you have enabled Copacetic in the configuration, Helmper will also communicate with an external Trivy server and Buildkit daemon.

```mermaid
graph LR;
 service[helmper]--->|"<.registries[].url>"|pod3[OCI Registry];
 service[helmper]--->|"<.charts[].repo.url>"|pod4[Chart Repository];
 
 service[helmper]-..->|<.import.copacetic.trivy.addr>|pod1[Trivy];
 service[helmper]-..->|<.import.copacetic.buildkitd.addr>|pod2[Buildkit];
```
