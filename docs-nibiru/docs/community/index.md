---
order: 1
canonicalUrl: "https://nibiru.fi/docs/community/"
metaTitle: "Community Hub | Nibiru Chain"
footer:
  newsletter: false
---

# Nibiru Community

Engage with the Nibiru Chain community, a globally distributed community of software developers, content creators, and Web3 enthusiasts. {synopsis}

## Main Community Channels

<template>
  <DocCards :boxes="boxesSocials" />
</template>

<script>
const boxesSocials = [
  { id: 1, title: "Brand Kit", href: "https://nibiru.fi/brand",
    text: `Official creative assets for Nibiru Chain's brand identity.`},
  { id: 2, title: "Twitter / X", href: "https://twitter.com/NibiruChain",
    text: `Official announcements, Ask Me Anything (AMAs) sessions, and upcoming news/releases.` },
  { id: 3, title: "Blog", href: "https://nibiru.fi/blog/",
    text: `Official Nibiru blog.`},
  { id: 4, title: "YouTube", href: "https://www.youtube.com/@nibiruchain",
    text: `Level up your knowledge with Nibiru's library of video and audio content.`},
  { id: 5, title: "Discord", href: "https://discord.gg/nibirufi",
    text: `Chat with core contributors, developers, and community members.`},
  { id: 6, title: "Telegram", href: "https://t.me/NibiruChain",
    text: `Official Nibiru Telegram chat.`},
]
const boxesBuilders = [
  { id: 1, title: "Developer Hub", href: "https://nibiru.fi/docs/dev/",
    text: `Everything you need to build on Nibiru. Your go-to hub to develop smart contracts applications for the decentralized web.`},
  { id: 7, title: "Codebase (GitHub)", href: "https://github.com/NibiruChain",
    text: `Source code of the Nibiru blockchain, smart contracts, tools, and SDKs (Rust, TypeScript, Golang, Python).`},
  { id: 2, title: "Blog (For Devs)", href: "https://nibiru.fi/blog/tags/for%20devs",
    text: `Developer-focused tutorials and blog content on Nibiru and all things
Web3.`},
  { id: 3, title: "Hackathons", href: "https://t.me/nibiruhackathon",
    text: `Nibiru hackathons give ambitious founders and developers a platform to collaborate with other buidlers and bring innovative ideas to life. `},
  { id: 4, title: "Requests for Protocols (RFPs)", href: "https://nibiru.notion.site/9365f31b339f4ce69ac25d88dd519690?v=f3483f760045490bb463a75ac091dad5&pvs=4",
    text: `Thorough specifications outlining desired Web3 initiatives and applications in the Nibiru Ecosystem.`},
  { id: 5, title: "Nibiru Grants", href: "https://nibiru.fi/ecosystem/grants",
    text: `We welcome skilled developers passionate about enriching the Nibiru ecosystem to apply. Successful candidates may earn milestone-based rewards in addition to accelerator-like support for engineering, fundraising, and marketing.`},
]
export default {
  name: "CommunityCards",
  data() {
    return { boxesSocials, boxesBuilders } 
  }
}
</script>

## Resources for Builders

<template>
  <DocCards :boxes="boxesBuilders" />
</template>

## Other Resources

| | |
| --- | --- |
| [Careers](https://jobs.lever.co/nibiru) | Explore career opportunities to work on the core entities contributing to the development of Nibiru Chain. |
| [TikTok](https://www.tiktok.com/@nibiruchain) | [tiktok.com/@nibiruchain](https://www.tiktok.com/@nibiruchain) |
| [Instagram](https://www.instagram.com/nibiruchainofficial) | [instagram.com/nibiruchainofficial](https://www.instagram.com/nibiruchainofficial) |
| [LinkedIn](https://www.linkedin.com/company/nibiruchain) | [linkedin.com/company/nibiruchain](https://www.linkedin.com/company/nibiruchain) |
| [Contact Us](https://docs.google.com/forms/d/e/1FAIpQLSfstYs9Gkvcw3yW7ivYHP1rPV3ifCCCHsvwOWSN3tNBhjpwkA/viewform) | Get in direct contact with the Nibiru development team. |
