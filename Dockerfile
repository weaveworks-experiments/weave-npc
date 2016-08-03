FROM scratch
COPY ./weave-npc /bin/weave-npc
ENTRYPOINT ["/bin/weave-npc"]
