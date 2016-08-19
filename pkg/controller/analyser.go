package controller

import (
	"k8s.io/kubernetes/pkg/apis/extensions"
)

func AnalysePolicy(policy *extensions.NetworkPolicy) (nsSelectors, podSelectors map[string]*selector, err error) {
	nsSelectors = make(map[string]*selector)
	podSelectors = make(map[string]*selector)

	podSelector, err := NewSelector(&policy.Spec.PodSelector)
	if err != nil {
		return nil, nil, err
	}
	podSelectors[podSelector.str] = podSelector

	for _, ingressRule := range policy.Spec.Ingress {
		if ingressRule.From != nil {
			for _, peer := range ingressRule.From {
				if peer.PodSelector != nil {
					podSelector, err := NewSelector(peer.PodSelector)
					if err != nil {
						return nil, nil, err
					}
					podSelectors[podSelector.str] = podSelector
				}
				if peer.NamespaceSelector != nil {
					nsSelector, err := NewSelector(peer.NamespaceSelector)
					if err != nil {
						return nil, nil, err
					}
					nsSelectors[nsSelector.str] = nsSelector
				}
			}
		}
	}

	return nsSelectors, podSelectors, nil
}
