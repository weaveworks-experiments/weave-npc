# iptables chains

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

## WEAVE-NPC

Static configuration:

```
iptables -A WEAVE-NPC -m state --state RELATED,ESTABLISHED -j ACCEPT
#iptables -A WEAVE-NPC -m state --state NEW -m set ! --match-set $ALLNSIPSET dst -j ACCEPT
iptables -A WEAVE-NPC -m state --state NEW -j WEAVE-NPC-DEFAULT
iptables -A WEAVE-NPC -m state --state NEW -j WEAVE-NPC-INGRESS
iptables -A WEAVE-NPC -j DROP
```

## WEAVE-NPC-DEFAULT

For each namespace that has the default ingress policy:

```
iptables -A WEAVE-NPC-DEFAULT -m set --match-set $NSIPSET dst -j ACCEPT
```

## WEAVE-NPC-INGRESS

For each namespace network policy ingress rule peer/port combination:

```
iptables -A WEAVE-NPC-INGRESS -p $PROTO [-m set --match-set $SRCSET] -m set --match-set $DSTSET --dport $DPORT -j ACCEPT
```

