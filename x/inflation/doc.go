/*
Package inflation implements the token inflation for the staking and utility
token of the network as described by the "Community" portion of the Nibiru Chain
tokenomics. The inflation curve is implemented as a polynomial that corresponds
to a normalized exponential decay until the network reaches its maximum supply.

References:
  - https://github.com/NibiruChain/tokenomics
  - https://nibiru.fi/docs/learn/tokenomics.html
*/
package inflation
