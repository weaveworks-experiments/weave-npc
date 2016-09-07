package controller

import (
	"fmt"
	"github.com/weaveworks/weave-npc/pkg/util/ipset"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/util/intstr"
)

func (ns *ns) analysePolicy(policy *extensions.NetworkPolicy) (rules map[ResourceKey]ResourceSpec, nsSelectors, podSelectors map[string]*selector, err error) {
	nsSelectors = make(map[string]*selector)
	podSelectors = make(map[string]*selector)
	rules = make(map[ResourceKey]ResourceSpec)

	dstSelector, err := newSelector(&policy.Spec.PodSelector, ns.name, ipset.HashIP)
	if err != nil {
		return nil, nil, nil, err
	}
	podSelectors[dstSelector.str] = dstSelector

	for _, ingressRule := range policy.Spec.Ingress {
		if ingressRule.Ports != nil && len(ingressRule.Ports) == 0 {
			// Ports is empty, this rule matches no ports (no traffic matches).
			continue
		}

		if ingressRule.From != nil && len(ingressRule.From) == 0 {
			// From is empty, this rule matches no sources (no traffic matches).
			continue
		}

		if ingressRule.From == nil {
			// From is not provided, this rule matches all sources (traffic not restricted by source).
			if ingressRule.Ports == nil {
				// Ports is not provided, this rule matches all ports (traffic not restricted by port).
				rule := NewRuleResourceSpec(nil, nil, dstSelector, nil)
				rules[rule.Key()] = rule
			} else {
				// Ports is present and contains at least one item, then this rule allows traffic
				// only if the traffic matches at least one port in the ports list.
				withNormalisedProtoAndPort(ingressRule.Ports, func(proto, port string) {
					rule := NewRuleResourceSpec(&proto, nil, dstSelector, &port)
					rules[rule.Key()] = rule
				})
			}
		} else {
			// From is present and contains at least on item, this rule allows traffic only if the
			// traffic matches at least one item in the from list.
			for _, peer := range ingressRule.From {
				var srcSelector *selector
				if peer.PodSelector != nil {
					srcSelector, err = newSelector(peer.PodSelector, ns.name, "hash:ip")
					if err != nil {
						return nil, nil, nil, err
					}
					podSelectors[srcSelector.str] = srcSelector
				}
				if peer.NamespaceSelector != nil {
					srcSelector, err = newSelector(peer.NamespaceSelector, "", ipset.ListSet)
					if err != nil {
						return nil, nil, nil, err
					}
					nsSelectors[srcSelector.str] = srcSelector
				}

				if ingressRule.Ports == nil {
					// Ports is not provided, this rule matches all ports (traffic not restricted by port).
					rule := NewRuleResourceSpec(nil, srcSelector, dstSelector, nil)
					rules[rule.Key()] = rule
				} else {
					// Ports is present and contains at least one item, then this rule allows traffic
					// only if the traffic matches at least one port in the ports list.
					withNormalisedProtoAndPort(ingressRule.Ports, func(proto, port string) {
						rule := NewRuleResourceSpec(&proto, srcSelector, dstSelector, &port)
						rules[rule.Key()] = rule
					})
				}
			}
		}
	}

	return rules, nsSelectors, podSelectors, nil
}

func withNormalisedProtoAndPort(npps []extensions.NetworkPolicyPort, f func(proto, port string)) {
	for _, npp := range npps {
		// If no proto is specified, default to TCP
		proto := string(api.ProtocolTCP)
		if npp.Protocol != nil {
			proto = string(*npp.Protocol)
		}

		// If no port is specified, match any port. Let iptables executable handle
		// service name resolution
		port := "0:65535"
		if npp.Port != nil {
			switch npp.Port.Type {
			case intstr.Int:
				port = fmt.Sprintf("%d", npp.Port.IntVal)
			case intstr.String:
				port = npp.Port.StrVal
			}
		}

		f(proto, port)
	}
}
