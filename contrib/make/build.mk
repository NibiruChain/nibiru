
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


OS_NAME := $(shell uname -s | tr A-Z a-z)
ifeq ($(shell uname -m),x86_64)
	ARCH_NAME := amd64
else
	ARCH_NAME := arm64
endif

SDK_PACK := $(shell go list -m github.com/cosmos/cosmos-sdk | sed  's/ /\@/g')
TM_VERSION := $(shell go list -m github.com/cometbft/cometbft | sed 's:.* ::') # grab everything after the space in "github.com/tendermint/tendermint v0.34.7"
ROCKSDB_VERSION := 8.1.1
WASMVM_VERSION := $(shell go list -m github.com/CosmWasm/wasmvm | awk '{sub(/^v/, "", $$2); print $$2}')
DOCKER := $(shell which docker)
BUILDDIR ?= $(CURDIR)/build
TEMPDIR ?= $(CURDIR)/temp

export GO111MODULE = on

# process build tags
build_tags = netgo osusergo rocksdb pebbledb grocksdb_no_link static_wasm muslc ledger
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
		  -X github.com/tendermint/tendermint/version.TMCoreSemVer=$(TM_VERSION) \
		  -X github.com/cosmos/cosmos-sdk/types.DBBackend=rocksdb \
		  -linkmode=external \
		  -w -s

ldflags += $(LDFLAGS)
ldflags := $(strip $(ldflags))

BUILD_FLAGS := -tags "$(build_tags)" -ldflags '$(ldflags)'
CGO_CFLAGS  := -I$(TEMPDIR)/include
CGO_LDFLAGS := -L$(TEMPDIR)/lib -lrocksdb -lstdc++ -lm -ldl
ifeq ($(OS_NAME),darwin)
	CGO_LDFLAGS += -lz -lbz2
else
	CGO_LDFLAGS += -static -lwasmvm_muslc
endif

###############################################################################
###                                  Build                                  ###
###############################################################################

$(TEMPDIR)/:
	mkdir -p $(TEMPDIR)/

# download required libs
rocksdblib: $(TEMPDIR)/
	@mkdir -p $(TEMPDIR)/include
	@mkdir -p $(TEMPDIR)/lib
    ifeq (",$(wildcard $(TEMPDIR)/include/rocksdb)")
	    wget https://github.com/NibiruChain/gorocksdb/releases/download/v$(ROCKSDB_VERSION)/include.$(ROCKSDB_VERSION).tar.gz -O - | tar -xz -C $(TEMPDIR)/include/
    endif
    ifeq (",$(wildcard $(TEMPDIR)/lib/librocksdb.a)")
	    wget https://github.com/NibiruChain/gorocksdb/releases/download/v$(ROCKSDB_VERSION)/librocksdb_$(ROCKSDB_VERSION)_$(OS_NAME)_$(ARCH_NAME).tar.gz -O - | tar -xz -C $(TEMPDIR)/lib/
    endif

wasmvmlib: $(TEMPDIR)/
	@mkdir -p $(TEMPDIR)/lib
    ifeq (",$(wildcard $(TEMPDIR)/lib/libwasmvm*.a)")
        ifeq ($(OS_NAME),darwin)
	        wget https://github.com/CosmWasm/wasmvm/releases/download/v$(WASMVM_VERSION)/libwasmvmstatic_darwin.a -O $(TEMPDIR)/lib/libwasmvmstatic_darwin.a
        else
            ifeq ($(ARCH_NAME),amd64)
	            wget https://github.com/CosmWasm/wasmvm/releases/download/v$(WASMVM_VERSION)/libwasmvm_muslc.x86_64.a -O $(TEMPDIR)/lib/libwasmvm_muslc.a
            else
	            wget https://github.com/CosmWasm/wasmvm/releases/download/v$(WASMVM_VERSION)/libwasmvm_muslc.aarch64.a -O $(TEMPDIR)/lib/libwasmvm_muslc.a
            endif
        endif
    endif

# command for make build and make install
build: BUILDARGS=-o $(BUILDDIR)/
build install: go.sum $(BUILDDIR)/ rocksdblib wasmvmlib
	CGO_ENABLED=1 CGO_CFLAGS="$(CGO_CFLAGS)" CGO_LDFLAGS="$(CGO_LDFLAGS)" go $@ -mod=readonly $(BUILD_FLAGS) $(BUILDARGS) ./...

# ensure build directory exists
$(BUILDDIR)/:
	mkdir -p $(BUILDDIR)/

go.sum: go.mod
	@echo "--> Ensure dependencies have not been modified"
	@go mod verify

.PHONY: build install
