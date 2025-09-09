###############################################################################
###                           Simple Localnet                               ###
###############################################################################

# Simple localnet script for testing
.PHONY: localnet localnet-uniswap
localnet:
	./contrib/scripts/localnet.sh $(FLAGS)

localnet-uniswap:
	./contrib/scripts/localnet-uniswap.sh $(FLAGS)