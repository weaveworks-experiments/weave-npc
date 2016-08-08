package ipset

import ()

type ListSet interface {
	Name() string
	AddList(list string) error
	DelList(list string) error
	Count() int
}

type listSet struct {
	name  string
	lists map[string]struct{}
}

func NewListSet(name string) ListSet {
	return &listSet{
		name:  name,
		lists: make(map[string]struct{})}
}

func (ipset *listSet) Name() string {
	return ipset.name
}

func (ipset *listSet) AddList(list string) error {
	ipset.lists[list] = struct{}{}
	return nil
}

func (ipset *listSet) DelList(list string) error {
	delete(ipset.lists, list)
	return nil
}

func (ipset *listSet) Count() int {
	return len(ipset.lists)
}
