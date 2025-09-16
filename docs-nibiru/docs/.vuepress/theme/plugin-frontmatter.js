/** This is a vuepress plugin that uses the Markdown frontmatter to configure the
 * corresponding HTML.
 *
 * See Vuepress Frontmatter documentation for more info on each field:
 * https://vuepress.vuejs.org/guide/frontmatter.html#predefined-variables
 * */
const matter = require("gray-matter")
const attrs = require("markdown-it-attrs")
const md = require("markdown-it")().use(attrs, {
  allowedAttributes: ["prereq", "hide", "synopsis"],
})
const cheerio = require("cheerio")

module.exports = (options = {}, context) => ({
  extendPageData($page) {
    let description = ""
    let frontmatter = {}
    try {
      const $ = cheerio.load(md.render($page._content))
      description = $("[synopsis]").text()
    } catch {
      console.log(
        `Error in processing description: $page.content is ${$page._content}`,
      )
    }
    try {
      frontmatter = matter($page._content, { delims: ["<!--", "-->"] }).data
    } catch {
      console.log(
        `Error in processing frontmatter: $page.content is ${$page._content}`,
      )
    }

    // <meta property="description" content="" />
    // <meta property="twitter:description" content="" />
    // <meta property="og:description" content="" />
    const meta = [
      { property: "og:description", content: description },
      { property: "twitter:description", content: description },
    ]
    $page.frontmatter = {
      description,
      meta,
      ...$page.frontmatter,
      ...frontmatter,
    }
    try {
      const tokens = md.parse($page._content, {})
      tokens.forEach((t, i) => {
        if (t.type === "heading_open" && ["h1"].includes(t.tag)) {
          $page.title = tokens[i + 1].content
          return
        }
      })
    } catch {
      console.log(`Error in processing headings.`)
    }
  },
})
