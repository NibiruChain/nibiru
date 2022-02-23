# Decision record for stablecoin seeker modeling

## Status

Proposed

## Context

We want to understand the impact of stablecoin seeker on the market. We are mainly focused on their interaction with the
insurance fund. Those agents will always pay the IF wether they mint or burn their stable coins, which means that from a
risk perspective, a rally for minting or a "bank run" would not bankrupt the IF.

We still need to define how to model the volume of minting and burning we have per day. 

There's 2 main options to do this modeling:
- **One shot** : We simulate the mint with a random function and then simulate the burn on another random function
    dependent of the supply of stablecoin.
- **Frequency intensity model** : We simulate both mint and burn as a combination of the frequency of operations and
    an intensity or volume of each transaction. This approach allows us to combine for example a Poisson distribution 
    with a Gamma distribution to get the final total amount of asset minted and burned.

## Decision

We are doing a one shot prediction with a gamma law. The person doing ht esimulation can adjust the parameters to test
out different scenarios.
The main driver for this decision is that because we have no risk of bank run for the IF balance, there's no need to 
aim at simulating long tail types of events.

## Consequences

The process is easier to model. One first random variable will be the amount minted at each epoch, while the other will
be the amount burned in function of the current total supply of stablecoins.

All approach studied and chosen assume independence and same distribution for the random varaibles respresenting the
mint and burn.