package controller

import (
	"github.com/weaveworks/weave-npc/pkg/ipset"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/types"
)

type podSelector struct {
	policies map[types.UID]struct{} // set of policies which utilise this selector
	selector labels.Selector        // k8s selector for matching pod labels
	ipset    ipset.HashIP           // hash:ip ipset of matching pod IPs
}

func newPodSelector(labelSelector labels.Selector) *podSelector {
	return &podSelector{
		policies: make(map[types.UID]struct{}),
		selector: labelSelector,
		ipset:    ipset.NewHashIP("meh")}
}

func (ps *podSelector) matches(labelMap map[string]string) bool {
	return ps.selector.Matches(labels.Set(labelMap))
}

func (ps *podSelector) addIP(ip string) error {
	return ps.ipset.AddIP(ip)
}

func (ps *podSelector) delIP(ip string) error {
	return ps.ipset.DelIP(ip)
}
