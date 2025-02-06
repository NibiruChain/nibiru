#########################################################################
# Tests
#########################################################################

.PHONY: test-unit
test-unit:
	go test ./... -short

.PHONY: test-coverage-unit
test-coverage-unit:
	go test ./... -short \
		-tags=pebbledb \
		-coverprofile=coverage.txt \
		-covermode=atomic \
		-race

# NOTE: Using the verbose flag breaks the coverage reporting in CI.
# Used for CI by Codecov
.PHONY: test-coverage-integration
test-coverage-integration:
	go test ./... \
		-tags=pebbledb \
		-coverprofile=coverage.txt \
		-covermode=atomic \
		-race

# Require Python3
.PHONY: test-create-test-cases
test-create-test-cases:
	@python scripts/testing/stableswap_model.py
