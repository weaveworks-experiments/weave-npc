package controller

import (
	"k8s.io/kubernetes/pkg/util/iptables"
)

const (
	MainChain    = iptables.Chain("WEAVE-NPC")
	DefaultChain = iptables.Chain("WEAVE-NPC-DEFAULT")
	IngressChain = iptables.Chain("WEAVE-NPC-INGRESS")
)
