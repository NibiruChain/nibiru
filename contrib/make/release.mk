###############################################################################
###                               Release                                   ###
###############################################################################

PACKAGE_NAME		  := github.com/NibiruChain/nibiru
GOLANG_CROSS_VERSION  ?= v1.21.5

# The `make release` command is running a Docker container with the image 
# `gorelease/goreleaser-cross:${GOLANG_CROSS_VERSION}`. This command:
# `-v "$(CURDIR)":/go/src/$(PACKAGE_NAME)`: mounts the current directory 
# `release --clean`: executes the release inside the directory
release:
	docker run \
		--rm \
		--platform linux/amd64 \
		-v "$(CURDIR)":/go/src/$(PACKAGE_NAME) \
		-w /go/src/$(PACKAGE_NAME) \
		-e CGO_ENABLED=1 \
		-e GITHUB_TOKEN=${GITHUB_TOKEN} \
		goreleaser/goreleaser-cross:${GOLANG_CROSS_VERSION} \
		release --clean

release-snapshot:
	docker run \
		--rm \
		--platform linux/amd64 \
		-v /tmp:/tmp \
		-v "$(CURDIR)":/go/src/$(PACKAGE_NAME) \
		-w /go/src/$(PACKAGE_NAME) \
		-e CGO_ENABLED=1 \
		goreleaser/goreleaser-cross:${GOLANG_CROSS_VERSION} \
		release --clean --snapshot