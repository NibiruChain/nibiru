.PHONY: proto

proto:
	docker build -t dev:proto --progress="plain" -f ./contrib/proto.dockerfile .
	docker run -v "$(CURDIR):/work" -w /work/proto dev:proto buf mod update
	docker run -v "$(CURDIR):/work" -w /work/proto dev:proto buf generate --template buf.gen.yaml

###############################################################################
###                                  Build                                  ###
###############################################################################

build: go.sum
ifeq ($(OS),Windows_NT)
	go build $(BUILD_FLAGS) -o build/matrixd.exe ./cmd/matrixd
else
	go build $(BUILD_FLAGS) -o build/matrixd ./cmd/matrixd
endif

build-linux: go.sum
	LEDGER_ENABLED=false GOOS=linux GOARCH=amd64 $(MAKE) build

install: go.sum
	go install $(BUILD_FLAGS) ./cmd/matrixd

build-reproducible: go.sum
	$(DOCKER) rm latest-build || true
	$(DOCKER) run --volume=$(CURDIR):/sources:ro \
        --env TARGET_PLATFORMS='linux/amd64 darwin/amd64 linux/arm64' \
        --env APP=matrixd \
        --env VERSION=$(VERSION) \
        --env COMMIT=$(COMMIT) \
        --env LEDGER_ENABLED=$(LEDGER_ENABLED) \
        --name latest-build cosmossdk/rbuilder:latest
	$(DOCKER) cp -a latest-build:/home/builder/artifacts/ $(CURDIR)/

.PHONY: build

###############################################################################
###                               Localnet                                  ###
###############################################################################

# Run a single testnet locally
localnet:
	./scripts/localnet.sh

.PHONY: localnet
