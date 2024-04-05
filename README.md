# Kubernetes Operator

## Tools

- Minikube - v1.32.0
- https://sdk.operatorframework.io/ - v1.34.1
- Golang = 1.21.9

## The role and behavior of Kubernetes Operators

A Kubernetes Operator manages your application's logistics. It contains code called a **controller** that runs periodically and checks the _current state_ of your service's namespaced resources against the _desired state_. If the controller finds any differences, it restores your service to the desired state in a process called reconciliation. For instance, if a resource crashed, the controller restarts it.

You can imagine an unofficial agreement between you and the Kubernetes Operator:

- You: "Hey Opo, I am creating the following resources. Now it's your responsibility to keep them running."
- Operator: "Roger that! Will check back regularly."

You can build an operator with Helm Charts, Ansible playbooks, or Golang. In this article, we use Golang. We'll focus on a namespace-scoped operator (as opposed to a cluster-scoped operator) because it's more flexible and because we want to control only our own application. See the Kubernetes Operators 101 series for more background on operators.

## Operator Overview

for this project, we will create a Kubernetes Operator that manages a platform for domains. The operator will create a new management namespace for each domain hold the following information:

- Domain ID
- Environemnts

Example of a `Domain`:

```yaml
apiVersion: domain.platform.com/v1
kind: Domain
metadata:
  name: domain1
spec:
  domainID: domain1 # must be equal to the metadata name
  environments:
    - dev
    - staging
    - production
```

The Operator will:

- Create a Management Namespace for each Domain
- Store the Domain data in a database

## How to Run

### 1 - Setting up a cluster on Minikube

1. Start the cluster

```sh
minikube start --kubernetes-version=v1.28.3
```

2. Create the namespace for the operator

```sh
kubectl create namespace domain-operator-system
```

### 2 - Create the Database Required for the Operator

1. Deploy the database

```sh
helm repo add bitnami https://charts.bitnami.com/bitnami
helm install -n domain-operator-system postgresql -f ./db/values.yaml bitnami/postgresql --create-namespace
```

The `db/values.yaml` contains the configuration for the database, including the password for the admin user and the init script to initialize the database tables.

2. Verify the database is running

```sh
$ kubectl get pods -n domain-operator-system

NAME           READY   STATUS    RESTARTS   AGE
postgresql-0   1/1     Running   0          48s

$ kubectl logs -n domain-operator-system postgresql-0
...
2024-04-05 18:31:12.858 GMT [1] LOG:  database system is ready to accept connections
```

### 3 - Build the Domain Operator

1. Move to the operator directory to execute the `make` commands:

```sh
cd domain-operator
```

1. Build the Operator Docker image

```sh
make docker-build IMAGE_TAG_BASE=gbrotas/domain-operator VERSION=0.0.1
```

2. Push the image to the Docker Hub

```sh
make docker-push IMAGE_TAG_BASE=gbrotas/domain-operator VERSION=0.0.1
```

### 4 - Run the Operator

There are three ways to run the operator:

- As a Go program outside a cluster
- As a Deployment inside a Kubernetes cluster
- Managed by the Operator Lifecycle Manager (OLM) in bundle format

In this project, we will run the operator as a Deployment inside a Kubernetes cluster.

1. Install the CRDs into the cluster

```sh
make install
```

2. Deploy the operator to the cluster

By default, a new namespace is created with name `<project-name>-system`, ex. `domain-operator-system`, and will be used for the deployment.

Run the following to deploy the operator:

```sh
make deploy IMG=gbrotas/domain-operator:0.0.1
```

Verify that the memcached-operator is up and running:

```sh
$ kubectl get deployment -n domain-operator-system
NAME                                    READY   UP-TO-DATE   AVAILABLE   AGE
domain-operator-controller-manager   1/1     1            1           18m

$ kubectl get pods -n domain-operator-system
NAME                                                  READY   STATUS    RESTARTS   AGE
domain-operator-controller-manager-5fdfcc86f6-p5bwh   2/2     Running   0          103s
postgresql-0                                          1/1     Running   0          18m
```

You can follow the logs of the operator by running:

```sh
kubectl logs -n domain-operator-system domain-operator-controller-manager-5fdfcc86f6-p5bwh -c manager --follow
```

_Note: replace the pod name with the one in your cluster_

### 4 - Create a Domain resource

1. from the root of the project, run the following command to create a new Domain resource

```sh
$ kubectl apply -f manifests/domain-a.yaml
domain.domain.platform.com/domain-a created
```

2. Verify that the Domain resource was created

```sh
$ kubectl get domain
NAME       AGE
domain-a   19s
```

When a domain is created, the operator will create a new namespace with the same name as the domain. Verify that the namespace was created:

```sh
$ kubectl get ns domain-a
NAME       STATUS   AGE
domain-a   Active   67s
```

Additionally, if you remove the Domain resource, the operator will delete the namespace created for the domain.

## Refs:

- https://betterprogramming.pub/build-a-kubernetes-operator-in-10-minutes-11eec1492d30
- https://dev.to/eminetto/creating-kubernetes-operators-with-operator-sdk-49f9
- https://medium.com/developingnodes/mastering-kubernetes-operators-your-definitive-guide-to-starting-strong-70ff43579eb9
