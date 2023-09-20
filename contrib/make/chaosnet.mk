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
	docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml up --detach --build --pull always

# Stop chaosnet
.PHONY: chaosnet-down
chaosnet-down:
	docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml down --timeout 1 --volumes

###############################################################################
###                               Chaosnet Logs                             ###
###############################################################################

.PHONY: chaosnet-logs
chaosnet-logs:
	docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml logs

.PHONY: chaosnet-logs-faucet
chaosnet-logs-faucet:
	docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml logs go-faucet

.PHONY: chaosnet-logs-pricefeeder
chaosnet-logs-pricefeeder:
	docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml logs pricefeeder --follow

.PHONY: chaosnet-logs-go-heartmonitor
chaosnet-logs-go-heartmonitor:
	docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml logs go-heartmonitor

###############################################################################
###                              Chaosnet SSH                               ###
###############################################################################

.PHONY: chaosnet-ssh-nibiru
chaosnet-ssh-nibiru:
	docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml exec -it nibiru /bin/sh

.PHONY: chaosnet-ssh-go-heartmonitor
chaosnet-ssh-go-heartmonitor:
	docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml exec -it go-heartmonitor /bin/sh

