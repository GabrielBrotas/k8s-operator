# Step By Step - Local Development

## 1 - Setting up a cluster on Minikube

```sh
minikube start --kubernetes-version=v1.28.3
```

## 2 - Setting up the Operator SDK

1. create a new directory for the operator

```sh
mkdir domain-operator && cd domain-operator
```

2. Initialize the operator

```sh
operator-sdk init --domain=platform.com --repo=github.com/gabriel-brotas/domain-operator
```

3. Create a new API

```sh
operator-sdk create api --group domain --version v1alpha1 --kind Domain
```

4. Download the dependencies

```sh
go mod tidy
go mod vendor
```

5. Implement the controller logic

A Kubernetes Operator runs iteratively to reconcile the state of your application, it's very important to write the controller to be **idempotent**: In other words, the controller can run the code multiple times without creating multiple instances of a resource.

The following file includes a controller that reconciles the state of a Domain resource:

[domain-operator/internal/controller/domain_controller.go](./domain-operator/internal/controller/domain_controller.go)

## 3 - Create the Database for the Operator

1. Deploy the database

```sh
helm repo add bitnami https://charts.bitnami.com/bitnami
helm install postgresql -f ./db/values.yaml bitnami/postgresql
```

2. DB Info

```sh
# DNS: postgresql.default.svc.cluster.local - Read/Write connection

# Get Password
##  To get the password for "postgres" run:
export POSTGRES_ADMIN_PASSWORD=$(kubectl get secret --namespace default postgresql -o jsonpath="{.data.postgres-password}" | base64 -d)

## To get the password for "admin" run:
export POSTGRES_PASSWORD=$(kubectl get secret --namespace default postgresql -o jsonpath="{.data.password}" | base64 -d)

# To connect to your database run the following command:
kubectl run postgresql-client --rm --tty -i --restart='Never' --namespace default --image docker.io/bitnami/postgresql:15.4.0-debian-11-r10 --env="PGPASSWORD=$POSTGRES_PASSWORD" \
      --command -- psql --host postgresql -U admin -d postgresqlDB -p 5432
```

4. Port Forward

```sh
kubectl port-forward svc/postgresql 5432:5432
```

## 4 - Deploy the Operator

1. Install the CRDs into the cluster

```sh
make install
```

This command registers our custom kind schema (`Domain` in this case) within our Kubernetes cluster. Now any new request specifying this kind will be forwarded to our `Domain` controller internally.

2. Deploy the operator to the cluster

```sh
make run
```

## 5 - Create a Domain resource

1. from the root of the project, run the following command to create a new Domain resource

```sh
kubectl apply -f manifests/domain-a.yaml
kubectl apply -f manifests/domain-b.yaml
```
