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
	ipset        ipset.IPSet                             // hash:ip ipset of pod IPs in this namespace
	nsSelectors  map[string]*selector                    // selector string -> nsSelector
	podSelectors map[string]*selector                    // selector string -> podSelector
}

func newNS(name string, nsSelectors map[string]*nsSelector) *ns {
	return &ns{
		name:         name,
		pods:         make(map[types.UID]*api.Pod),
		policies:     make(map[types.UID]*extensions.NetworkPolicy),
		ipset:        ipset.NewHashIP(encodeBase95("ns", name)),
		nsSelectors:  make(map[string]*selector),
		podSelectors: make(map[string]*selector)}
}

func (ns *ns) empty() bool {
	return len(ns.pods) == 0 && len(ns.policies) == 0 && ns.namespace == nil
}

func (ns *ns) addPod(obj *api.Pod) error {
	ns.pods[obj.ObjectMeta.UID] = obj

	if !hasIP(obj) {
		return nil
	}

	return ns.addToMatching(obj)
}

func (ns *ns) updatePod(oldObj, newObj *api.Pod) error {
	delete(ns.pods, oldObj.ObjectMeta.UID)
	ns.pods[newObj.ObjectMeta.UID] = newObj

	if !hasIP(oldObj) && !hasIP(newObj) {
		return nil
	}

	if hasIP(oldObj) && !hasIP(newObj) {
		return ns.delFromMatching(oldObj)
	}

	if !hasIP(oldObj) && hasIP(newObj) {
		return ns.addToMatching(newObj)
	}

	if !equals(oldObj.ObjectMeta.Labels, newObj.ObjectMeta.Labels) ||
		oldObj.Status.PodIP != newObj.Status.PodIP {

		for _, ps := range ns.podSelectors {
			oldMatch := ps.matches(oldObj.ObjectMeta.Labels)
			newMatch := ps.matches(newObj.ObjectMeta.Labels)
			if oldMatch == newMatch && oldObj.Status.PodIP == newObj.Status.PodIP {
				continue
			}
			if oldMatch {
				if err := ps.delEntry(oldObj.Status.PodIP); err != nil {
					return err
				}
			}
			if newMatch {
				if err := ps.addEntry(newObj.Status.PodIP); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (ns *ns) deletePod(obj *api.Pod) error {
	delete(ns.pods, obj.ObjectMeta.UID)

	if !hasIP(obj) {
		return nil
	}

	return ns.delFromMatching(obj)
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
	ns.namespace = obj

	for _, nss := range ns.nsSelectors {
		if nss.matches(obj.ObjectMeta.Labels) {
			if err := nss.addEntry(ns.ipset.Name()); err != nil {
				return err
			}
		}
	}

	return nil
}

func (ns *ns) updateNamespace(oldObj, newObj *api.Namespace) error {
	ns.namespace = newObj

	if !equals(oldObj.ObjectMeta.Labels, newObj.ObjectMeta.Labels) {
		for _, nss := range ns.nsSelectors {
			oldMatch := nss.matches(oldObj.ObjectMeta.Labels)
			newMatch := nss.matches(newObj.ObjectMeta.Labels)
			if oldMatch == newMatch {
				continue
			}
			if oldMatch {
				if err := nss.delEntry(ns.ipset.Name()); err != nil {
					return err
				}
			}
			if newMatch {
				if err := nss.addEntry(ns.ipset.Name()); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (ns *ns) deleteNamespace(obj *api.Namespace) error {
	ns.namespace = nil

	for _, nss := range ns.nsSelectors {
		if nss.matches(obj.ObjectMeta.Labels) {
			if err := nss.delEntry(ns.ipset.Name()); err != nil {
				return err
			}
		}
	}

	return nil
}

func (ns *ns) addToMatching(obj *api.Pod) error {
	if err := ns.ipset.AddEntry(obj.Status.PodIP); err != nil {
		return err
	}

	for _, ps := range ns.podSelectors {
		if ps.matches(obj.ObjectMeta.Labels) {
			if err := ps.addEntry(obj.Status.PodIP); err != nil {
				return err
			}
		}
	}

	return nil
}

func (ns *ns) delFromMatching(obj *api.Pod) error {
	if err := ns.ipset.DelEntry(obj.Status.PodIP); err != nil {
		return err
	}

	for _, ps := range ns.podSelectors {
		if ps.matches(obj.ObjectMeta.Labels) {
			if err := ps.delEntry(obj.Status.PodIP); err != nil {
				return err
			}
		}
	}

	return nil
}

func hasIP(pod *api.Pod) bool {
	return len(pod.Status.PodIP) > 0
}

func equals(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for ak, av := range a {
		if b[ak] != av {
			return false
		}
	}
	return true
}
