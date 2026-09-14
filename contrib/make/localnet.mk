###############################################################################
###                           Simple Localnet                               ###
###############################################################################

# Simple localnet script for testing
.PHONY: localnet
localnet:
	bash ./x/cli/localnet.sh --run $(FLAGS)
