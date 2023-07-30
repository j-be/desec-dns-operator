# desec-dns-operator
A Kubernetes operator to create and update [deSEC](https://desec.io/) domains.

## Description
This operator will manage your deSEC domains based on information in your `Ingress` resources.
You can also add domains manually using the provided CRD.

This project is still experimental and should be used with caution.

## Installation

The installation is based on [Kustomize](https://kustomize.io/).
We assume you are familiar with it.

The operator takes reads its config, as well as the credentials form file.
The imho. easiest way to provide those is via mounting a `ConfigMap` and a `Secret` as volumes to `/mnt/config` and `/mnt/secret` respectively.

First of all, you'll need a `ConfigMap` like:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: desec-dns-operator
data:
  domain: your-domain.dedyn.io
  namespace: desec-dns-operator
```

You may also use Kustomize's `configMapGenerator`, or any other way to provide the `ConfigMap`.

As `domain` provide a domain on deSEC.io. Either choose one you already own, or one that does not exist yet.
If the domain does not exist yet, the operator will create it.

**Beware:** If the domain already exists, the operator will overwrite the IP associated with it.
The operator assumes this domain is only used for the cluster the operator is running in.
If that is not the case for you **DO NOT USE THE OPERATOR** as is.

For `namespace` choose any existing Kubernetes namespace.
This is the namespace where the Custom Resources associated with the domains will be created.
If you don't have a reason not to, simply use the namespace which contains the operator itself,

Next, you need a `Secret` containing your deSEC token.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: desec-token
type: Opaque
data:
  token: <your-token-as-Base64>
```

Finally, add those to your `Deployment` and set the image tag to the version you want to deploy (see [here](https://github.com/j-be/desec-dns-operator/pkgs/container/desec-dns-operator) for all available versions) using a `kustomization.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: controller-manager
spec:
  template:
    spec:
      containers:
      - name: manager
        image: ghcr.io/j-be/desec-dns-operator:<the-version-to-be-deployed>
        volumeMounts:
        - name: desec-config
          mountPath: /mnt/config
          readOnly: true
        - name: desec-secret
          mountPath: /mnt/secret
          readOnly: true
      volumes:
      - name: desec-config
        configMap:
          name: desec-dns-operator
      - name: desec-secret
        secret:
          secretName: desec-token
```

That's it, deploy using Kustomize and enjoy.

## Development - Getting Started
You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### Running on the cluster
1. Install Instances of Custom Resources:

```sh
kubectl apply -f config/samples/
```

2. Build and push your image to the location specified by `IMG`:

```sh
make docker-build docker-push IMG=<some-registry>/desec-dns-operator:tag
```

3. Deploy the controller to the cluster with the image specified by `IMG`:

```sh
make deploy IMG=<some-registry>/desec-dns-operator:tag
```

### Uninstall CRDs
To delete the CRDs from the cluster:

```sh
make uninstall
```

### Undeploy controller
UnDeploy the controller from the cluster:

```sh
make undeploy
```

## Contributing
If you can: fork, patch, push, PR.

If not, consider leaving an issue and we can discuss stuff there.

### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/),
which provide a reconcile function responsible for synchronizing resources until the desired state is reached on the cluster.

### Test It Out
1. Install the CRDs into the cluster:

```sh
make install
```

2. Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

```sh
make run
```

**NOTE:** You can also run this in one step by running: `make install run`

### Modifying the API definitions
If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
