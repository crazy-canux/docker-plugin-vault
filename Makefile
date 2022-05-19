VERSION ?= 0.0.1

create:
	docker build -t rootfsimage .
	docker create --name tmp rootfsimage
	mkdir -p plugin/rootfs/
	docker export tmp | tar -x -C plugin/rootfs/
	docker rm -vf tmp
	docker plugin create canux-dev.eu-west-1.artifactory.canux.com/docker-plugin-vault:$(VERSION) ./plugin
	docker rmi --force rootfsimage

enable:
	docker plugin enable canux-dev.eu-west-1.artifactory.canux.com/docker-plugin-vault:$(VERSION)

push:
	docker plugin push canux-dev.eu-west-1.artifactory.canux.com/docker-plugin-vault:$(VERSION)

.PHONY: clean
clean: clean-cache

.PHONY: clean-cache
clean-cache:
	rm -fr plugin/rootfs/.dockerenv plugin/rootfs/*
	docker plugin rm -f canux-dev.eu-west-1.artifactory.canux.com/docker-plugin-vault:$(VERSION) 


