# Overview

# ipsets

The policy controller maintains a number of ipsets which are
subsequently referred to by the iptables rules used to effect network
policy specifications. These ipsets are created, modified and
destroyed automatically in response to Pod, Namespace and
NetworkPolicy object updates from the k8s API server:

* A `hash:ip` set per namespace, containing the IP addresses of all
  pods in that namespace
* A `list:set` per distinct (across all network policies in all
  namespaces) namespace selector mentioned in a network policy,
  containing the names of any of the above hash:ip sets whose
  corresponding namespace labels match the selector
* A `hash:ip` set for each distinct (within the scope of the
  containing network policy's namespace) pod selector mentioned in a
  network policy, containing the IP addresses of all pods in the
  namespace whose labels match that selector

ipset names are generated deterministically from a string
representation of the corresponding label selector. Because ipset
names are limited to 31 characters in length, this is done by taking a
SHA hash of the selector string and then printing that out as a base
85 string with a "weave-" prefix e.g.:

    weave-k?Z;25^M}|1s7P3|H9i;*;MhG

Because pod selectors are scoped to a namespace, we need to make sure
that if the same selector definition is used in different namespaces
that we maintain distinct ipsets. Consequently, for such selectors the
namespace name is prepended to the label selector string before
hashing to avoid clashes.

# iptables chains

The policy controller maintains two iptables chains in response to
changes to pods, namespaces and network policies. One chain contains
the ingress rules that implement the network policy specifications,
and the other is used to bypass the ingress rules for namespaces which
have an ingress isolation policy of `DefaultAllow`.

## Dynamically maintained `WEAVE-NPC-DEFAULT` chain

The policy controller maintains a rule in this chain for every
namespace whose ingress isolation policy is `DefaultAllow`. The
purpose of this rule is simply to ACCEPT any traffic destined for such
namespaces before it reaches the ingress chain.

```
iptables -A WEAVE-NPC-DEFAULT -m set --match-set $NSIPSET dst -j ACCEPT
```

## Dynamically maintained `WEAVE-NPC-INGRESS` chain

For each namespace network policy ingress rule peer/port combination:

```
iptables -A WEAVE-NPC-INGRESS -p $PROTO [-m set --match-set $SRCSET] -m set --match-set $DSTSET --dport $DPORT -j ACCEPT
```

## Static `WEAVE-NPC` chain

Static configuration:

```
iptables -A WEAVE-NPC -m state --state RELATED,ESTABLISHED -j ACCEPT
#iptables -A WEAVE-NPC -m state --state NEW -m set ! --match-set $ALLNSIPSET dst -j ACCEPT
iptables -A WEAVE-NPC -m state --state NEW -j WEAVE-NPC-DEFAULT
iptables -A WEAVE-NPC -m state --state NEW -j WEAVE-NPC-INGRESS
iptables -A WEAVE-NPC -j DROP
```

# [WIP] Steering traffic into the policy engine

To direct traffic into the policy engine:

iptables -A FORWARD -i weave -o weave -j WEAVE-NPC

Note this only affects traffic which is _forwarded over_ the specified
bridge device. This rule will not match:

* Traffic which originates on the node itself (e.g. kubelet
  healthchecks) - this goes via the OUTPUT/INPUT chains
* Traffic originating in a container with a non-container destination
  (typically e.g `-i weave -o eth0` with masquerading)
* Traffic originating from an off-node non-container source which is
  then DNATted to a container IP (typically e.g. `-i eth0 -o weave`
  with DNAT)
