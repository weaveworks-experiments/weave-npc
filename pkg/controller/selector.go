package controller

import (
	"fmt"
	"github.com/weaveworks/weave-npc/pkg/util/ipset"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/types"
)

type selector struct {
	json *unversioned.LabelSelector // JSON representation (from API server)
	dom  labels.Selector            // k8s domain object (for matching)
	str  string                     // string representation (for hash keying/equality comparison)

	ipsetType ipset.Type // type of ipset to provision
	ipsetName ipset.Name // generated ipset name

	policies map[types.UID]struct{} // set of policies depending on this selector
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
		ipsetName: ipset.Name("weave-" + shortName(nsName+":"+str))}, nil
}

func (s *selector) matches(labelMap map[string]string) bool {
	return s.dom.Matches(labels.Set(labelMap))
}

func (s *selector) provision(ips ipset.Interface) error {
	if s.policies != nil {
		return fmt.Errorf("Selector already provisioned: %s", s.str)
	}

	s.policies = make(map[types.UID]struct{})

	return ips.Create(s.ipsetName, s.ipsetType)
}

func (s *selector) deprovision(ips ipset.Interface) error {
	if s.policies == nil {
		return fmt.Errorf("Selector already deprovisioned: %s", s.str)
	}

	if len(s.policies) != 0 {
		return fmt.Errorf("Cannot deprovision in-use selector: %s", s.str)
	}

	defer func() {
		s.policies = nil
	}()

	return ips.Destroy(s.ipsetName)
}

type selectorSet map[string]*selector

func newSelectorSet() selectorSet {
	return selectorSet(make(map[string]*selector))
}
