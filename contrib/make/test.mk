###############################################################################
###                            		  Tests 					                        ###
###############################################################################

PACKAGES_NOSIMULATION = ${shell go list ./... | grep -v simapp}

.PHONY: test-unit
test-unit:
	go test -v $(PACKAGES_NOSIMULATION) -short -coverprofile=coverage.txt -covermode=count

.PHONY: test-integration
test-integration:
	go test -v $(PACKAGES_NOSIMULATION) -coverprofile=coverage.txt -covermode=count

# Used for CI by Codecov
.PHONY: test-coverage
test-coverage:
	go test ./... -short \
		-coverprofile=coverage.txt \
		-covermode=atomic \
		-race | grep -v "no test" | grep -v "no statement"

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
