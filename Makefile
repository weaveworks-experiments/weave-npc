.DEFAULT: all
.PHONY: all clean deploy

all: build/.image.done

clean:
	go clean
	rm -rf ./build

godeps=$(shell go list -f '{{join .Deps "\n"}}' $1 | grep -v /vendor/ | xargs go list -f '{{if not .Standard}}{{ $$dep := . }}{{range .GoFiles}}{{$$dep.Dir}}/{{.}} {{end}}{{end}}')

DEPS=$(call godeps,./cmd/weave-npc)

cmd/weave-npc/weave-npc: $(DEPS)
cmd/weave-npc/weave-npc: cmd/weave-npc/*.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $@ cmd/weave-npc/main.go

build/.image.done: cmd/weave-npc/Dockerfile cmd/weave-npc/weave-npc
	mkdir build
	cp $^ build
	sudo docker build -t harrisonadamw/weave-npc -f build/Dockerfile ./build
	sudo docker push harrisonadamw/weave-npc
	touch $@

deploy: all
	kubectl delete -f k8s/daemonset.yaml
	kubectl create -f k8s/daemonset.yaml

