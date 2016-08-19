package controller

import (
	"github.com/weaveworks/weave-npc/pkg/ipset"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/types"
)

type selector struct {
	json     *unversioned.LabelSelector // JSON representation
	dom      labels.Selector            // k8s domain object
	str      string                     // string representation
	policies map[types.UID]struct{}     // set of policies which depend on this selector
	ipset    ipset.IPSet
}

func newSelector(json *unversioned.LabelSelector) (*selector, error) {
	dom, err := unversioned.LabelSelectorAsSelector(json)
	if err != nil {
		return nil, err
	}
	return &selector{
		json: json,
		dom:  dom,
		str:  dom.String()}, nil
}

func (s *selector) matches(labelMap map[string]string) bool {
	return s.dom.Matches(labels.Set(labelMap))
}

func (s *selector) addEntry(name string) error {
	return s.ipset.AddEntry(name)
}

func (s *selector) delEntry(name string) error {
	return s.ipset.DelEntry(name)
}

func (s *selector) realise() {
	if s.policies == nil {
		s.policies = make(map[types.UID]struct{})
	}
}
