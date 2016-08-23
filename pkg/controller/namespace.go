package controller

import (
	"github.com/pkg/errors"
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
	podSelectors selectorSet                             // selector string -> podSelector
	nss          map[string]*ns                          // ns name -> ns struct
	nsSelectors  selectorSet                             // selector string -> nsSelector
}

func newNS(name string, nss map[string]*ns, nsSelectors selectorSet) (*ns, error) {
	ipset := ipset.New(shortName(name), "hash:ip")
	if err := ipset.Create(); err != nil {
		return nil, err
	}
	return &ns{
		name:         name,
		pods:         make(map[types.UID]*api.Pod),
		policies:     make(map[types.UID]*extensions.NetworkPolicy),
		ipset:        ipset,
		podSelectors: newSelectorSet(),
		nss:          nss,
		nsSelectors:  nsSelectors}, nil
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
	ns.policies[obj.ObjectMeta.UID] = obj

	// Analyse policy, determine which rules and ipsets are required
	rules, nsSelectors, podSelectors, err := ns.analysePolicy(obj)
	if err != nil {
		return err
	}

	// Provision any missing namespace selector ipsets; reference existing
	for selectorKey, selector := range nsSelectors {
		if existingSelector, found := ns.nsSelectors[selectorKey]; found {
			existingSelector.policies[obj.ObjectMeta.UID] = obj
		} else {
			if err := selector.provision(); err != nil {
				return err
			}

			selector.policies[obj.ObjectMeta.UID] = obj

			for _, otherNs := range ns.nss {
				if otherNs.namespace != nil {
					if selector.matches(otherNs.namespace.ObjectMeta.Labels) {
						if err := selector.addEntry(otherNs.ipset.Name()); err != nil {
							return err
						}
					}
				}
			}
		}
	}

	// Provision any missing pod selector ipsets; reference existing
	for selectorKey, selector := range podSelectors {
		if existingSelector, found := ns.podSelectors[selectorKey]; found {
			existingSelector.policies[obj.ObjectMeta.UID] = obj
		} else {
			if err := selector.provision(); err != nil {
				return err
			}

			selector.policies[obj.ObjectMeta.UID] = obj

			for _, pod := range ns.pods {
				if hasIP(pod) {
					if selector.matches(pod.ObjectMeta.Labels) {
						if err := selector.addEntry(pod.Status.PodIP); err != nil {
							return err
						}
					}
				}
			}
		}
	}

	// No need to reference count rules - iptables permits duplicates
	for _, rule := range rules {
		if err := rule.provision(); err != nil {
			return err
		}
	}

	return nil
}

func (ns *ns) updateNetworkPolicy(oldObj, newObj *extensions.NetworkPolicy) error {
	delete(ns.policies, oldObj.ObjectMeta.UID)
	ns.policies[newObj.ObjectMeta.UID] = newObj

	// Analyse the old and the new policy so we can determine differences
	oldRules, oldNsSelectors, oldPodSelectors, err := ns.analysePolicy(oldObj)
	if err != nil {
		return err
	}
	newRules, newNsSelectors, newPodSelectors, err := ns.analysePolicy(newObj)
	if err != nil {
		return err
	}

	{
		// Handle namespace selector changes. Deprovision selector ipsets we no
		// longer use, and create any new ones we require
		for key, _ := range oldNsSelectors {
			selector := ns.nsSelectors[key]
			if _, found := newNsSelectors[key]; found {
				// Object UIDs should not change, but handle it anyway
				delete(selector.policies, oldObj.ObjectMeta.UID)
				selector.policies[newObj.ObjectMeta.UID] = newObj
			} else {
				delete(selector.policies, oldObj.ObjectMeta.UID)
				if len(selector.policies) == 0 {
					if err := selector.deprovision(); err != nil {
						return err
					}
					delete(ns.nsSelectors, key)
				}
			}
		}

		for key, selector := range newNsSelectors {
			if _, found := ns.nsSelectors[key]; !found {
				if err := selector.provision(); err != nil {
					return err
				}

				selector.policies[newObj.ObjectMeta.UID] = newObj

				for _, otherNs := range ns.nss {
					if otherNs.namespace != nil {
						if selector.matches(otherNs.namespace.ObjectMeta.Labels) {
							if err := selector.addEntry(otherNs.ipset.Name()); err != nil {
								return err
							}
						}
					}
				}
			}

		}
	}

	{
		// Handle pod selector changes. Deprovision selector ipsets we no
		// longer use, and create any new ones we require
		for key, _ := range oldPodSelectors {
			selector := ns.podSelectors[key]
			if _, found := newPodSelectors[key]; found {
				// Object UIDs should not change, but handle it anyway
				delete(selector.policies, oldObj.ObjectMeta.UID)
				selector.policies[newObj.ObjectMeta.UID] = newObj
			} else {
				delete(selector.policies, oldObj.ObjectMeta.UID)
				if len(selector.policies) == 0 {
					if err := selector.deprovision(); err != nil {
						return err
					}
					delete(ns.podSelectors, key)
				}
			}
		}

		for key, selector := range newPodSelectors {
			if _, found := ns.podSelectors[key]; !found {
				if err := selector.provision(); err != nil {
					return err
				}

				selector.policies[newObj.ObjectMeta.UID] = newObj

				for _, pod := range ns.pods {
					if hasIP(pod) {
						if selector.matches(pod.ObjectMeta.Labels) {
							if err := selector.addEntry(pod.Status.PodIP); err != nil {
								return err
							}
						}
					}
				}
			}

		}
	}

	// Take advantage of iptables behaviour to avoid diffing/reference counting rules
	for _, rule := range newRules {
		if err := rule.provision(); err != nil {
			return err
		}
	}
	for _, rule := range oldRules {
		if err := rule.deprovision(); err != nil {
			return err
		}
	}

	return nil
}

func (ns *ns) deleteNetworkPolicy(obj *extensions.NetworkPolicy) error {
	delete(ns.policies, obj.ObjectMeta.UID)

	// Analyse the old and the new policy so we can determine differences
	rules, nsSelectors, podSelectors, err := ns.analysePolicy(obj)
	if err != nil {
		return err
	}

	// Remove rules first, so that ipsets are freed. No need to reference count
	// rules - iptables deletion removes duplicated rules one at a time
	for _, rule := range rules {
		if err := rule.deprovision(); err != nil {
			return err
		}
	}

	// Deprovision pod selector ipsets that are no longer in use
	for key, selector := range podSelectors {
		delete(selector.policies, obj.ObjectMeta.UID)
		if len(selector.policies) == 0 {
			if err := selector.deprovision(); err != nil {
				return err
			}
			delete(ns.podSelectors, key)
		}
	}

	// Deprovision namespace selector ipsets that are no longer in use
	for key, selector := range nsSelectors {
		delete(selector.policies, obj.ObjectMeta.UID)
		if len(selector.policies) == 0 {
			if err := selector.deprovision(); err != nil {
				return err
			}
			delete(ns.nsSelectors, key)
		}
	}

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
		return errors.Wrap(err, "addToMatching")
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
