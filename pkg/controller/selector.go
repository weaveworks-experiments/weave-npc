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
	json      *unversioned.LabelSelector // JSON representation (from API server)
	ipsetType ipset.Type                 // type of ipset to provision

	dom       labels.Selector                         // k8s domain object (for matching)
	str       string                                  // string representation (for hash keying/equality comparison)
	policies  map[types.UID]*extensions.NetworkPolicy // set of policies which depend on this selector
	ipsetName string                                  // generated ipset name
	ipset     ipset.Interface                         // concrete ipset
}

func newSelector(json *unversioned.LabelSelector, nsName string, ipsetType ipset.Type) (*selector, error) {
	dom, err := unversioned.LabelSelectorAsSelector(json)
	if err != nil {
		return nil, err
	}
	str := dom.String()
	return &selector{
		json:      json,
		ipsetType: ipsetType,
		dom:       dom,
		str:       str,
		// We prefix the selector string with the namespace name when generating
		// the shortname because you can specify the same selector in multiple
		// namespaces - we need those to map to distinct ipsets
		ipsetName: "weave-" + shortName(nsName+":"+str)}, nil
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
	s.ipset = ipset.New(s.ipsetName, s.ipsetType)

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
