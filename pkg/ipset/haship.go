package ipset

import ()

type hashIP struct {
	name string
	ips  map[string]struct{}
}

func NewHashIP(name string) IPSet {
	return &hashIP{
		name: name,
		ips:  make(map[string]struct{})}
}

func (ipset *hashIP) Name() string {
	return ipset.name
}

func (ipset *hashIP) AddEntry(ip string) error {
	ipset.ips[ip] = struct{}{}
	return nil
}

func (ipset *hashIP) DelEntry(ip string) error {
	delete(ipset.ips, ip)
	return nil
}

func (ipset *hashIP) Count() int {
	return len(ipset.ips)
}
