package controller

import (
	"github.com/weaveworks/weave-npc/pkg/ipset"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/types"
)

type ns struct {
	name         string
	namespace    *api.Namespace
	pods         map[types.UID]*api.Pod                  // pod UID -> k8s Pods
	policies     map[types.UID]*extensions.NetworkPolicy // policy UID -> k8s NetworkPolicy
	ipset        ipset.HashIP                            // hash:ip ipset of pod IPs in this namespace
	podSelectors map[string]*podSelector                 // selector string -> podSelector
}

func newNS(name string) *ns {
	return &ns{
		name:         name,
		pods:         make(map[types.UID]*api.Pod),
		policies:     make(map[types.UID]*extensions.NetworkPolicy),
		ipset:        ipset.NewHashIP(encodeBase95("ns", name)),
		podSelectors: make(map[string]*podSelector)}
}

func (ns *ns) empty() bool {
	return len(ns.pods) == 0 && len(ns.policies) == 0 && ns.namespace == nil
}

func (ns *ns) addPod(obj *api.Pod) error {
	ns.pods[obj.ObjectMeta.UID] = obj
	return ns.addPodIP(obj)
}

func (ns *ns) updatePod(oldObj, newObj *api.Pod) error {
	if oldObj.Status.PodIP != newObj.Status.PodIP {
		if err := ns.delPodIP(oldObj); err != nil {
			return err
		}
		if err := ns.addPodIP(newObj); err != nil {
			return err
		}
	}

	// TODO re-evaluate on label change

	return nil
}

func (ns *ns) deletePod(obj *api.Pod) error {
	if err := ns.delPodIP(obj); err != nil {
		return err
	}
	delete(ns.pods, obj.ObjectMeta.UID)
	return nil
}

func (ns *ns) addNetworkPolicy(obj *extensions.NetworkPolicy) error {
	return nil
}

func (ns *ns) updateNetworkPolicy(oldObj, newObj *extensions.NetworkPolicy) error {
	return nil
}

func (ns *ns) deleteNetworkPolicy(obj *extensions.NetworkPolicy) error {
	return nil
}

func (ns *ns) addNamespace(obj *api.Namespace) error {
	return nil
}

func (ns *ns) updateNamespace(oldObj, newObj *api.Namespace) error {
	return nil
}

func (ns *ns) deleteNamespace(obj *api.Namespace) error {
	return nil
}

func (ns *ns) addPodIP(pod *api.Pod) error {
	if len(pod.Status.PodIP) == 0 {
		return nil
	}
	if err := ns.ipset.AddIP(pod.Status.PodIP); err != nil {
		return err
	}
	for _, ps := range ns.podSelectors {
		if err := ps.addPodIP(pod); err != nil {
			return err
		}
	}
	return nil
}

func (ns *ns) delPodIP(pod *api.Pod) error {
	if len(pod.Status.PodIP) == 0 {
		return nil
	}
	if err := ns.ipset.DelIP(pod.Status.PodIP); err != nil {
		return err
	}
	for _, ps := range ns.podSelectors {
		if err := ps.delPodIP(pod); err != nil {
			return err
		}
	}
	return nil
}
