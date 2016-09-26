Kubernetes Network Policy Controller

# Building

First clone the repo to the correct location in your `$GOPATH`:

    git clone git@github.com:weaveworks/weave-npc $GOPATH/src/github.com/weaveworks/weave-npc

Then build the `weaveworks/weave-npc` container image:

    cd $GOPATH/src/github.com/weaveworks/weave-npc
    make image

# Deploy

Minikube is recommended for testing - the `minikube-publish` target
transfers the `weaveworks/weave-npc` image directly into the minikube
VM, bypassing the Docker hub:

    minikube start
    cd $GOPATH/src/github.com/weaveworks/weave-npc
    make minikube-publish kube-deploy

Alternatively you can deploy to any k8s installation with kubectl
after publishing the image to the Docker hub:

    make publish-image
    kubectl create -f k8s/daemonset.yaml

# Use

Create a network policy in the `default` namespace:

    kubectl create -f k8s/np.yaml

Set the `default` namespace ingress isolation policy to `DefaultDeny`:

    kubectl apply -f k8s/default-ns-deny.yaml
