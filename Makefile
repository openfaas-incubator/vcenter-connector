TAG?=latest
NAMESPACE?=functions
.PHONY: build

build:
	./build.sh $(TAG)

armhf-build:
	./build.sh $(TAG)