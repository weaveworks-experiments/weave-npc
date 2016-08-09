package controller

import (
	"github.com/weaveworks/weave-npc/pkg/ipset"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/types"
)

type podSelector struct {
	policies map[types.UID]struct{} // set of policies which utilise this selector
	selector labels.Selector        // k8s selector for matching pod labels
	ipset    ipset.HashIP           // hash:ip ipset of matching pod IPs
}

func newPodSelector() *podSelector {
	return &podSelector{
		policies: make(map[types.UID]struct{}),
		ipset:    ipset.NewHashIP("meh")}
}

func (ps *podSelector) addPodIP(pod *api.Pod) error {
	if len(pod.Status.PodIP) == 0 {
		return nil
	}
	if err := ps.ipset.AddIP(pod.Status.PodIP); err != nil {
		return err
	}
	return nil
}

func (ps *podSelector) delPodIP(pod *api.Pod) error {
	if len(pod.Status.PodIP) == 0 {
		return nil
	}
	if err := ps.ipset.DelIP(pod.Status.PodIP); err != nil {
		return err
	}
	return nil
}
