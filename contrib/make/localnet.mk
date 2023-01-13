###############################################################################
###                           Simple Localnet                               ###
###############################################################################

# Simple localnet script for testing
.PHONY: localnet
localnet:
	./scripts/localnet.sh

###############################################################################
###                           Docker Localnet                               ###
###############################################################################

.PHONY: build-docker-node
build-docker-node:
	docker build -t nibiru/node .

# Run a 4-node testnet locally
.PHONY: localnet-start
localnet-start: build-docker-node localnet-stop
	@if ! [ -f data/node0/nibid/config/genesis.json ]; then \
	docker run --rm -v $(CURDIR)/data:/nibiru:Z nibiru/node testnet \
	--v 4 \
	-o /nibiru \
	--chain-id nibiru-localnet-0 \
	--starting-ip-address 192.168.11.2 \
	--keyring-backend=test; \
	fi
	docker-compose up -d

# Stop testnet
.PHONY: localnet-stop
localnet-stop:
	docker-compose down