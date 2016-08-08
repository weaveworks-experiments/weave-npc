package controller

import (
	"fmt"
	"github.com/weaveworks/weave-npc/pkg/ipset"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/types"
	"sync"
)

type NetworkPolicyController interface {
	AddNamespace(ns *api.Namespace) error
	UpdateNamespace(old, new *api.Namespace) error
	DeleteNamespace(ns *api.Namespace) error

	AddPod(obj *api.Pod) error
	UpdatePod(old, new *api.Pod) error
	DeletePod(obj *api.Pod) error

	AddNetworkPolicy(obj *extensions.NetworkPolicy) error
	UpdateNetworkPolicy(old, new *extensions.NetworkPolicy) error
	DeleteNetworkPolicy(obj *extensions.NetworkPolicy) error
}

type controller struct {
	sync.Mutex

	namespaces map[string]*api.Namespace                          // ns name -> ns (for matching NamespaceSelector)
	pods       map[string]map[types.UID]*api.Pod                  // ns name -> pod UID -> pod (for matching PodSelector)
	policies   map[string]map[types.UID]*extensions.NetworkPolicy // ns name -> policy UID -> policy

	nsIPSets          map[string]ipset.HashIP             // ns name -> hash:ip ipset
	podSelectorIPSets map[string]map[string]ipset.HashIP  // ns name -> selector string -> hash:ip ipset
	nsSelectorIPSets  map[string]map[string]ipset.ListSet // ns name -> selector string -> list:set ipset
}

func New() NetworkPolicyController {
	return &controller{}
}

func (npc *controller) AddPod(pod *api.Pod) error {
	npc.Lock()
	defer npc.Unlock()

	haship, found := npc.nsIPSets[pod.ObjectMeta.Namespace]
	if !found {
		haship = ipset.NewHashIP(encodeBase95("ns", pod.ObjectMeta.Namespace))
		npc.nsIPSets[pod.ObjectMeta.Namespace] = haship
	}

	if len(pod.Status.PodIP) > 0 {
		return haship.AddIP(pod.Status.PodIP)
	}

	return nil
}

func (npc *controller) UpdatePod(old, new *api.Pod) error {
	npc.Lock()
	defer npc.Unlock()

	if old.Status.PodIP != new.Status.PodIP {
		haship, found := npc.nsIPSets[old.ObjectMeta.Namespace]
		if !found {
			return fmt.Errorf("Attempt to update pod %s in unknown namespace %s",
				old.ObjectMeta.Name, old.ObjectMeta.Namespace)
		}
		if len(old.Status.PodIP) > 0 {
			if err := haship.DelIP(old.Status.PodIP); err != nil {
				return err
			}
		}
		if len(new.Status.PodIP) > 0 {
			if err := haship.AddIP(new.Status.PodIP); err != nil {
				return err
			}
		}
	}

	return nil
}

func (npc *controller) DeletePod(pod *api.Pod) error {
	npc.Lock()
	defer npc.Unlock()

	if len(pod.Status.PodIP) > 0 {
		haship, found := npc.nsIPSets[pod.ObjectMeta.Namespace]
		if !found {
			return fmt.Errorf("Attempt to delete pod %s in unknown namespace %s",
				pod.ObjectMeta.Name, pod.ObjectMeta.Namespace)
		}
		if err := haship.DelIP(pod.Status.PodIP); err != nil {
			return err
		}
		if haship.Count() == 0 {
			delete(npc.nsIPSets, pod.ObjectMeta.Namespace)
		}
	}

	return nil
}

func (npc *controller) AddNetworkPolicy(np *extensions.NetworkPolicy) error {
	return nil
}

func (npc *controller) UpdateNetworkPolicy(old, new *extensions.NetworkPolicy) error {
	return nil
}

func (npc *controller) DeleteNetworkPolicy(np *extensions.NetworkPolicy) error {
	return nil
}

func (npc *controller) AddNamespace(np *api.Namespace) error {
	return nil
}

func (npc *controller) UpdateNamespace(old, new *api.Namespace) error {
	return nil
}

func (npc *controller) DeleteNamespace(np *api.Namespace) error {
	return nil
}
