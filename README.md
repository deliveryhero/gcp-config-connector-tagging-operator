# Delivery Hero GCP Config Connector Tagging Operator

[![Delivery Hero](./img/dh-logo.png)](#)

GCP Config Connector Tagging Operator helps you to add tags to GCP resources managed by the Config Connector controller.

## Description

At Delivery Hero, we have requirements to limit access to GCP resources based on tags with Config Connector (implementing [ABAC](https://cloud.google.com/iam/docs/tags-access-control)). To achieve this in our Kubernetes-centric setup, leveraging [the GCP Config Connector project](https://github.com/GoogleCloudPlatform/k8s-config-connector), we needed a way to dynamically create and update tag values in a project—even if the same value is used in more than one namespace or cluster.

This project helps solve this issue by adding a layer that syncs tag keys and values in GCP from Kubernetes labels. It then generates the necessary tag binding Config Connector resources, providing an automagical experience for tags, similar to how Kubernetes labels are automatically made available as resource labels by Config Connector.

> **Note:** This operator requires the `TagsLocationTagBinding` CRD from the Config Connector Operator. This CRD might need to be installed manually, as it is only available at the v1alpha1 level currently. You can find instructions on how to install it [here](https://cloud.google.com/config-connector/docs/how-to/install-alpha-crds).


## Getting Started

### Prerequisites

- Go version v1.22.0+
- Docker version 17.03+
- Kubectl version v1.11.3+
- Access to a Kubernetes v1.11.3+ cluster with Config Connector v1.121.0+ installed.

### Deploying on the Cluster Using Helm

**Install the chart from the Helm repository:**

```sh
helm install gcp-config-connector-tagging-operator oci://ghcr.io/deliveryhero/gcp-config-connector-tagging-operator/helm-chart/gcp-config-connector-tagging-operator \
  --create-namespace \
  --namespace "gcp-config-connector-tagging-operator-system"
```

### Grant the `tagAdmin` Role to the Service Account

```sh
PROJECT_ID=<your-gcp-project-id>
PROJECT_NUMBER=<your-gcp-project-number>
gcloud projects add-iam-policy-binding ${PROJECT_ID} \
  --role=roles/resourcemanager.tagAdmin \
  --member=principal://iam.googleapis.com/projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/${PROJECT_ID}.svc.id.goog/subject/ns/gcp-config-connector-tagging-operator-system/sa/gcp-config-connector-tagging-operator-controller-manager \
  --condition=None
```

### Deploying on the Cluster

**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/gcp-config-connector-tagging-operator:tag
```

> **Note:** Ensure this image is published in the personal registry you specified and that you have the proper permissions to pull the image from the working environment.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/gcp-config-connector-tagging-operator:tag
```

> **Note:** If you encounter RBAC errors, you may need to grant yourself cluster-admin privileges or be logged in as an admin.

**Create instances of your solution by applying the samples from the `config/samples` directory:**

```sh
kubectl apply -k config/samples/
```

> **Note:** Ensure that the samples have default values to test it out.

### Uninstalling

**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs (CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

### Building the Installer

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/gcp-config-connector-tagging-operator:tag
```

> **Note:** This generates an `install.yaml` file in the `dist` directory, containing all the Kustomize-built resources necessary to install this project without its dependencies.

### Using the Installer

Users can install the project by running the following command:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/gcp-config-connector-tagging-operator/<tag or branch>/dist/install.yaml
```

## Contributing

To contribute, please read our [contributing documentation](CONTRIBUTING.md).

> **Note:** Run `make help` for more information on all potential `make` targets. Additional information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html).

## License

&copy; 2024 Delivery Hero SE
Contents of this repository is licensed under the Apache-2.0 [License](LICENSE).
