const themes = require("vuepress/config")

/** GITHUB_REPO: Repo identifier without the GitHub prefix. Used to direct users
 * to submit issues. */
const GITHUB_REPO = "NibiruChain/website-help"

const metaImgUrl = "https://nibiru.fi/nibiru-meta-tag.png"

module.exports = themes.defineConfig4CustomTheme({
  globalLayout: "../theme/layouts/Layout.vue",
  title: "Nibiru",
  description:
    "Documentation on Nibiru, a breakthrough smart contract ecosystem and decentralized blockchain powering a Web3 hub with RWAs, DeFi, gas royalties, gaming, and more.",
  locales: {
    "/": {
      lang: "en-US",
    },
  },
  base: "/docs/",
  head: [
    [
      "script",
      {
        async: true,
        type: "text/javascript",
        src: "https://www.googletagmanager.com/gtm.js?id=GTM-MTZCQ3SH&l=dataLayer",
      },
    ],
    [
      "script",
      {
        async: true,
        type: "text/javascript",
        src: "https://static.klaviyo.com/onsite/js/klaviyo.js?company_id=TknCTU",
      },
    ],
    // NOTE that we don't need to set the following tags because they are configured
    // page-wise using the synopsis or header content:
    // <meta property="description" content="" />
    // <meta property="twitter:description" content="" />
    // <meta property="og:description" content="" />
    ["meta", { name: "og:type", content: "website" }],
    ["meta", { name: "og:locale", content: "en_US" }],
    [
      "meta",
      {
        name: "twitter:site",
        property: "twitter:site",
        content: "@NibiruChain",
      },
    ],
    [
      "meta",
      {
        name: "twitter:card",
        property: "twitter:card",
        content: "summary_large_image",
      },
    ],
    [
      "meta",
      { name: "twitter:image", content: metaImgUrl, property: "twitter:image" },
    ],
    ["meta", { name: "og:image", content: metaImgUrl, property: "og:image" }],

    [
      "link",
      {
        rel: "icon",
        type: "image/png",
        sizes: "64x64",
        href: "/favicon/favicon-64x64.png",
      },
    ],
    ["link", { rel: "manifest", href: "/site.webmanifest" }],
    ["meta", { name: "msapplication-TileColor", content: "#2e3148" }],
    [
      "meta",
      {
        name: "google-site-verification",
        content: "Uj8rxJHITFhFY8jBVBQfiPt9JcLl77JMkR50e9t4qGM",
      },
    ],
    ["meta", { name: "theme-color", content: "#ffffff" }],
    [
      "link",
      { rel: "icon", type: "image/svg+xml", href: "/favicon/favicon.svg" },
    ],
    [
      "link",
      { rel: "icon", type: "image/x-icon", href: "/favicon/favicon.ico" },
    ],
    [
      "link",
      {
        rel: "apple-touch-icon",
        sizes: "160x160",
        href: "/favicon/apple-touch-icon-160x160.png",
      },
    ],
    [
      "link",
      {
        rel: "apple-touch-icon-precomposed",
        href: "/favicon/apple-touch-icon-precomposed-100x100.png",
      },
    ],
    [
      "link",
      {
        rel: "stylesheet",
        href: "https://cdnjs.cloudflare.com/ajax/libs/KaTeX/0.5.1/katex.min.css",
      },
    ],
    [
      "link",
      {
        rel: "stylesheet",
        href: "https://cdn.jsdelivr.net/github-markdown-css/2.2.1/github-markdown.css",
      },
    ],
  ],
  markdown: {
    extendMarkdown: (md) => {
      md.use(require("markdown-it-katex"))
    },
  },
  themeConfig: {
    repo: GITHUB_REPO,
    docsRepo: GITHUB_REPO,
    docsDir: "docs",
    editLinks: true,
    label: "hub", // options: sdk, ibc, hub
    // TODO
    //algolia: {
    //  id: "BH4D9OD16A",
    //  key: "ac317234e6a42074175369b2f42e9754",
    //  index: "ibc-go"
    //},
    // Logo in the top left corner, file in .vuepress/public/
    logo: {
      src: "/docs/logo-main.svg",
    },
    // versions: [ // For mutliple versions of the docs.
    //   { "label": "main", "key": "main" },
    // ],
    topbar: {
      banner: true,
    },
    sidebar: {
      auto: false,
      nav: [
        {
          title: "Learn",
          children: [
            {
              title: "Overview",
              directory: false,
              path: "/",
            },
            {
              title: "Core Concepts",
              directory: true,
              path: "/concepts",
            },
            // {
            //   title: "Nibi-Indexer (GraphQL)", // TODO: Write-up
            //   directory: false,
            //   path: "https://nibiru.fi/blog/posts/008-mar23.html#:~:text=participating%20in%20consensus.-,Heart%20Monitor,-%E2%80%94%20A%20scalable%20indexing",
            // },
            {
              title: "Nibiru Architecture",
              children: [
                { title: "Nibiru Architecture", path: "/arch/" },
                { title: "NibiruBFT", path: "/arch/nibiru-bft/" },
                { title: "Execution Engine", path: "/arch/execution/" },
                { title: "Advanced", path: "/arch/advanced/" },
              ],
            },
            {
              title: "Nibiru Wasm",
              directory: true,
              path: "/ecosystem/wasm",
            },
            {
              title: "Nibiru EVM",
              children: [
                { title: "Nibiru EVM", path: "/evm/" },
                { title: "FunToken Mechanism", path: "/evm/funtoken" },
                { title: "Precompiles", path: "/evm/precompiles/" },
                { title: "Developer Guides", path: "/dev/evm/" },
                { title: "News on Nibiru EVM", path: "/evm/news" },
              ],
            },
          ],
        },
        {
          title: "User Hub",
          children: [
            {
              title: "Usage Guides",
              directory: true,
              children: [
                {
                  title: "All User Guides",
                  path: "/use/",
                  directory: false,
                },
                { title: "Web App", path: "https://app.nibiru.fi/stake" },
                {
                  title: "Create a Wallet",
                  path: "/wallets/",
                },
                {
                  title: "Guide: Importing a Token on MetaMask (EVM)",
                  path: "/use/import-token-evm.html",
                },
                {
                  title: "Guide: Liquid Staking on Nibiru (stNIBI)",
                  path: "/use/liquid-stake.html",
                },
                {
                  title: "Guide: Staking on Nibiru",
                  path: "/use/stake.html",
                },
                {
                  title: "Guide: Nibiru Bridge UI",
                  path: "/use/bridge.html",
                },
                {
                  title:
                    "Guide: Withdraw/Deposit with Centralized Exchanges (EVM)",
                  path: "/use/cex.html",
                },
              ],
            },
            {
              title: "Wallets",
              directory: true,
              path: "/wallets/",
            },
            {
              title: "Community",
              directory: true,
              path: "/community/",
            },
          ],
        },
        {
          title: "Developer Hub",
          path: "dev",
          children: [
            {
              title: "Developer Hub",
              directory: false,
              path: "/dev/",
            },
            {
              title: "Developer Tools",
              children: [
                {
                  title: "Nibiru CLI",
                  path: "/dev/cli",
                },
                {
                  title: "Guide: TypeScript SDK (NibiJS)",
                  path: "/dev/tools/nibijs",
                },
                { title: "Golang SDK", path: "/dev/tools/go-sdk" },
                {
                  title: "Golang Collections",
                  path: "/dev/tools/go-chain-state",
                },
                { title: "Python SDK", path: "/dev/tools/py-sdk" },
                { title: "Oracle Solutions", path: "/dev/tools/oracle" },
              ],
            },
            {
              title: "NibiJS: Building Apps with TypeScript",
              directory: true,
              path: "/dev/tools/nibijs",
            },
            {
              title: "Wasm Guides",
              directory: true,
              path: "/dev/cw",
            },
            {
              title: "EVM Guides",
              directory: true,
              path: "/dev/evm",
            },
          ],
        },
        // ---
        {
          title: "Special Topics",
          children: [
            {
              title: "Tokenomics (NIBI)",
              children: [
                {
                  title: "Tokenomics",
                  path: "/learn/tokenomics.html",
                },
                {
                  title: "Nibiru (NIBI) Token",
                  path: "/learn/nibi.html",
                },
                {
                  title: "Liquid Staked Nibiru (stNIBI)",
                  path: "/learn/liquid-stake/",
                },
                {
                  title: "How to Liquid Stake NIBI (Nibiru)",
                  path: "/use/liquid-stake.html",
                },
                {
                  title: "Staking Yield on Nibiru",
                  path: "/learn/staking.html",
                },
                {
                  title: "How to Stake NIBI (Nibiru)",
                  path: "/use/stake.html",
                },
                // TODO: Write page on delegated proof of stake here
              ],
            },
            {
              title: "Ecosystem Updates",
              directory: false,
              path: "/ecosystem/updates",
            },
            {
              title: "Roadmap - Lagrange Point",
              children: [
                {
                  title: "Nibiru Roadmap (2025)",
                  path: "/ecosystem/future",
                },
                {
                  title: "Sai Perps DEX",
                  path: "/ecosystem/apps/sai-fun-perps",
                },
                // {
                //   title: "Nibiru USD (NUSD)",
                //   path: "/ecosystem/future/nusd",
                // },
              ],
            },
            {
              title: "Common Questions",
              directory: true,
              path: "/learn/faq/",
            },
          ],
        },
        // ---
        {
          title: "Full Nodes",
          children: [
            {
              title: "Full Nodes",
              directory: true,
              path: "/run-nodes/full-nodes",
            },
            {
              title: "Validator Nodes",
              directory: true,
              path: "/run-nodes/validators",
            },
            // TODO: Previsouly feature app section
            // {
            //   title: "Nibiru Web App",
            //   path: "https://app.nibiru.fi/stake",
            // },
            // {
            //   title: "Featured dApps",
            //   directory: true,
            //   path: "/ecosystem/",
            //   children: [
            //     {
            //       title: "Bridge App (Squid)",
            //       path: "https://app.squidrouter.com/?chains=1%2Ccataclysm-1&tokens=0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48%2Cunibi",
            //     },
            //     {
            //       title: "Bridge App (Swing)",
            //       path: "https://app.nibiru.fi/bridge",
            //     },
            //     { title: "Coded Estate", path: "https://codedestate.com" },
            //     {
            //       title: "Nibiru (NIBI) Token",
            //       directory: false,
            //       path: "/learn/nibi.html",
            //     },
            //     {
            //       title: "How to Stake NIBI (Nibiru)",
            //       directory: false,
            //       path: "/use/stake.html",
            //     },
            //   ],
            // },
          ],
        },
        // ---
      ],
    },

    gutter: {
      title: "Help & Support",
      editLink: true,
      chat: {
        title: "Discord",
        text: "Chat with Nibiru developers on Discord.",
        url: "https://discord.gg/HFvbn7Wtud",
        bg: "linear-gradient(225.11deg, #2E3148 0%, #161931 95.68%)",
      },
      github: {
        title: "Found an Issue?",
        text: "Help us improve this page by suggesting edits on GitHub.",
        url: `https://github.com/${GITHUB_REPO}`,
      },
    },

    footer: {
      logo: "/docs/logo-main.svg",
      question: {
        text: "Chat with Nibiru developers on <a href='https://discord.gg/HFvbn7Wtud' target='_blank'>Discord</a>.",
      },
      textLink: {
        text: "nibiru.fi",
        url: "https://nibiru.fi",
      },
      services: [
        {
          service: "twitter",
          url: "https://twitter.com/NibiruChain",
        },
        {
          service: "linkedin",
          url: "https://www.linkedin.com/company/nibiruchain",
        },
        {
          service: "github",
          url: "https://github.com/NibiruChain",
        },
        {
          service: "discord",
          url: "https://discord.gg/HFvbn7Wtud",
        },
        // {
        //   service: "reddit",
        //   url: "https://reddit.com/r/cosmosnetwork"
        // },
        {
          service: "telegram",
          url: "https://t.me/nibiruchain",
        },
        {
          service: "youtube",
          url: "https://www.youtube.com/@nibiruchain",
        },
      ],
      /** smallprint: You can use both HTML and MD here. */
      smallprint: `<a href="https://nibiru.fi/terms-of-service.pdf" target="_blank">Terms and Conditions</a> | <a href="https://nibiru.fi/privacy-policy-nibiru.pdf" target="_blank">Privacy Policy</a>`,
      links: [
        {
          title: "Using Nibiru",
          children: [
            {
              title: "User Guides",
              url: "https://nibiru.fi/docs/use/",
            },
            {
              title: "Create Nibiru Wallet",
              url: "https://nibiru.fi/docs/wallets/",
            },
            {
              title: "Guides: Staking on Nibiru",
              url: "https://nibiru.fi/docs/use/stake.html",
            },
            {
              title: "Guides: Nibiru Bridges",
              url: "https://nibiru.fi/docs/use/bridge.html",
            },
          ],
        },
        {
          title: "Community",
          children: [
            {
              title: "Brand Kit",
              url: "https://nibiru.fi/brand",
            },
            {
              title: "Community Hub",
              url: "https://nibiru.fi/docs/community/",
            },
            {
              title: "Blossom: Ambassador Program",
              url: "https://nibiru.fi/blog/posts/059-blossom-ambassador-program.html",
            },
            {
              title: "Nibiru Blog",
              url: "https://nibiru.fi/blog",
            },
          ],
        },
        {
          title: "Developers",
          children: [
            {
              title: "Developer Hub",
              url: `https://nibiru.fi/docs/dev`,
            },
            {
              title: "Source code on GitHub",
              url: `https://github.com/NibiruChain/nibiru`,
            },
          ],
        },
      ],
    },
  },
  plugins: [
    [
      // npm: vuepress-plugin-sitemap -> dist/sitemap.xml
      "sitemap",
      {
        hostname: "https://nibiru.fi/docs",
      },
    ],
  ],
})
