---
order: 1
canonicalUrl: "https://nibiru.fi/docs/community/"
---

# Community Hub | Nibiru

<template>
  <DocCards :boxes="boxesSocials" />
</template>

<script>
const boxesSocials = [
  { id: 1, title: "Join the Nibiru Collective", href: "../../community/",
    text: `Engage with the Nibiru Chain community, a globally distributed community of software developers, content creators, and Web3 enthusiasts.`},
]
export default {
  name: "CommunityCards",
  data() {
    return { boxesSocials } 
  }
}
</script>

<!-- This page has been moved to docs/community. 
This pointer page is here to prevent links from breaking.
-->
