package controller

import (
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/types"
)

type selector struct {
	json     *unversioned.LabelSelector // JSON representation
	dom      labels.Selector            // k8s domain object
	str      string                     // string representation
	policies map[types.UID]struct{}     // set of policies which depend on this selector
}

func NewSelector(json *unversioned.LabelSelector) (*selector, error) {
	dom, err := unversioned.LabelSelectorAsSelector(json)
	if err != nil {
		return nil, err
	}
	return &selector{
		json: json,
		dom:  dom,
		str:  dom.String()}, nil
}

func (s *selector) realise() {
	if s.policies == nil {
		s.policies = make(map[types.UID]struct{})
	}
}
