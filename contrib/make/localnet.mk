###############################################################################
###                           Simple Localnet                               ###
###############################################################################

# Simple localnet script for testing
.PHONY: localnet
localnet:
	./contrib/scripts/localnet.sh

###############################################################################
###                           Docker Localnet                               ###
###############################################################################

.PHONY: build-docker-node
build-docker-node:
	docker buildx build --load -t nibiru/node -f Dockerfile .

# Run a 4-node testnet locally
.PHONY: localnet-start
localnet-start: localnet-stop build-docker-node
	@if ! [ -f data/node0/nibid/config/genesis.json ]; then \
		docker run --rm -v $(CURDIR)/data:/nibiru:Z nibiru/node testnet \
			--v 4 \
			-o /nibiru \
			--chain-id nibiru-localnet-0 \
			--starting-ip-address 192.168.11.2 \
			--keyring-backend=test; \
	fi
	docker compose -f ./contrib/docker-compose/docker-compose.yml up -d

# Stop testnet
.PHONY: localnet-stop
localnet-stop:
	docker compose -f ./contrib/docker-compose/docker-compose.yml down