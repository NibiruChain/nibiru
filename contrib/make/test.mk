#########################################################################
# Tests
#########################################################################

PACKAGES_NOSIMULATION = ${shell go list ./... | grep -v simapp}

.PHONY: test-coverage
test-coverage:
	go test ./... $(PACKAGES_NOSIMULATION) -short \
		-coverprofile=coverage.txt \
		-covermode=atomic \
		-race | grep -v "no test" | grep -v "no statement"

# NOTE: Using the verbose flag breaks the coverage reporting in CI.
# Used for CI by Codecov
.PHONY: test-coverage-integration
test-coverage-integration:
	go test ./... \
		-coverprofile=coverage.txt \
		-covermode=atomic \
		-race | grep -v "no test" | grep -v "no statement"

# Require Python3
.PHONY: test-create-test-cases
test-create-test-cases:
	@python scripts/testing/stableswap_model.py
