package controller

import (
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/weaveworks/weave-npc/pkg/util/ipset"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/types"
	"k8s.io/kubernetes/pkg/util/iptables"
)

type ns struct {
	ipt iptables.Interface // interface to iptables
	ips ipset.Interface    // interface to ipset

	name      string                                  // k8s Namespace name
	namespace *api.Namespace                          // k8s Namespace object
	pods      map[types.UID]*api.Pod                  // k8s Pod objects by UID
	policies  map[types.UID]*extensions.NetworkPolicy // k8s NetworkPolicy objects by UID

	ipsetName ipset.Name // Name of hash:ip ipset storing pod IPs in this namespace

	nsSelectors  *selectorSet
	podSelectors *selectorSet
	rules        *ruleSet
}

func newNS(name string, ipt iptables.Interface, ips ipset.Interface, nsSelectors *selectorSet) (*ns, error) {
	ipsetName := ipset.Name("weave-" + shortName(name))
	if err := ips.Create(ipsetName, ipset.HashIP); err != nil {
		return nil, err
	}

	n := &ns{
		ipt:         ipt,
		ips:         ips,
		name:        name,
		pods:        make(map[types.UID]*api.Pod),
		policies:    make(map[types.UID]*extensions.NetworkPolicy),
		ipsetName:   ipsetName,
		nsSelectors: nsSelectors,
		rules:       newRuleSet(ipt)}

	n.podSelectors = newSelectorSet(ips, n.onNewPodSelector)

	return n, nil
}

func (ns *ns) empty() bool {
	return len(ns.pods) == 0 && len(ns.policies) == 0 && ns.namespace == nil
}

func (ns *ns) destroy() error {
	if err := ns.ips.Destroy(ns.ipsetName); err != nil {
		return err
	}
	return nil
}

