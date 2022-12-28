###############################################################################
###                                  Proto                                  ###
###############################################################################

containerProtoVer=v0.3
containerProtoImage=tendermintdev/sdk-proto-gen:$(containerProtoVer)
containerProtoGen=cosmos-sdk-proto-gen-$(containerProtoVer)
containerProtoGenSwagger=cosmos-sdk-proto-gen-swagger-$(containerProtoVer)
containerProtoFmt=cosmos-sdk-proto-fmt-$(containerProtoVer)

mock-gen:
	go generate ./...
proto-gen:
	@echo "Generating Protobuf files"
	@if docker ps -a --format '{{.Names}}' | grep -Eq "^${containerProtoGen}$$"; then docker start -a $(containerProtoGen); else docker run --name $(containerProtoGen) -v "$(CURDIR)":/workspace --workdir /workspace $(containerProtoImage) \
		sh ./scripts/protocgen.sh; fi

.PHONY: proto

###############################################################################
###                               Build Flags                               ###
###############################################################################

BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
COMMIT := $(shell git log -1 --format='%H')

# don't override user values
ifeq (,$(VERSION))
  VERSION := $(shell git describe --exact-match 2>/dev/null)
  # if VERSION is empty, then populate it with branch's name and raw commit hash
  ifeq (,$(VERSION))
    VERSION := $(BRANCH)-$(COMMIT)
  endif
endif

SDK_PACK := $(shell go list -m github.com/cosmos/cosmos-sdk | sed  's/ /\@/g')
TM_VERSION := $(shell go list -m github.com/tendermint/tendermint | sed 's:.* ::') # grab everything after the space in "github.com/tendermint/tendermint v0.34.7"
DOCKER := $(shell which docker)
BUILDDIR ?= $(CURDIR)/build

export GO111MODULE = on

# process build tags
build_tags = netgo
build_tags += $(BUILD_TAGS)
build_tags := $(strip $(build_tags))

whitespace :=
whitespace += $(whitespace)
comma := ,
build_tags_comma_sep := $(subst $(whitespace),$(comma),$(build_tags))

# process linker flags
ldflags = -X github.com/cosmos/cosmos-sdk/version.Name=nibiru \
		  -X github.com/cosmos/cosmos-sdk/version.AppName=nibid \
		  -X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
		  -X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT) \
		  -X "github.com/cosmos/cosmos-sdk/version.BuildTags=$(build_tags_comma_sep)" \
			-X github.com/tendermint/tendermint/version.TMCoreSemVer=$(TM_VERSION)
ldflags += $(LDFLAGS)
ldflags := $(strip $(ldflags))

BUILD_FLAGS := -tags "$(build_tags)" -ldflags '$(ldflags)'

###############################################################################
###                                  Build                                  ###
###############################################################################

# command for make build and make install
build: BUILD_ARGS=-o $(BUILDDIR)/
build install: go.sum $(BUILDDIR)/
	go $@ -mod=readonly $(BUILD_FLAGS) $(BUILD_ARGS) ./...

# ensure build directory exists
$(BUILDDIR)/:
	mkdir -p $(BUILDDIR)/

# build for linux architecture
build-linux: go.sum
	CGO_ENABLED=1 LEDGER_ENABLED=false GOOS=linux GOARCH=amd64 $(MAKE) build

go.sum: go.mod
	@echo "--> Ensure dependencies have not been modified"
	@go mod verify

.PHONY: build install

###############################################################################
###                           Docker Localnet                               ###
###############################################################################

build-docker-nibidnode:
	docker build --tag nibiru/nibidnode .

# Run a 4-node testnet locally
localnet-start: build-linux localnet-stop
	@if ! [ -f build/node0/nibid/config/genesis.json ]; then docker run --rm -v $(CURDIR)/build:/nibid:Z nibiru/nibidnode testnet --v 4 -o . --starting-ip-address 192.168.11.2 --keyring-backend=test ; fi
	docker-compose up -d

# Stop testnet
localnet-stop:
	docker-compose down

.PHONY: localnet

###############################################################################
###                            Shell Localnet                               ###
###############################################################################

# Run a single testnet locally
localnet:
	./scripts/localnet.sh

