# Chaosnet commands use the Compose file at contrib/docker-compose/docker-compose-chaosnet.yml.
# `up`, `up-ibc`, and `up-hm` preserve named volumes. Run `destroy` only when a fresh local state is wanted.

# List Chaosnet recipes.
setup:
    just --justfile just-chaos.just --list

# Rebuild Chaosnet images without cache and pull base images.
build:
    docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml build --no-cache --pull

# Start the single-validator Chaosnet profile without deleting local state.
up:
    docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml up --detach --build

# Start the IBC Chaosnet profile without deleting local state.
up-ibc:
    docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml --profile ibc up --detach --build

# Start the Heart Monitor Chaosnet profile without deleting local state.
up-hm:
    docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml --profile heartmonitor up --detach --build

# Stop every Chaosnet profile while preserving named volumes.
down:
    docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml --profile ibc down --timeout 1
    docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml --profile heartmonitor down --timeout 1

# Remove every Chaosnet profile and its named volumes.
[confirm]
destroy:
    docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml --profile ibc down --timeout 1 --volumes
    docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml --profile heartmonitor down --timeout 1 --volumes

# Print logs from every Chaosnet service.
logs:
    docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml --profile ibc --profile heartmonitor logs

# Print logs from the Heart Monitor service.
logs-hm:
    docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml logs heartmonitor

# Open a shell in the primary validator container.
sh-nibiru-0:
    docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml exec -it nibiru-0 /bin/sh

# Open a shell in the IBC validator container.
sh-nibiru-1:
    docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml exec -it nibiru-1 /bin/sh

# Open a shell in the Heart Monitor container.
sh-go-hm:
    docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml exec -it heartmonitor /bin/sh
