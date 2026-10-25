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

# Start 4 validators on one chain (multival profile) without deleting local state.
up-multival:
    docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml --profile multival build multival-genesis
    docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml --profile multival up --detach multival-genesis nibiru-val-0 nibiru-val-1 nibiru-val-2 nibiru-val-3

# Check app hash agreement across 4 validators under load and restarts. Ex: "LOAD_SECONDS=300 just chaos determinism"
determinism:
    bash contrib/scripts/determinism.sh

# Self-test: plant a non-deterministic EndBlock and pass only if `determinism` catches it.
determinism-canary:
    #!/usr/bin/env bash
    set -uo pipefail
    log="$(mktemp)"
    MULTIVAL_IMAGE=nibiru-multival:canary \
      MULTIVAL_BUILD_TAGS=nibiru_nondeterminism_canary \
      LOAD_SECONDS="${LOAD_SECONDS:-30}" \
      bash contrib/scripts/determinism.sh 2>&1 | tee "$log"
    status="${PIPESTATUS[0]}"
    if [[ "$status" -ne 0 ]] && grep -q "first height where app hashes differ" "$log"; then
      echo "✅ determinism caught the planted bug"
      rm -f "$log"
      exit 0
    fi
    echo "❌ determinism did not catch the planted bug (exit $status)"
    rm -f "$log"
    exit 1

# Stop every Chaosnet profile while preserving named volumes.
down:
    docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml --profile ibc down --timeout 1
    docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml --profile heartmonitor down --timeout 1
    docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml --profile multival down --timeout 1

# Remove every Chaosnet profile and its named volumes.
[confirm]
destroy:
    docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml --profile ibc down --timeout 1 --volumes
    docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml --profile heartmonitor down --timeout 1 --volumes
    docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml --profile multival down --timeout 1 --volumes

# Print logs from every Chaosnet service.
logs:
    docker compose -f ./contrib/docker-compose/docker-compose-chaosnet.yml --profile ibc --profile heartmonitor --profile multival logs

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
