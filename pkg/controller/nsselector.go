package controller

import (
	"github.com/weaveworks/weave-npc/pkg/ipset"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/types"
)

type nsSelector struct {
	policies map[types.UID]struct{} // policies which reference this selector
	selector labels.Selector        // k8s selector for matching namespace labels
	ipset    ipset.ListSet          // list:set ipset of matching namespace hash:ip ipsets
}

func newNSSelector(labelSelector labels.Selector) *nsSelector {
	return &nsSelector{
		policies: make(map[types.UID]struct{}),
		selector: labelSelector,
		ipset:    ipset.NewListSet("meh")}
}

func (nss *nsSelector) matches(labels map[string]string) bool {
	return false
}

func (ns *nsSelector) addList(name string) error {
	return ns.ipset.AddList(name)
}

func (ns *nsSelector) delList(name string) error {
	return ns.ipset.DelList(name)
}
