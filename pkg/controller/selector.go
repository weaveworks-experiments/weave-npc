package controller

import (
	"github.com/weaveworks/weave-npc/pkg/util/ipset"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/types"
)

type selectorSpec struct {
	json *unversioned.LabelSelector // JSON representation (from API server)
	dom  labels.Selector            // k8s domain object (for matching)
	key  string                     // string representation (for hash keying/equality comparison)

	ipsetType ipset.Type // type of ipset to provision
	ipsetName ipset.Name // generated ipset name
}

func newSelectorSpec(json *unversioned.LabelSelector, nsName string, ipsetType ipset.Type) (*selectorSpec, error) {
	dom, err := unversioned.LabelSelectorAsSelector(json)
	if err != nil {
		return nil, err
	}
	key := dom.String()
	return &selectorSpec{
		json:      json,
		ipsetType: ipsetType,
		dom:       dom,
		key:       key,
		// We prefix the selector string with the namespace name when generating
		// the shortname because you can specify the same selector in multiple
		// namespaces - we need those to map to distinct ipsets
		ipsetName: ipset.Name("weave-" + shortName(nsName+":"+key))}, nil
}

type selector struct {
	spec *selectorSpec
}

func (s *selector) matches(labelMap map[string]string) bool {
	return s.spec.dom.Matches(labels.Set(labelMap))
}

type selectorFn func(selector *selector) error

type selectorSet struct {
	ips           ipset.Interface
	onNewSelector selectorFn
	users         map[string]map[types.UID]struct{} // list of users per selector
	entries       map[string]*selector
}

func newSelectorSet(ips ipset.Interface, onNewSelector selectorFn) *selectorSet {
	return &selectorSet{
		ips:           ips,
		onNewSelector: onNewSelector,
		users:         make(map[string]map[types.UID]struct{}),
		entries:       make(map[string]*selector)}
}

func (ss *selectorSet) DeprovisionUnused(user types.UID, current, desired map[string]*selectorSpec) error {
	for key, spec := range current {
		if _, found := desired[key]; !found {
			delete(ss.users[key], user)
			if len(ss.users[key]) == 0 {
				if err := ss.ips.Destroy(spec.ipsetName); err != nil {
					return err
				}
				delete(ss.entries, key)
				delete(ss.users, key)
			}
		}
	}
	return nil
}

func (ss *selectorSet) ProvisionNew(user types.UID, current, desired map[string]*selectorSpec) error {
	for key, spec := range desired {
		if _, found := current[key]; !found {
			if _, found := ss.users[key]; !found {
				if err := ss.ips.Create(spec.ipsetName, spec.ipsetType); err != nil {
					return err
				}
				selector := &selector{spec}
				if err := ss.onNewSelector(selector); err != nil {
					return err
				}
				ss.users[key] = make(map[types.UID]struct{})
				ss.entries[key] = selector
			}
			ss.users[key][user] = struct{}{}
		}
	}
	return nil
}
