###############################################################################
###                           Simple Localnet                               ###
###############################################################################

# Simple localnet script for testing
.PHONY: localnet
localnet:
	bash ./cmd/nibid/localnet.sh --run $(FLAGS)
