###############################################################################
###                               Release                                   ###
###############################################################################

PACKAGE_NAME		  := github.com/NibiruChain/nibiru
GOLANG_CROSS_VERSION  ?= v1.21.5
CMT_VERSION 		  = $(shell go list -m github.com/cometbft/cometbft | sed 's:.* ::')

DOCKER_YQ = docker run --rm -v $(PWD):/work -w /work mikefarah/yq:4

DARWIN_TAGS = $(shell $(DOCKER_YQ) e \
    '.builds | map(select(.id == "darwin")) | .[0].tags | join(",")' \
    .goreleaser.yml)

LINUX_TAGS = $(shell $(DOCKER_YQ) e \
    '.builds | map(select(.id == "linux")) | .[0].tags | join(",")' \
    .goreleaser.yml)

# The `make release` command is running a Docker container with the image
# `gorelease/goreleaser-cross:${GOLANG_CROSS_VERSION}`. This command:
# `-v "$(CURDIR)":/go/src/$(PACKAGE_NAME)`: mounts the current directory
# `release --clean`: executes the release inside the directory
release:
	@echo "Darwin Tags: $(DARWIN_TAGS)"
	@echo "Linux Tags: $(LINUX_TAGS)"
	docker run \
		--rm \
		--platform linux/amd64 \
		-v "$(CURDIR)":/go/src/$(PACKAGE_NAME) \
		-w /go/src/$(PACKAGE_NAME) \
		-e GITHUB_TOKEN=${GITHUB_TOKEN} \
		-e CMT_VERSION=$(CMT_VERSION) \
		-e DARWIN_TAGS="$(DARWIN_TAGS)" \
		-e LINUX_TAGS="$(LINUX_TAGS)" \
		goreleaser/goreleaser-cross:${GOLANG_CROSS_VERSION} \
		release --clean

release-snapshot:
	@echo "Darwin Tags: $(DARWIN_TAGS)"
	@echo "Linux Tags: $(LINUX_TAGS)"
	docker run \
		--rm \
		--platform linux/amd64 \
		-v /tmp:/tmp \
		-v "$(CURDIR)":/go/src/$(PACKAGE_NAME) \
		-w /go/src/$(PACKAGE_NAME) \
		-e CMT_VERSION=$(CMT_VERSION) \
		-e DARWIN_TAGS="$(DARWIN_TAGS)" \
		-e LINUX_TAGS="$(LINUX_TAGS)" \
		goreleaser/goreleaser-cross:${GOLANG_CROSS_VERSION} \
		release --clean --snapshot