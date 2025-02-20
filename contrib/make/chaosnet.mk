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
	docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml up --detach --build

# Run a chaosnet testnet locally with IBC enabled
.PHONY: chaosnet-ibc
chaosnet-ibc: chaosnet-down
	docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml --profile ibc up --detach --build

# Run a chaosnet testnet locally with heartmonitor+graphql stack enabled
.PHONY: chaosnet-heartmonitor
chaosnet-heartmonitor: chaosnet-down
	docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml --profile heartmonitor up --detach --build

# Stop chaosnet, need to specify all profiles to ensure all services are stopped
# See https://stackoverflow.com/questions/76781634/docker-compose-down-all-profiles
.PHONY: chaosnet-down
chaosnet-down:
	docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml --profile ibc down --timeout 1 --volumes
	docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml --profile heartmonitor down --timeout 1 --volumes

###############################################################################
###                               Chaosnet Logs                             ###
###############################################################################

.PHONY: chaosnet-logs
chaosnet-logs:
	docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml logs

.PHONY: chaosnet-logs-pricefeeder
chaosnet-logs-pricefeeder:
	docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml logs pricefeeder --follow

.PHONY: chaosnet-logs-go-heartmonitor
chaosnet-logs-go-heartmonitor:
	docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml logs go-heartmonitor

###############################################################################
###                              Chaosnet SSH                               ###
###############################################################################

.PHONY: chaosnet-ssh-nibiru-0
chaosnet-ssh-nibiru-0:
	docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml exec -it nibiru-0 /bin/sh

.PHONY: chaosnet-ssh-nibiru-1
chaosnet-ssh-nibiru-1:
	docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml exec -it nibiru-1 /bin/sh

.PHONY: chaosnet-ssh-go-heartmonitor
chaosnet-ssh-go-heartmonitor:
	docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml exec -it go-heartmonitor /bin/sh

