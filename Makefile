TAG?=latest
NAMESPACE?=functions
.PHONY: build

build:
	./build.sh $(TAG)

push:
	docker push openfaas/vcenter-connector:$(TAG)

armhf-build:
	./build.sh $(TAG)