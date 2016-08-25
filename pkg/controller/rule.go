package controller

import (
	"k8s.io/kubernetes/pkg/util/iptables"
)

const (
	WeaveChain = iptables.Chain("WEAVE-NPC")
)

type rule struct {
	proto   *string
	srcHost *selector
	dstHost *selector
	dstPort *string
}

func newRule(proto *string, srcHost *selector, dstHost *selector, dstPort *string) *rule {
	return &rule{proto, srcHost, dstHost, dstPort}
}

func (r *rule) provision(ipt iptables.Interface) error {
	_, err := ipt.EnsureRule(iptables.Append, iptables.TableFilter, WeaveChain, r.args()...)
	return err
}

func (r *rule) deprovision(ipt iptables.Interface) error {
	return ipt.DeleteRule(iptables.TableFilter, WeaveChain, r.args()...)
}

func (r *rule) args() []string {
	args := []string{}
	if r.proto != nil {
		args = append(args, "-p", *r.proto)
	}
	if r.srcHost != nil {
		args = append(args, "-m", "set", "--match-set", r.srcHost.ipset.Name(), "src")
	}
	if r.dstHost != nil {
		args = append(args, "-m", "set", "--match-set", r.dstHost.ipset.Name(), "dst")
	}
	if r.dstPort != nil {
		args = append(args, "--dport", *r.dstPort)
	}
	return append(args, "-m", "state", "--state", "NEW", "-j", "ACCEPT")
}
