package main

import (
	//"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/client/cache"
	"k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/controller/framework"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/util/wait"
	"log"
	"time"
)

func main() {
	client, err := unversioned.NewInCluster()
	if err != nil {
		log.Fatal(err)
	}

	eventHandlers := framework.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			np := obj.(*extensions.NetworkPolicy)
			log.Println("Add", np.Spec)
		},
		DeleteFunc: func(obj interface{}) {
			np := obj.(*extensions.NetworkPolicy)
			log.Println("Delete", np)
		},
		UpdateFunc: func(old, new interface{}) {
			log.Println("Update", old, new)
		}}

	npWatch := cache.NewListWatchFromClient(client.ExtensionsClient, "networkpolicies", "", fields.Everything())
	_, npController := framework.NewInformer(npWatch, &extensions.NetworkPolicy{}, 0, eventHandlers)

	// podWatch := cache.NewListWatchFromClient(client, "pods", "", fields.Everything())
	// _, podController := framework.NewInformer(podWatch, &api.Pod{}, 0, eventHandlers)

	go npController.Run(wait.NeverStop)
	// go podController.Run(wait.NeverStop)

	time.Sleep(time.Minute * 5)
}
