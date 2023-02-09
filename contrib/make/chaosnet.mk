###############################################################################
###                           Chaosnet Localnet                             ###
###############################################################################

# Triggers a force rebuild of the chaosnet images
.PHONY: chaosnet-build
chaosnet-build:
	docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml build --no-cache --pull

# Run a chaosnet testnet locally
.PHONY: chaosnet
chaosnet: chaosnet-down
	docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml up --detach

# Stop chaosnet
.PHONY: chaosnet-down
chaosnet-down:
	docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml down --timeout 1 --volumes

###############################################################################
###                               Chaosnet Logs                             ###
###############################################################################

# Run a chaosnet testnet locally
.PHONY: chaosnet-logs
chaosnet-logs:
	docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml logs

# Run a chaosnet testnet locally
.PHONY: chaosnet-logs-faucet
chaosnet-logs-faucet:
	docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml logs faucet

###############################################################################
###                              Chaosnet SSH                               ###
###############################################################################

# Run a chaosnet testnet locally
.PHONY: chaosnet-ssh-nibiru
chaosnet-ssh-nibiru:
	docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml exec -it nibiru /bin/sh

