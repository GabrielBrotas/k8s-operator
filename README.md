# Kubernetes Operator

## Tools

- Minikube - v1.32.0
- https://sdk.operatorframework.io/ - v1.34.1
- Golang = 1.21.9

## The role and behavior of Kubernetes Operators

A Kubernetes Operator manages your application's logistics. It contains code called a controller that runs periodically and checks the current state of your service's namespaced resources against the desired state. If the controller finds any differences, it restores your service to the desired state in a process called reconciliation. For instance, if a resource crashed, the controller restarts it.

You can imagine an unofficial agreement between you and the Kubernetes Operator:

- You: "Hey Opo, I am creating the following resources. Now it's your responsibility to keep them running."
- Operator: "Roger that! Will check back regularly."

You can build an operator with Helm Charts, Ansible playbooks, or Golang. In this article, we use Golang. We'll focus on a namespace-scoped operator (as opposed to a cluster-scoped operator) because it's more flexible and because we want to control only our own application. See the Kubernetes Operators 101 series for more background on operators.

## Getting Started

### 1 - Setting up a cluster on Minikube

```sh
minikube start --kubernetes-version=v1.28.3
```

### 2 - Setting up the Operator SDK

```sh
cd my-operator

go mod init my-operator.com
operator-sdk init
``` 
### 3 - Create APIs and a custom resource

In Kubernetes, the functions exposed for each service you want to provide are grouped together in a resource. Thus, when we create the APIs for our application, we also create their resource through a CustomResourceDefinition (CRD).

The following command creates an API and labels it Traveller through the --kind option. In the YAML configuration files created by the command, you can find a field labeled kind with the value Traveller. This field indicates that Traveller is used throughout the development process to refer to our APIs:
```sh
$ operator-sdk create api --version=v1alpha1 --kind=Traveller

Create Resource [y/n]
y
Create Controller [y/n]
y
...
...
```

We have asked the command also to create a controller to handle all operations corresponding to our kind. The file defining the controller is named traveller_controller.go.

The --version option can take any string, and you can set it to track your development on a project. Here, we've started with a modest value, indicating that our application is in alpha.

### 4 - Implement the controller

### 5 - Build and run the operator

```sh
make install

make run
```

https://medium.com/developingnodes/mastering-kubernetes-operators-your-definitive-guide-to-starting-strong-70ff43579eb9# k8s-operator
