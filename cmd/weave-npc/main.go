package main

import (
	"github.com/pkg/errors"
	weavenpc "github.com/weaveworks/weave-npc/pkg/controller"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/client/cache"
	"k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/controller/framework"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/runtime"
	utildbus "k8s.io/kubernetes/pkg/util/dbus"
	utilexec "k8s.io/kubernetes/pkg/util/exec"
	utiliptables "k8s.io/kubernetes/pkg/util/iptables"
	utilwait "k8s.io/kubernetes/pkg/util/wait"
	"log"
	"os/exec"
	"time"
)

func handleError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func makeController(getter cache.Getter, resource string,
	objType runtime.Object, handlers framework.ResourceEventHandlerFuncs) *framework.Controller {
	listWatch := cache.NewListWatchFromClient(getter, resource, "", fields.Everything())
	_, controller := framework.NewInformer(listWatch, objType, 0, handlers)
	return controller
}

func ensureFlushedChain(ipt utiliptables.Interface, chain utiliptables.Chain) error {
	needFlush, err := ipt.EnsureChain(utiliptables.TableFilter, chain)
	if err != nil {
		return err
	}

	if needFlush {
		if err := ipt.FlushChain(utiliptables.TableFilter, chain); err != nil {
			return err
		}
	}
	return nil
}

func resetIPTables(ipt utiliptables.Interface) error {
	// Flush chains first so there are no refs to extant ipsets
	if err := ensureFlushedChain(ipt, weavenpc.IngressChain); err != nil {
		return err
	}

	if err := ensureFlushedChain(ipt, weavenpc.DefaultChain); err != nil {
		return err
	}

	if err := ensureFlushedChain(ipt, weavenpc.MainChain); err != nil {
		return err
	}

	// Configure main chain static rules
	if _, err := ipt.EnsureRule(utiliptables.Append, utiliptables.TableFilter, weavenpc.MainChain,
		"-m", "state", "--state", "RELATED,ESTABLISHED", "-j", "ACCEPT"); err != nil {
		return err
	}

	if _, err := ipt.EnsureRule(utiliptables.Append, utiliptables.TableFilter, weavenpc.MainChain,
		"-m", "state", "--state", "NEW", "-j", "WEAVE-NPC-DEFAULT"); err != nil {
		return err
	}

	if _, err := ipt.EnsureRule(utiliptables.Append, utiliptables.TableFilter, weavenpc.MainChain,
		"-m", "state", "--state", "NEW", "-j", "WEAVE-NPC-INGRESS"); err != nil {
		return err
	}

	if _, err := ipt.EnsureRule(utiliptables.Append, utiliptables.TableFilter, weavenpc.MainChain,
		"-j", "DROP"); err != nil {
		return err
	}

	// TODO should restrict ipset operations to the `weave-` prefix:

	// Now flush the ipsets to clear out list:set interdependencies
	if err := exec.Command("ipset", "flush").Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return errors.Wrapf(err, "ipset flush failed: %s", ee.Stderr)
		} else {
			return errors.Wrapf(err, "ipset flush failed")
		}
	}

	// Finally destroy the ipsets
	if err := exec.Command("ipset", "destroy").Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return errors.Wrapf(err, "ipset destroy failed: %s", ee.Stderr)
		} else {
			return errors.Wrapf(err, "ipset destroy failed")
		}
	}

	return nil
}

func main() {

	client, err := unversioned.NewInCluster()
	if err != nil {
		log.Fatal(err)
	}

	ipt := utiliptables.New(utilexec.New(), utildbus.New(), utiliptables.ProtocolIpv4)

	handleError(resetIPTables(ipt))

	npc := weavenpc.New(ipt)

	nsController := makeController(client, "namespaces", &api.Namespace{},
		framework.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				handleError(npc.AddNamespace(obj.(*api.Namespace)))
			},
			DeleteFunc: func(obj interface{}) {
				handleError(npc.DeleteNamespace(obj.(*api.Namespace)))
			},
			UpdateFunc: func(old, new interface{}) {
				handleError(npc.UpdateNamespace(old.(*api.Namespace), new.(*api.Namespace)))
			}})

	podController := makeController(client, "pods", &api.Pod{},
		framework.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				handleError(npc.AddPod(obj.(*api.Pod)))
			},
			DeleteFunc: func(obj interface{}) {
				handleError(npc.DeletePod(obj.(*api.Pod)))
			},
			UpdateFunc: func(old, new interface{}) {
				handleError(npc.UpdatePod(old.(*api.Pod), new.(*api.Pod)))
			}})

	npController := makeController(client.ExtensionsClient, "networkpolicies", &extensions.NetworkPolicy{},
		framework.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				handleError(npc.AddNetworkPolicy(obj.(*extensions.NetworkPolicy)))
			},
			DeleteFunc: func(obj interface{}) {
				handleError(npc.DeleteNetworkPolicy(obj.(*extensions.NetworkPolicy)))
			},
			UpdateFunc: func(old, new interface{}) {
				handleError(npc.UpdateNetworkPolicy(old.(*extensions.NetworkPolicy), new.(*extensions.NetworkPolicy)))
			}})

	go nsController.Run(utilwait.NeverStop)
	go podController.Run(utilwait.NeverStop)
	go npController.Run(utilwait.NeverStop)

	// TODO wait for signal here
	time.Sleep(time.Minute * 5)

}
