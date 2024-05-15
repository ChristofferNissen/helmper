---
sidebar_label: 'ER Diagram'
sidebar_position: 1
---

# ER

In the diagram above it can be seen how the different OCI entities relate. This is the structure that Helmper parses and conceptually handles before considering distributing/patching/signing artifacts.

Helmper parses Helm Charts from remote registries, identifies enabled dependency charts and analyses all values.yaml files for references to container images.

```mermaid
erDiagram
    REGISTRY ||--o{ "Helm Chart" : contains
    REGISTRY ||--o{ "Container Image (OCI)" : contains
    
    "Helm Chart" ||--|{ "values.yaml" : has
    "Helm Chart" ||--o{ "Dependency Helm Chart" : has
    "Dependency Helm Chart" ||--|{ "values.yaml" : has
    "values.yaml" ||--|{  "Container Image (OCI)" : references

    "Container Image (OCI)"{
        string Registry
        string Repository
        string Name
        string Tag
    }

    "Container Image (OCI)" ||--o| "Signature" : has
    "Container Image (OCI)" ||--o| "Digest" : has
```
