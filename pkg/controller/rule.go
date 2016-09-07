package controller

import (
	"k8s.io/kubernetes/pkg/types"
	"k8s.io/kubernetes/pkg/util/iptables"
	"strings"
)

type ruleSpec struct {
	key  string
	args []string
}

func newRuleSpec(proto *string, srcHost *selectorSpec, dstHost *selectorSpec, dstPort *string) *ruleSpec {
	args := []string{}
	if proto != nil {
		args = append(args, "-p", *proto)
	}
	if srcHost != nil {
		args = append(args, "-m", "set", "--match-set", string(srcHost.ipsetName), "src")
	}
	if dstHost != nil {
		args = append(args, "-m", "set", "--match-set", string(dstHost.ipsetName), "dst")
	}
	if dstPort != nil {
		args = append(args, "--dport", *dstPort)
	}
	args = append(args, "-j", "ACCEPT")
	key := strings.Join(args, " ")

	return &ruleSpec{key, args}
}

type ruleSet struct {
	ipt   iptables.Interface
	users map[string]map[types.UID]struct{}
}

func newRuleSet(ipt iptables.Interface) *ruleSet {
	return &ruleSet{ipt, make(map[string]map[types.UID]struct{})}
}

func (rs *ruleSet) DeprovisionUnused(user types.UID, current, desired map[string]*ruleSpec) error {
	for key, spec := range current {
		if _, found := desired[key]; !found {
			delete(rs.users[key], user)
			if len(rs.users[key]) == 0 {
				if err := rs.ipt.DeleteRule(iptables.TableFilter, IngressChain, spec.args...); err != nil {
					return err
				}
				delete(rs.users, key)
			}
		}
	}

	return nil
}

func (rs *ruleSet) ProvisionNew(user types.UID, current, desired map[string]*ruleSpec) error {
	for key, spec := range desired {
		if _, found := current[key]; !found {
			if _, found := rs.users[key]; !found {
				_, err := rs.ipt.EnsureRule(iptables.Append, iptables.TableFilter, IngressChain, spec.args...)
				if err != nil {
					return err
				}
				rs.users[key] = make(map[types.UID]struct{})
			}
			rs.users[key][user] = struct{}{}
		}
	}

	return nil
}
