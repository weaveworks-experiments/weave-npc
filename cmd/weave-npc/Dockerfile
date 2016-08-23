FROM alpine
RUN apk add --update \
    iptables \
    ipset \
  && rm -rf /var/cache/apk/*
COPY ./weave-npc /usr/bin/weave-npc
ENTRYPOINT ["/usr/bin/weave-npc"]
