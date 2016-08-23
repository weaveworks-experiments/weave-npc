package ipset

import (
	"github.com/pkg/errors"
	"os/exec"
)

type IPSet interface {
	Name() string
	Create() error
	AddEntry(entry string) error
	DelEntry(entry string) error
	Destroy() error
}

type ipset struct {
	name     string
	typeName string
	entries  map[string]struct{}
}

func New(name, typeName string) IPSet {
	return &ipset{name, typeName, make(map[string]struct{})}
}

func (i *ipset) Name() string {
	return i.name
}

func (i *ipset) TypeName() string {
	return i.typeName
}

func (i *ipset) Create() error {
	if _, err := exec.Command("ipset", "create", i.name, i.typeName).Output(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return errors.Wrapf(err, "ipset create %s %s failed: %s", i.name, i.typeName, ee.Stderr)
		} else {
			return errors.Wrapf(err, "ipset create %s %s failed", i.name, i.typeName)
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
