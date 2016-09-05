# Features

* Default-allow 'ingress' to external addresses (use an
  all-namespace selector with `! --match-set` to direct)
* Add comments to ipset entries and iptables rules
* Conntrack flushing on rule change

# Refactorings

* Represent namespace ipset as an empty pod selector
* Leverage kernel ipset refcounters to avoid keeping per selector
  policy list
* Introduce networkpolicy class to hold add/update/delete behaviour
* Extract ExitError machinery
