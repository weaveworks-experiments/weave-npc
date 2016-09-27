.DEFAULT: all
.PHONY: all clean image publish-image minikube-publish kube-deploy kube-redeploy kube-unedeploy

IMAGE_TAG=latest

all: image

clean:
	go clean
	rm -f cmd/weave-npc/weave-npc
	rm -rf ./build

godeps=$(shell go get $1 && go list -f '{{join .Deps "\n"}}' $1 | grep -v /vendor/ | xargs go list -f '{{if not .Standard}}{{ $$dep := . }}{{range .GoFiles}}{{$$dep.Dir}}/{{.}} {{end}}{{end}}')

DEPS=$(call godeps,./cmd/weave-npc)
VERSION=git-$(shell git rev-parse --short=12 HEAD)

cmd/weave-npc/weave-npc: $(DEPS)
cmd/weave-npc/weave-npc: cmd/weave-npc/*.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o $@ cmd/weave-npc/main.go

build/.image.done: cmd/weave-npc/Dockerfile cmd/weave-npc/weave-npc
	mkdir -p build
	cp $^ build
	sudo docker build -t weaveworks/weave-npc:$(IMAGE_TAG) -f build/Dockerfile ./build
	touch $@

image: build/.image.done

publish-image: image
	sudo docker push weaveworks/weave-npc:$(IMAGE_TAG)

minikube-publish: image
	sudo docker save weaveworks/weave-npc:$(IMAGE_TAG) | (eval $$(minikube docker-env) && docker load)

kube-deploy: all
	kubectl create -f k8s/daemonset.yaml

kube-redeploy: all
	kubectl delete pods --namespace kube-system -l k8s-app=weave-npc

kube-undeploy:
	kubectl delete -f k8s/daemonset.yaml
