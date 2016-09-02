package controller

import (
	"fmt"
	"github.com/weaveworks/weave-npc/pkg/ipset"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/types"
)

type selectorSet map[string]*selector

func newSelectorSet() selectorSet {
	return selectorSet(make(map[string]*selector))
}

type selector struct {
	json *unversioned.LabelSelector // JSON representation
	dom  labels.Selector            // k8s domain object
	str  string                     // string representation

	policies      map[types.UID]*extensions.NetworkPolicy // set of policies which depend on this selector
	namespaceName string                                  // for namespace scoped pod selectors
	ipsetTypeName string                                  // type of ipset to provision
	ipset         ipset.IPSet
}

func newSelector(json *unversioned.LabelSelector, ipsetName, ipsetTypeName string) (*selector, error) {
	dom, err := unversioned.LabelSelectorAsSelector(json)
	if err != nil {
		return nil, err
	}
	return &selector{
		json:          json,
		dom:           dom,
		str:           dom.String(),
		ipsetName:     ipsetName,
		ipsetTypeName: ipsetTypeName}, nil
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

func (s *selector) provision() error {
	if s.policies != nil {
		return fmt.Errorf("Selector already provisioned: %s", s.str)
	}

	s.policies = make(map[types.UID]*extensions.NetworkPolicy)
	// TODO needs to be prefixed with namespace name for pod selectors!
	s.ipset = ipset.New("weave-"+shortName(s.str), s.ipsetTypeName)

	return s.ipset.Create()
}

func (s *selector) deprovision() error {
	if s.policies == nil {
		return fmt.Errorf("Selector already deprovisioned: %s", s.str)
	}

	if len(s.policies) != 0 {
		return fmt.Errorf("Cannot deprovision in-use selector: %s", s.str)
	}

	defer func() {
		s.policies = nil
		s.ipset = nil
	}()

	return s.ipset.Destroy()
}
