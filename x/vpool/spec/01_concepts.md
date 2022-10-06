<!--
order: 1
-->
# Concepts | x/vpool                    <!-- omit in toc -->

A virtual pool is a virtual automated market maker based on the xyk model popularized by [Uniswap V2](https://uniswap.org/docs/v2/protocol-overview/how-uniswap-works/).
These vpools only interact with the clearing house, which uses collateral posted by a trader to swap the quote for the base, or swap the base for the quote.

This means that operations behave the same as a classic xyk AMM swap pool, but if a trader opens a position he will not receive a token. Instead, the clearing house will hold the state of the traders position, and await for new transactions to interact with it.

Since the clearing house is the only place interacting with the vpools, this module does not contains transactions. You can however use the queries to get information about the state of the pool at any height of the chain.
