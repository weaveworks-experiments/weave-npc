package ipset

import (
	"github.com/pkg/errors"
	"os/exec"
)

type Type string

const (
	ListSet = Type("list:set")
	HashIP  = Type("hash:ip")
)

type Interface interface {
	Name() string
	Create() error
	AddEntry(entry string) error
	DelEntry(entry string) error
	Destroy() error
}

type ipset struct {
	name      string
	ipsetType Type
	entries   map[string]struct{}
}

func New(name string, ipsetType Type) Interface {
	return &ipset{name, ipsetType, make(map[string]struct{})}
}

func (i *ipset) Name() string {
	return i.name
}

func (i *ipset) Create() error {
	if _, err := exec.Command("ipset", "create", i.name, string(i.ipsetType)).Output(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return errors.Wrapf(err, "ipset create %s %s failed: %s", i.name, string(i.ipsetType), ee.Stderr)
		} else {
			return errors.Wrapf(err, "ipset create %s %s failed", i.name, string(i.ipsetType))
		}
	}
	return nil
}

func (i *ipset) AddEntry(entry string) error {
	if _, found := i.entries[entry]; !found {
		if err := exec.Command("ipset", "add", i.name, entry).Run(); err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				return errors.Wrapf(err, "ipset add %s %s failed: %s", i.name, entry, ee.Stderr)
			} else {
				return errors.Wrapf(err, "ipset add %s %s failed", i.name, entry)
			}
		}
		i.entries[entry] = struct{}{}
	}
	return nil
}

func (i *ipset) DelEntry(entry string) error {
	if _, found := i.entries[entry]; found {
		if err := exec.Command("ipset", "del", i.name, entry).Run(); err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				return errors.Wrapf(err, "ipset del %s %s failed: %s", i.name, entry, ee.Stderr)
			} else {
				return errors.Wrapf(err, "ipset del %s %s failed", i.name, entry)
			}
		}
		delete(i.entries, entry)
	}
	return nil
}

func (i *ipset) Destroy() error {
	if err := exec.Command("ipset", "destroy", i.name).Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return errors.Wrapf(err, "ipset destroy %s failed: %s", i.name, ee.Stderr)
		} else {
			return errors.Wrapf(err, "ipset destroy %s failed", i.name)
		}
	}
	return nil
}
