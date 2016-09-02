* Introduce networkpolicy class to hold add/update/delete behaviour
* Leverage kernel ipset refcounters to avoid keeping per selector policy list
* Represent namespace ipset as an empty pod selector
* Use an empty namespace selector with `! -m set` to direct off-net traffic
