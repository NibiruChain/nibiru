all: docker generator runner

# The build context is the repo root, since bft/ belongs to the root Go
# module. `just bft e2e-image` is the maintained entry point.
docker:
	@echo "Building E2E Docker image"
	@docker build \
		--tag cometbft/e2e-node:local-version \
		-f docker/Dockerfile ../../..

# We need to build support for database backends into the app in
# order to build a binary with a CometBFT node in it (for built-in
# ABCI testing).
node:
	go build -o build/node -tags badgerdb,boltdb,cleveldb,rocksdb ./node

generator:
	go build -o build/generator ./generator

runner:
	go build -o build/runner ./runner

.PHONY: all node docker generator runner 
