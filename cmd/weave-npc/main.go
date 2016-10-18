package main

import (
	"os"
	"os/signal"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	coreapi "k8s.io/client-go/pkg/api/v1"
	extnapi "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/pkg/util/wait"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kubernetes/pkg/util/dbus"
	"k8s.io/kubernetes/pkg/util/exec"
	"k8s.io/kubernetes/pkg/util/iptables"

	weavenpc "github.com/weaveworks/weave-npc/pkg/controller"
	"github.com/weaveworks/weave-npc/pkg/metrics"
	"github.com/weaveworks/weave-npc/pkg/ulogd"
	"github.com/weaveworks/weave-npc/pkg/util/ipset"
)

var (
	version     = "(unreleased)"
	metricsAddr string
)

func handleError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func makeController(getter cache.Getter, resource string,
	objType runtime.Object, handlers cache.ResourceEventHandlerFuncs) *cache.Controller {
	listWatch := cache.NewListWatchFromClient(getter, resource, "", fields.Everything())
	_, controller := cache.NewInformer(listWatch, objType, 0, handlers)
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

func root(cmd *cobra.Command, args []string) {
	log.Infof("Starting Weaveworks NPC %s", version)

	if err := metrics.Start(metricsAddr); err != nil {
		log.Fatalf("Failed to start metrics: %v", err)
	}

	if err := ulogd.Start(); err != nil {
		log.Fatalf("Failed to start ulogd: %v", err)
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	ipt := iptables.New(exec.New(), dbus.New(), iptables.ProtocolIpv4)
	ips := ipset.New()

	handleError(resetIPTables(ipt))
	handleError(resetIPSets(ips))

	npc := weavenpc.New(ipt, ips)

	nsController := makeController(client.Core().RESTClient(), "namespaces", &coreapi.Namespace{},
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				handleError(npc.AddNamespace(obj.(*coreapi.Namespace)))
			},
			DeleteFunc: func(obj interface{}) {
				handleError(npc.DeleteNamespace(obj.(*coreapi.Namespace)))
			},
			UpdateFunc: func(old, new interface{}) {
				handleError(npc.UpdateNamespace(old.(*coreapi.Namespace), new.(*coreapi.Namespace)))
			}})

	podController := makeController(client.Core().RESTClient(), "pods", &coreapi.Pod{},
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				handleError(npc.AddPod(obj.(*coreapi.Pod)))
			},
			DeleteFunc: func(obj interface{}) {
				handleError(npc.DeletePod(obj.(*coreapi.Pod)))
			},
			UpdateFunc: func(old, new interface{}) {
				handleError(npc.UpdatePod(old.(*coreapi.Pod), new.(*coreapi.Pod)))
			}})

	npController := makeController(client.Extensions().RESTClient(), "networkpolicies", &extnapi.NetworkPolicy{},
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				handleError(npc.AddNetworkPolicy(obj.(*extnapi.NetworkPolicy)))
			},
			DeleteFunc: func(obj interface{}) {
				handleError(npc.DeleteNetworkPolicy(obj.(*extnapi.NetworkPolicy)))
			},
			UpdateFunc: func(old, new interface{}) {
				handleError(npc.UpdateNetworkPolicy(old.(*extnapi.NetworkPolicy), new.(*extnapi.NetworkPolicy)))
			}})

	go nsController.Run(wait.NeverStop)
	go podController.Run(wait.NeverStop)
	go npController.Run(wait.NeverStop)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	log.Fatalf("Exiting: %v", <-signals)
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "weave-npc",
		Short: "Weaveworks Kubernetes Network Policy Controller",
		Run:   root}

	rootCmd.PersistentFlags().StringVar(&metricsAddr, "metrics-addr", ":8686", "metrics server bind address")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
