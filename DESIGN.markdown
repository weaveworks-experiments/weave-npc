# iptables chains

## WEAVE-NPC (DENY)

iptables -A FORWARD -m state --state RELATED,ESTABLISHED -j ACCEPT
iptables -A FORWARD -m state --state NEW -j WEAVE-NPC-DEFAULT
iptables -A FORWARD -m state --state NEW -j WEAVE-NPC-INGRESS

## WEAVE-NPC-DEFAULT

For each namespace that has the default ingress policy:

iptables -A WEAVE-NPC-DEFAULT -m set --match-set $NSIPSET dst -j ACCEPT

## WEAVE-NPC-INGRESS

For each namespace network policy ingress rule peer/port combination:

iptables -A WEAVE-NPC-INGRESS -p $PROTO [-m set --match-set $SRCSET] [-m set --match-set $DSTSET] --dport $DPORT -j ACCEPT


