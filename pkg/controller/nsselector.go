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
