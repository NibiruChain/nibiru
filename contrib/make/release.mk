###############################################################################
###                               Release                                   ###
###############################################################################

PACKAGE_NAME		  := github.com/NibiruChain/nibiru
GOLANG_CROSS_VERSION  ?= v1.19.4

release:
	docker run \
		--rm \
		--privileged \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v "$(CURDIR)":/go/src/$(PACKAGE_NAME) \
		-v "$(CURDIR)/goreleaser-sysroot":/sysroot \
		-w /go/src/$(PACKAGE_NAME) \
		-e CGO_ENABLED=1 \
		goreleaser/goreleaser-cross:${GOLANG_CROSS_VERSION} \
		build --rm-dist --debug --skip-validate
