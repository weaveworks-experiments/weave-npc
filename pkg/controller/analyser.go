package controller

import (
	"k8s.io/kubernetes/pkg/apis/extensions"
)

func analysePolicy(policy *extensions.NetworkPolicy) (nsSelectors, podSelectors selectorSet, err error) {
	nsSelectors = newSelectorSet()
	podSelectors = newSelectorSet()

	podSelector, err := newSelector(&policy.Spec.PodSelector)
	if err != nil {
		return nil, nil, err
	}
	podSelectors[podSelector.str] = podSelector

	for _, ingressRule := range policy.Spec.Ingress {
		if ingressRule.From != nil {
			for _, peer := range ingressRule.From {
				if peer.PodSelector != nil {
					podSelector, err := newSelector(peer.PodSelector)
					if err != nil {
						return nil, nil, err
					}
					podSelectors[podSelector.str] = podSelector
				}
				if peer.NamespaceSelector != nil {
					nsSelector, err := newSelector(peer.NamespaceSelector)
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
