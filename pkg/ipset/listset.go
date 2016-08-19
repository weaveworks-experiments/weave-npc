package ipset

import ()

type listSet struct {
	name  string
	lists map[string]struct{}
}

func NewListSet(name string) IPSet {
	return &listSet{
		name:  name,
		lists: make(map[string]struct{})}
}

func (ipset *listSet) Name() string {
	return ipset.name
}

func (ipset *listSet) AddEntry(list string) error {
	ipset.lists[list] = struct{}{}
	return nil
}

func (ipset *listSet) DelEntry(list string) error {
	delete(ipset.lists, list)
	return nil
}

func (ipset *listSet) Count() int {
	return len(ipset.lists)
}
