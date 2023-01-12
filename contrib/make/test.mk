###############################################################################
###                            		  Tests 					                        ###
###############################################################################

PACKAGES_NOSIMULATION = $(shell go list ./... | grep -v '/simapp')

.PHONY: test-unit
test-unit:
	@go test $(PACKAGES_NOSIMULATION) -short -cover

.PHONY: test-integration
test-integration:
	@go test -v $(PACKAGES_NOSIMULATION) -cover

# Require Python3
.PHONY: test-create-test-cases
test-create-test-cases:
	@python scripts/testing/stableswap_model.py