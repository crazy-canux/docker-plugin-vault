VERSION ?= 0.0.1

create:
	docker build -t rootfsimage .
	docker create --name tmp rootfsimage
	mkdir -p plugin/rootfs/
	docker export tmp | tar -x -C plugin/rootfs/
	docker rm -vf tmp
	docker plugin create kaizen--dev.artifactory.geo.arm.com/docker-plugin-vault:$(VERSION) ./plugin
	docker rmi --force rootfsimage

enable:
	docker plugin enable kaizen--dev.artifactory.geo.arm.com/docker-plugin-vault:$(VERSION)

push:
	docker plugin push kaizen--dev.artifactory.geo.arm.com/docker-plugin-vault:$(VERSION)

.PHONY: clean
clean: clean-cache

.PHONY: clean-cache
clean-cache:
	rm -fr plugin/rootfs/.dockerenv plugin/rootfs/*
	docker plugin rm -f kaizen--dev.artifactory.geo.arm.com/docker-plugin-vault:$(VERSION) 


