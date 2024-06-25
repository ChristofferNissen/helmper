---
sidebar_label: 'Compatibility'
sidebar_position: 5
---

# Compatibility

Helmper utilizes the Helm SDK to maintain full compatibility with both Helm Repositories and OCI registries for storing Helm Charts.

In practice, Helmper currently pushes charts and images to the same destination registry, so it must be OCI compliant. 

Helmper utilizes `oras-go` to push OCI artifacts. Helmper utilizes the Helm SDK to push Helm Charts, as the Helm SDK sets the correct metadata attributes.

Oras and Helm state support all registries with OCI support, for example:

- [(Amazon Elastic Container Registry)](https://docs.aws.amazon.com/AmazonECR/latest/userguide/push-oci-artifact.html)  
- [Azure Container Registry](https://docs.microsoft.com/azure/container-registry/container-registry-helm-repos#push-chart-to-registry-as-oci-artifact)
- [CNCF Distribution](https://oras.land/docs/compatible_oci_registries#cncf-distribution) - local/offline verification
- [Docker Hub](https://docs.docker.com/docker-hub/oci-artifacts/)
- [Google Artifact Registry](https://cloud.google.com/artifact-registry/docs/helm/manage-charts)
- [GitHub Packages container registry](https://oras.land/docs/compatible_oci_registries#github-packages-container-registry-ghcr)
- [Harbor](https://goharbor.io/docs/main/administration/user-defined-oci-artifact/)
- [JFrog Artifactory](https://jfrog.com/help/r/jfrog-artifactory-documentation/helm-oci-repositories)
- [IBM Cloud Container Registry](https://cloud.ibm.com/docs/Registry?topic=Registry-registry_helm_charts)
- [Zot Registry](https://zotregistry.dev/)

Sources: [Helm](https://helm.sh/docs/topics/registries/#use-hosted-registries) [Oras](https://oras.land/docs/compatible_oci_registries)

For testing, Helmper is using the [CNCF Distribution](https://github.com/distribution/distribution) registry.

:::note

Amazon Elastic Container Registry (**ECR**) currently has a problem. When pushing new artifacts to ECR, repositories are not created automatically. If the repositories are created up front, Helmper works with ECR. Otherwise you will get a 404 error.

:::