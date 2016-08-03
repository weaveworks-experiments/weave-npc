
Three kinds of ipset:
    * Namespace -> hash:ip
    * PodSelector -> hash:ip
    * NamespaceSelector -> list:set (of Namespace)

* Add Pod
    * If Pod has a podIP, add it to the Namespace and matching
      PodSelector ipsets
* Update Pod
    * If Pod has lost its podIP, remove podIP from Namespace and
      matching PodSelector ipsets
    * Else if Pod has gained a podIP, add podIP to Namespace and
      matching PodSelector ipsets
    * Else if Pod's labels are changed, add/remove to/from PodSelector
      ipsets
* Delete Pod
    * If Pod has lost its podIP, remove podIP from Namespace and
      matching PodSelector ipsets

* Add Namespace
* Update Namespace
* Delete Pod

* Add NetworkPolicy
* Upddate NetworkPolicy
* Delete NetworkPolicy