.PHONY: localnet

###############################################################################
###                            Tests & Simulations                          ###
###############################################################################

BINDIR               ?= $(GOPATH)/bin
BUILDDIR             ?= $(CURDIR)/build
SIMAPP                = ./simapp
PACKAGES_NOSIMULATION = $(shell go list ./... | grep -v '/simapp')
RUNSIM                = $(BINDIR)/runsim

test-unit:
	@go test $(PACKAGES_NOSIMULATION) -short -cover

test-integration:
	@go test -v $(PACKAGES_NOSIMULATION) -cover

runsim: $(RUNSIM)
$(RUNSIM):
	@echo "Installing runsim..."
	@(cd /tmp && go install github.com/cosmos/tools/cmd/runsim@v1.0.0)

test-sim-nondeterminism:
	@echo "Running non-determinism test..."
	@go test -mod=readonly $(SIMAPP) -run TestAppStateDeterminism -Enabled=true \
		-NumBlocks=100 -BlockSize=200 -Commit=true -Period=0 -v -timeout 24h

test-sim-default-genesis-fast:
	@echo "Running default genesis simulation..."
	@go test -mod=readonly $(SIMAPP) -run TestFullAppSimulation  \
		-Enabled=true -NumBlocks=100 -BlockSize=200 -Commit=true -Seed=99 -Period=5 -v

test-sim-custom-genesis-multi-seed: runsim
	@echo "Running multi-seed custom genesis simulation..."
	@$(RUNSIM) -SimAppPkg=$(SIMAPP) -ExitOnFail 400 5 TestFullAppSimulation

test-sim-multi-seed-long: runsim
	@echo "Running long multi-seed application simulation. This may take awhile!"
	@$(RUNSIM) -Jobs=4 -SimAppPkg=$(SIMAPP) -ExitOnFail 500 50 TestFullAppSimulation

test-sim-multi-seed-short: runsim
	@echo "Running short multi-seed application simulation. This may take awhile!"
	@$(RUNSIM) -Jobs=4 -SimAppPkg=$(SIMAPP) -ExitOnFail 50 10 TestFullAppSimulation

test-sim-benchmark-invariants:
	@echo "Running simulation invariant benchmarks..."
	@go test -mod=readonly $(SIMAPP) -benchmem -bench=BenchmarkInvariants -run=^$ \
	-Enabled=true -NumBlocks=1000 -BlockSize=200 \
	-Period=1 -Commit=true -Seed=57 -v -timeout 24h

# Require Python3
test-create-test-cases:
	@python scripts/testing/stableswap_model.py

###############################################################################
###                            Lint                                         ###
###############################################################################
release:
	docker run --rm -v "$(CURDIR)":/code -w /code goreleaser/goreleaser-cross --skip-publish --rm-dist

build-docker: go.sum $(BUILDDIR)/
	go build -mod=readonly $(BUILD_FLAGS) $(BUILD_ARGS) ./cmd/nibid/main.go
	tar -czvf build/$(TARNAME).tar.gz build/nibid
	rm build/nibid

build-linux-docker:
	BUILD_TAGS=muslc LINK_STATICALLY=true  TARNAME="nibiru_linux_amd64" CC=aarch64-linux-gnu-gcc BUILD_ARGS="-o $(BUILDDIR)/nibid -tags=muslc" CGO_ENABLED=1 LEDGER_ENABLED=false GOOS=linux GOARCH=amd64 $(MAKE) build-docker
	# TARNAME="nibiru_linux_arm64" BUILD_ARGS="-o $(BUILDDIR)/nibid -tags=muslc" CGO_ENABLED=1 LEDGER_ENABLED=false GOOS=linux GOARCH=arm64 $(MAKE) build-docker

###############################################################################
###                            Lint                                         ###
###############################################################################

lint:
	docker run -v $(CURDIR):/code --rm -w /code golangci/golangci-lint:v1.49-alpine golangci-lint run

.PHONY: \
test-sim-nondeterminism \
test-sim-custom-genesis-fast \
test-sim-custom-genesis-multi-seed \
test-sim-multi-seed-short \
test-sim-multi-seed-long \
test-sim-benchmark-invariants
