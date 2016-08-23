package controller

import (
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/util/intstr"
)

func analysePolicy(policy *extensions.NetworkPolicy) (nsSelectors, podSelectors selectorSet, rules []*rule, err error) {
	nsSelectors = newSelectorSet()
	podSelectors = newSelectorSet()
	rules = make([]*rule, 0)

	dstSelector, err := newSelector(&policy.Spec.PodSelector)
	if err != nil {
		return nil, nil, nil, err
	}
	podSelectors[dstSelector.str] = dstSelector

	for _, ingressRule := range policy.Spec.Ingress {
		// If Ports is present but empty, this rule matches no traffic
		if ingressRule.Ports != nil && len(ingressRule.Ports) == 0 {
			continue
		}
		if ingressRule.From != nil {
			for _, peer := range ingressRule.From {
				var srcSelector *selector
				if peer.PodSelector != nil {
					srcSelector, err := newSelector(peer.PodSelector)
					if err != nil {
						return nil, nil, nil, err
					}
					podSelectors[srcSelector.str] = srcSelector
				}
				if peer.NamespaceSelector != nil {
					srcSelector, err := newSelector(peer.NamespaceSelector)
					if err != nil {
						return nil, nil, nil, err
					}
					nsSelectors[srcSelector.str] = srcSelector
				}

				if ingressRule.Ports == nil {
					// Traffic is not restricted by proto/port
					rules = append(rules, newRule(nil, srcSelector, dstSelector, nil))
				} else {
					// Traffic is restricted by proto/port
					for _, npp := range ingressRule.Ports {
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
								port = string(npp.Port.IntVal)
							case intstr.String:
								port = npp.Port.StrVal
							}
						}

						rules = append(rules, newRule(&proto, srcSelector, dstSelector, &port))
					}
				}
			}
		}
	}

	return nsSelectors, podSelectors, rules, nil
}