func (ns *ns) onNewPodSelector(selector *selector) error {
	for _, pod := range ns.pods {
		if hasIP(pod) {
			if selector.matches(pod.ObjectMeta.Labels) {
				if err := ns.ips.AddEntry(selector.spec.ipsetName, pod.Status.PodIP); err != nil {
					return err
				}
			}
		}
	}
	return nil
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

		for _, ps := range ns.podSelectors.entries {
			oldMatch := ps.matches(oldObj.ObjectMeta.Labels)
			newMatch := ps.matches(newObj.ObjectMeta.Labels)
			if oldMatch == newMatch && oldObj.Status.PodIP == newObj.Status.PodIP {
				continue
			}
			if oldMatch {
				if err := ns.ips.DelEntry(ps.spec.ipsetName, oldObj.Status.PodIP); err != nil {
					return err
				}
			}
			if newMatch {
				if err := ns.ips.AddEntry(ps.spec.ipsetName, newObj.Status.PodIP); err != nil {
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
	if err := ns.nsSelectors.ProvisionNew(obj.ObjectMeta.UID, nil, nsSelectors); err != nil {
		return err
	}

	// Provision any missing pod selector ipsets; reference existing
	if err := ns.podSelectors.ProvisionNew(obj.ObjectMeta.UID, nil, podSelectors); err != nil {
		return err
	}

	// Reference iptables rules, creating if necessary
	if err := ns.rules.ProvisionNew(obj.ObjectMeta.UID, nil, rules); err != nil {
		return err
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

	// Deprovision unused rules
	if err := ns.rules.DeprovisionUnused(oldObj.ObjectMeta.UID, oldRules, newRules); err != nil {
		return err
	}

	// Deprovision namespace selector ipsets that are no longer in use
	if err := ns.nsSelectors.DeprovisionUnused(oldObj.ObjectMeta.UID, oldNsSelectors, newNsSelectors); err != nil {
		return err
	}

	// Deprovision pod selector ipsets that are no longer in use
	if err := ns.podSelectors.DeprovisionUnused(oldObj.ObjectMeta.UID, oldPodSelectors, newPodSelectors); err != nil {
		return err
	}

	// Provision any missing namespace selector ipsets; reference existing
	if err := ns.nsSelectors.ProvisionNew(oldObj.ObjectMeta.UID, oldNsSelectors, newNsSelectors); err != nil {
		return err
	}

	// Provision any missing pod selector ipsets; reference existing
	if err := ns.podSelectors.ProvisionNew(oldObj.ObjectMeta.UID, oldPodSelectors, newNsSelectors); err != nil {
		return err
	}

	// Reference iptables rules, creating if necessary
	if err := ns.rules.ProvisionNew(oldObj.ObjectMeta.UID, oldRules, newRules); err != nil {
		return err
	}

	return nil
}

func (ns *ns) deleteNetworkPolicy(obj *extensions.NetworkPolicy) error {
	delete(ns.policies, obj.ObjectMeta.UID)

	// Analyse network policy to free resources
	rules, nsSelectors, podSelectors, err := ns.analysePolicy(obj)
	if err != nil {
		return err
	}

	// Deprovision unused rules
	if err := ns.rules.DeprovisionUnused(obj.ObjectMeta.UID, rules, nil); err != nil {
		return err
	}

	// Deprovision namespace selector ipsets that are no longer in use
	if err := ns.nsSelectors.DeprovisionUnused(obj.ObjectMeta.UID, nsSelectors, nil); err != nil {
		return err
	}

	// Deprovision pod selector ipsets that are no longer in use
	if err := ns.podSelectors.DeprovisionUnused(obj.ObjectMeta.UID, podSelectors, nil); err != nil {
		return err
	}

	return nil
}

func (ns *ns) addNamespace(obj *api.Namespace) error {
	ns.namespace = obj

	// Insert a rule to bypass policies if namespace is DefaultAllow
	if !isDefaultDeny(obj) {
		if _, err := ns.ipt.EnsureRule(iptables.Append, iptables.TableFilter, DefaultChain,
			"-m", "set", "--match-set", string(ns.ipsetName), "dst", "-j", "ACCEPT"); err != nil {
			return err
		}
	}

	// Add namespace ipset to matching namespace selectors
	for _, selector := range ns.nsSelectors.entries {
		if selector.matches(obj.ObjectMeta.Labels) {
			if err := ns.ips.AddEntry(selector.spec.ipsetName, string(ns.ipsetName)); err != nil {
				return err
			}
		}
	}

	return nil
}

func (ns *ns) updateNamespace(oldObj, newObj *api.Namespace) error {
	ns.namespace = newObj

	// Update bypass rule if ingress default has changed
	oldDefaultDeny := isDefaultDeny(oldObj)
	newDefaultDeny := isDefaultDeny(newObj)

	if oldDefaultDeny != newDefaultDeny {
		if oldDefaultDeny {
			if _, err := ns.ipt.EnsureRule(iptables.Append, iptables.TableFilter, DefaultChain,
				"-m", "set", "--match-set", string(ns.ipsetName), "dst", "-j", "ACCEPT"); err != nil {
				return err
			}
		}
		if newDefaultDeny {
			if err := ns.ipt.DeleteRule(iptables.TableFilter, DefaultChain,
				"-m", "set", "--match-set", string(ns.ipsetName), "dst", "-j", "ACCEPT"); err != nil {
				return err
			}
		}
	}

	// Re-evaluate namespace selector membership if labels have changed
	if !equals(oldObj.ObjectMeta.Labels, newObj.ObjectMeta.Labels) {
		for _, selector := range ns.nsSelectors.entries {
			oldMatch := selector.matches(oldObj.ObjectMeta.Labels)
			newMatch := selector.matches(newObj.ObjectMeta.Labels)
			if oldMatch == newMatch {
				continue
			}
			if oldMatch {
				if err := ns.ips.DelEntry(selector.spec.ipsetName, string(ns.ipsetName)); err != nil {
					return err
				}
			}
			if newMatch {
				if err := ns.ips.AddEntry(selector.spec.ipsetName, string(ns.ipsetName)); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (ns *ns) deleteNamespace(obj *api.Namespace) error {
	ns.namespace = nil

	// Remove namespace ipset from any matching namespace selectors
	for _, selector := range ns.nsSelectors.entries {
		if selector.matches(obj.ObjectMeta.Labels) {
			if err := ns.ips.DelEntry(selector.spec.ipsetName, string(ns.ipsetName)); err != nil {
				return err
			}
		}
	}

	// Remove bypass rule
	if !isDefaultDeny(obj) {
		if err := ns.ipt.DeleteRule(iptables.TableFilter, DefaultChain,
			"-m", "set", "--match-set", string(ns.ipsetName), "dst", "-j", "ACCEPT"); err != nil {
			return err
		}
	}

	return nil
}

func (ns *ns) addToMatching(obj *api.Pod) error {
	if err := ns.ips.AddEntry(ns.ipsetName, obj.Status.PodIP); err != nil {
		return errors.Wrap(err, "addToMatching")
	}

	for _, ps := range ns.podSelectors.entries {
		if ps.matches(obj.ObjectMeta.Labels) {
			if err := ns.ips.AddEntry(ps.spec.ipsetName, obj.Status.PodIP); err != nil {
				return err
			}
		}
	}

	return nil
}

func (ns *ns) delFromMatching(obj *api.Pod) error {
	if err := ns.ips.DelEntry(ns.ipsetName, obj.Status.PodIP); err != nil {
		return err
	}

	for _, ps := range ns.podSelectors.entries {
		if ps.matches(obj.ObjectMeta.Labels) {
			if err := ns.ips.DelEntry(ps.spec.ipsetName, obj.Status.PodIP); err != nil {
				return err
			}
		}
	}

	return nil
}

func hasIP(pod *api.Pod) bool {
	// Ensure pod has an IP address and isn't sharing the host network namespace
	return len(pod.Status.PodIP) > 0 &&
		!(pod.Spec.SecurityContext != nil &&
			pod.Spec.SecurityContext.HostNetwork)
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

func isDefaultDeny(namespace *api.Namespace) bool {
	nnpJson, found := namespace.ObjectMeta.Annotations["net.beta.kubernetes.io/network-policy"]
	if !found {
		return false
	}

	var nnp NamespaceNetworkPolicy
	if err := json.Unmarshal([]byte(nnpJson), &nnp); err != nil {
		// If we can't understand the annotation, behave as if it isn't present
		// TODO log unmarshal failure
		return false
	}

	return nnp.Ingress != nil &&
		nnp.Ingress.Isolation != nil &&
		*(nnp.Ingress.Isolation) == DefaultDeny
}
