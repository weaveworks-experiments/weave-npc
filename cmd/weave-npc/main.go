package main

import (
	log "github.com/Sirupsen/logrus"
	weavenpc "github.com/weaveworks/weave-npc/pkg/controller"
	"github.com/weaveworks/weave-npc/pkg/util/ipset"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/client/cache"
	"k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/controller/framework"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/util/dbus"
	"k8s.io/kubernetes/pkg/util/exec"
	"k8s.io/kubernetes/pkg/util/iptables"
	"k8s.io/kubernetes/pkg/util/wait"
	"os"
	"os/signal"
	"syscall"
)

var version = "(unreleased)"

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

func ensureFlushedChain(ipt iptables.Interface, chain iptables.Chain) error {
	needFlush, err := ipt.EnsureChain(iptables.TableFilter, chain)
	if err != nil {
		return err
	}

	if needFlush {
		if err := ipt.FlushChain(iptables.TableFilter, chain); err != nil {
			return err
		}
	}
	return nil
}

func resetIPTables(ipt iptables.Interface) error {
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
	if _, err := ipt.EnsureRule(iptables.Append, iptables.TableFilter, weavenpc.MainChain,
		"-m", "state", "--state", "RELATED,ESTABLISHED", "-j", "ACCEPT"); err != nil {
		return err
	}

	if _, err := ipt.EnsureRule(iptables.Append, iptables.TableFilter, weavenpc.MainChain,
		"-m", "state", "--state", "NEW", "-j", string(weavenpc.DefaultChain)); err != nil {
		return err
	}

	if _, err := ipt.EnsureRule(iptables.Append, iptables.TableFilter, weavenpc.MainChain,
		"-m", "state", "--state", "NEW", "-j", string(weavenpc.IngressChain)); err != nil {
		return err
	}

	return nil
}

func resetIPSets(ips ipset.Interface) error {
	// TODO should restrict ipset operations to the `weave-` prefix:

	if err := ips.FlushAll(); err != nil {
		return err
	}

	if err := ips.DestroyAll(); err != nil {
		return err
	}

	return nil
}

func main() {
	log.Infof("Starting Weaveworks NPC %s", version)

	client, err := unversioned.NewInCluster()
	if err != nil {
		log.Fatal(err)
	}

	ipt := iptables.New(exec.New(), dbus.New(), iptables.ProtocolIpv4)
	ips := ipset.New()

	handleError(resetIPTables(ipt))
	handleError(resetIPSets(ips))

	npc := weavenpc.New(ipt, ips)

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

	go nsController.Run(wait.NeverStop)
	go podController.Run(wait.NeverStop)
	go npController.Run(wait.NeverStop)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	log.Fatalf("Exiting: %v", <-signals)
}
