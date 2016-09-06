package controller

import (
	"k8s.io/kubernetes/pkg/util/iptables"
	"strings"
)

type ruleResourceSpec struct {
	key  string
	args []string
}

type ruleResource struct {
	spec *ruleResourceSpec
}

type ruleResourceOps struct {
	ipt iptables.Interface
}

func NewRuleResourceOps(ipt iptables.Interface) ResourceOps {
	return &ruleResourceOps{ipt}
}

func NewRuleResourceSpec(proto *string, srcHost *selector, dstHost *selector, dstPort *string) ResourceSpec {
	args := []string{}
	if proto != nil {
		args = append(args, "-p", *proto)
	}
	if srcHost != nil {
		args = append(args, "-m", "set", "--match-set", srcHost.ipsetName, "src")
	}
	if dstHost != nil {
		args = append(args, "-m", "set", "--match-set", dstHost.ipsetName, "dst")
	}
	if dstPort != nil {
		args = append(args, "--dport", *dstPort)
	}
	args = append(args, "-j", "ACCEPT")
	key := strings.Join(args, " ")

	return &ruleResourceSpec{key, args}
}

func (rro *ruleResourceOps) Create(spec ResourceSpec) (Resource, error) {
	rrs := spec.(*ruleResourceSpec)
	_, err := rro.ipt.EnsureRule(iptables.Append, iptables.TableFilter, IngressChain, rrs.args...)
	return &ruleResource{rrs}, err
}

func (rro *ruleResourceOps) Destroy(resource Resource) error {
	rr := resource.(*ruleResource)
	return rro.ipt.DeleteRule(iptables.TableFilter, IngressChain, rr.spec.args...)
}

func (rr *ruleResource) Spec() ResourceSpec {
	return rr.spec
}

func (rrs *ruleResourceSpec) Key() ResourceKey {
	return ResourceKey(rrs.key)
}
