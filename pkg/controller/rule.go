package controller

import ()

type rule struct {
	proto   *string
	srcHost *selector
	dstHost *selector
	dstPort *string
}

func newRule(proto *string, srcHost *selector, dstHost *selector, dstPort *string) *rule {
	return &rule{proto, srcHost, dstHost, dstPort}
}

func (r *rule) provision() error {
	return nil
}

func (r *rule) deprovision() error {
	return nil
}
