* Introduce networkpolicy class to hold add/update/delete behaviour
* Leverage kernel ipset refcounters to avoid keeping per selector
  policy list
* Represent namespace ipset as an empty pod selector
* Use an empty namespace selector with `! -m set` to direct off-net
  traffic
* Add comments to ipset entries and iptables rules
* Implement ability to turn network policy off for a namespace
* Allow 'local' access for healthchecks - mark 'local' traffic in some
  way outside of the policy controller (e.g. have `weave expose` add a
  marking rule)
* Extract ExitError machinery
* Conntrack flushing on rule change
