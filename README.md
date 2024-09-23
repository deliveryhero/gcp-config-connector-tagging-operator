# Delivery Hero gcp-config-connector-tagging-operator

[![dh](./img/dh-logo.png)](#)

gcp-config-connector-tagging-operator helps you to add tags to GCP resources managed by the config-connector controller.

## Description

We at Delivery Hero have requirements to limit access to the GCP resources based on tags with Config Connector (implementing [ABAC](https://cloud.google.com/iam/docs/tags-access-control)).
To do this with our Kubernetes-centric setup, leveraging [the GCP config-connector project](https://github.com/GoogleCloudPlatform/k8s-config-connector), we needed a way to dynamically create and update the tag values in a project - even if the same value is used in more than one namespace or cluster.
This project helps solve this issue by adding a layer on top that can sync tag keys and values in GCP from Kubernetes labels, and then take care of generating the necessary tag binding config-connector resources.
In the end, this gives you an auto-magical experience for tags similar to how Kubernetes labels are also automatically made available as resource labels by config-connector.

## Getting Started

### Prerequisites

- go version v1.22.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### To Deploy on the cluster

**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/gcp-config-connector-tagging-operator:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/gcp-config-connector-tagging-operator:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
> privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

> **NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall

**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Following are the steps to build the installer and distribute this project to users.

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/gcp-config-connector-tagging-operator:tag
```

NOTE: The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without
its dependencies.

2. Using the installer

Users can just run kubectl apply -f <URL for YAML BUNDLE> to install the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/gcp-config-connector-tagging-operator/<tag or branch>/dist/install.yaml
```

## Contributing

To contribute, please read our [contributing docs](CONTRIBUTING.md).
**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright © 2024 Delivery Hero SE
