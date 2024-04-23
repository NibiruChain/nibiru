###
# Find OS and Go environment
# GO contains the Go binary
# FS contains the OS file separator
###
ifeq ($(OS),Windows_NT)
  GO := $(shell where go.exe 2> NUL)
  FS := "\\"
else
  GO := $(shell command -v go 2> /dev/null)
  FS := "/"
endif

ifeq ($(GO),)
  $(error could not find go. Is it in PATH? $(GO))
endif

GOPATH ?= $(shell $(GO) env GOPATH)
TOOLS_DESTDIR  ?= $(GOPATH)/bin
STATIK         = $(TOOLS_DESTDIR)/statik
BINDIR ?= $(GOPATH)/bin

protoImage=$(DOCKER) run --rm -v $(CURDIR):/workspace --workdir /workspace $(protoImageName)

statik: $(STATIK)
$(STATIK):
	@echo "Installing statik..."
	@go install github.com/rakyll/statik@v0.1.6

proto-swagger-gen:
	@echo "Generating Protobuf Swagger"
	@$(protoImage) sh ./contrib/scripts/protoc-swagger-gen.sh
	$(MAKE) update-swagger-docs

update-swagger-docs: statik
	$(BINDIR)/statik -src=contrib/swagger/swagger-ui -dest=contrib/swagger -f -m
	@if [ -n "$(git status --porcelain)" ]; then \
		echo "\033[91mSwagger docs are out of sync!!!\033[0m";\
		exit 1;\
	else \
		echo "\033[92mSwagger docs are in sync\033[0m";\
	fi
.PHONY: update-swagger-docs

