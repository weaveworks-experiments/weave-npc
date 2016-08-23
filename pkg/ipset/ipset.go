package ipset

import (
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
	if err := exec.Command("/usr/sbin/ipset", "create", i.name, i.typeName).Run(); err != nil {
		return err
	}
	return nil
}

func (i *ipset) AddEntry(entry string) error {
	if _, found := i.entries[entry]; !found {
		if err := exec.Command("/usr/sbin/ipset", "add", i.name, entry).Run(); err != nil {
			return err
		}
		i.entries[entry] = struct{}{}
	}
	return nil
}

func (i *ipset) DelEntry(entry string) error {
	if _, found := i.entries[entry]; found {
		if err := exec.Command("/usr/sbin/ipset", "del", i.name, entry).Run(); err != nil {
			return err
		}
		delete(i.entries, entry)
	}
	return nil
}

func (i *ipset) Destroy() error {
	if err := exec.Command("/usr/sbin/ipset", "destroy", i.name).Run(); err != nil {
		return err
	}
	return nil
}
