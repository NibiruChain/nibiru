---
order: 21
canonicalUrl: "https://nibiru.fi/docs/wallets/common-tokens-evm.html"
---

# Token Addresses (Nibiru EVM)

Listed below are the contract addresses for commonly used ERC20 assets within the Nibiru EVM ecosystem. To learn how to add manually assets to your wallet, [click here](../use/import-token-evm.md). 

## Core Assets

<template>
  <TokenBoxes :boxes="boxesMain" />
</template>

<script>
const boxesMain = [
  {
  id: 1,
  title: "WNIBI - Wrapped NIBI",
  href: "#",
  icon: "https://raw.githubusercontent.com/NibiruChain/nibiru/main/token-registry/img/000_nibiru-evm.png",
  // icon: "/img/wnibi-icon.svg",
  // text: "Optional text describing the element to go here.",
  address: "0x0CaCF669f8446BeCA826913a3c6B96aCD4b02a97"
  },
  {
  id: 2,
  title: "stNIBI - Liquid Staked NIBI",
  href: "https://nibiru.fi/ecosystem/apps/liquid-staked-nibiru-stnibi",
  icon: "https://raw.githubusercontent.com/NibiruChain/nibiru/main/token-registry/img/001_stnibi-evm.png",
  // text: "Optional text describing the element to go here.",
  address: "0xcA0a9Fb5FBF692fa12fD13c0A900EC56Bb3f0a7b"
  },
]

const boxesStable = [
  {
    id: 3,
    title: "USDC.e - USDC (Stargate)",
    href: "https://nibiru.fi/ecosystem/apps/layerzero-stargate",
    icon: "https://raw.githubusercontent.com/NibiruChain/nibiru/main/token-registry/img/002_usdc.png",
    // text: "Optional text describing the element to go here.",
    address: "0x0829F361A05D993d5CEb035cA6DF3446b060970b"
  },
  {
    id: 4,
    title: "USDT - USDT (Stargate)",
    href: "https://nibiru.fi/ecosystem/apps/layerzero-stargate",
    icon: "https://raw.githubusercontent.com/NibiruChain/nibiru/main/token-registry/img/006_usdt.svg",
    // text: "Optional text describing the element to go here.",
    address: "0x43F2376D5D03553aE72F4A8093bbe9de4336EB08"
  },
  {
    id: 5,
    title: "USDC.arb - Bridged USDC (VIA Labs)",
    href: "",
    icon: "https://raw.githubusercontent.com/NibiruChain/nibiru/main/token-registry/img/002_usdc-arb.png",
    // text: "Optional text describing the element to go here.",
    address: "0x08EBA8ff53c6ee5d37A90eD4b5239f2F85e7B291"
  },
]

const boxesEco = [
  {
    id: 1,
    title: "MIM - Magic Internet Money",
    href: "https://app.abracadabra.money/nibiru#/mim-swap?chain=nibiru",
    icon: "https://raw.githubusercontent.com/NibiruChain/nibiru/main/token-registry/img/007_mim.png",
    // text: "Optional text describing the element to go here.",
    address: "0xfCfc58685101e2914cBCf7551B432500db84eAa8"
  },
  {
    id: 2,
    title: "AXV - Astrovault (Wrapped)",
    href: "",
    icon: "https://raw.githubusercontent.com/NibiruChain/nibiru/main/token-registry/img/003_astrovault-axv.png",
    // text: "Optional text describing the element to go here.",
    address: "0x7168634Dd1ee48b1C5cC32b27fD8Fc84E12D00E6"
  },
  {
    id: 3,
    title: "WETH (Stargate) - Wrapped ETH",
    href: "https://nibiscan.io/address/0xcdA5b77E2E2268D9E09c874c1b9A4c3F07b37555",
    icon: "https://raw.githubusercontent.com/NibiruChain/nibiru/main/token-registry/img/005_eth.svg",
    // text: "Optional text describing the element to go here.",
    address: "0xcdA5b77E2E2268D9E09c874c1b9A4c3F07b37555"
  }
]

export default {
  data() {
    return {
      boxesMain,
      boxesStable,
      boxesEco,
    }
  }
}
</script>

## Stablecoins

<template>
  <TokenBoxes :boxes="boxesStable" />
</template>

## Ecosystem Assets

<template>
  <TokenBoxes :boxes="boxesEco" />
</template>

- Protocol: [Astrovault](https://astrovault.io/)
