package ipset

import ()

type HashIP interface {
	Name() string
	AddIP(ip string) error
	DelIP(ip string) error
	Count() int
}

type hashIP struct {
	name string
	ips  map[string]struct{}
}

func NewHashIP(name string) HashIP {
	return &hashIP{
		name: name,
		ips:  make(map[string]struct{})}
}

func (ipset *hashIP) Name() string {
	return ipset.name
}

func (ipset *hashIP) AddIP(ip string) error {
	ipset.ips[ip] = struct{}{}
	return nil
}

func (ipset *hashIP) DelIP(ip string) error {
	delete(ipset.ips, ip)
	return nil
}

func (ipset *hashIP) Count() int {
	return len(ipset.ips)
}
