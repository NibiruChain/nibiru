<template>
  <div style="height: 100%; position: relative">
    <div class="container">
      <a
        href="https://nibiru.fi/"
        v-if="!(compact === true)"
        class="logo__container"
        target="_blank"
      >
        <div class="logo">
          <div
            class="logo__img__custom"
            v-if="$themeConfig.logo && $themeConfig.logo.src"
          >
            <img :src="$themeConfig.logo.src" />
          </div>
          <div class="logo__img" v-else>
            <component :is="`logo-${$themeConfig.label || 'sdk'}`"></component>
          </div>
          <div class="logo__text" v-if="!$themeConfig.logo">
            {{ $site.title || "Documentation" }}
          </div>
        </div>
      </a>
      <div :class="[`footer__compact__${!!(compact === true)}`]" class="items">
        <div
          v-for="item in value"
          :style="{
            display:
              $themeConfig.sidebar.auto == false && item.title === ''
                ? 'none'
                : 'block',
          }"
          class="sidebar"
        >
          <div class="title">{{ item.title }}</div>
          <client-only>
            <tm-sidebar-tree
              :value="item.children"
              v-if="item.children"
              :tree="tree"
              :level="0"
              class="section"
            ></tm-sidebar-tree>
          </client-only>
        </div>
        <div class="sidebar version">
          <tm-select-version></tm-select-version>
        </div>
      </div>
      <div
        :class="[`footer__compact__${!!(compact === true)}`]"
        class="footer"
        v-if="!$themeConfig.custom"
      >
        <a
          :href="product.url"
          target="_blank"
          rel="noreferrer noopener"
          v-for="product in products"
          :style="{ '--color': product.color }"
          v-if="$themeConfig.label != product.label"
          class="footer__item"
        >
          <component
            :is="`tm-logo-${product.label}`"
            class="footer__item__icon"
          ></component>
          <div class="footer__item__title" v-html="md(product.name)"></div>
        </a>
      </div>
    </div>
  </div>
</template>

<style scoped>
.container {
  display: flex;
  flex-direction: column;
  height: 100%;
}
.logo {
  padding: 1.5rem 2rem;
  display: flex;
  align-items: center;
}
.logo:active {
  outline: none;
}
.logo__img {
  width: 2.5rem;
  height: 2.5rem;
  margin-right: 0.75rem;
}
.logo__img__custom {
  width: 100%;
  height: 2.5rem;
  margin-right: 0.75rem;
}
.logo__img__custom img {
  max-width: 100%;
  max-height: 100%;
}
.logo__text {
  font-weight: 600;
}
.logo__container {
  position: sticky;
  display: block;
  background-color: var(--color-bg);
  z-index: 1;
  top: 0;
}
.logo__container:after {
  position: absolute;
  content: "";
  top: 100%;
  left: 0;
  right: 0;
  background-color: var(--color-bg);
  height: 25px;
}
.sidebar {
  padding-left: 2rem;
  padding-right: 2rem;
  overflow-x: hidden;
  background-color: var(--color-bg);
  color: var(--color-text);
}
.version {
  margin-top: 2rem;
  display: none;
}
.items {
  flex-grow: 1;
  padding-bottom: 2rem;
}
.items.footer__compact__true {
  flex-grow: 0;
}
.title {
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.2em;
  color: var(--color-text-dim);
  margin-top: 2rem;
  margin-bottom: 0.5rem;
}
.footer.footer__compact__true {
  padding-bottom: 150px;
  bottom: initial;
  margin-top: 0;
  position: relative;
  flex-grow: 1;
}
.footer {
  height: var(--sidebar-footer-height);
  padding-top: 1rem;
  padding-bottom: 1rem;
  background-color: var(--sidebar-bg);
  position: sticky;
  bottom: 0;
  width: 100%;
  display: grid;
  grid-auto-flow: column;
  padding-left: 0.75rem;
  padding-right: 0.75rem;
  align-items: center;
  grid-auto-columns: 1fr;
}
.footer:before {
  content: "";
  position: absolute;
  top: -50px;
  left: 0;
  right: 0;
  bottom: 100%;
  pointer-events: none;
}
.footer__item {
  align-self: flex-start;
  display: flex;
  align-items: center;
  flex-direction: column;
  fill: var(--color-sidebar-dim);
}
.footer__item__icon {
  height: 32px;
  margin-bottom: 0.25rem;
}
.footer__item:hover {
  fill: var(--color);
}
.footer__item__title {
  text-align: center;
  font-size: 0.6875rem;
  line-height: 0.875rem;
}
@media screen and (max-width: 1135px) {
  .version {
    display: block;
  }
}
</style>

<script>
import {
  includes,
  isString,
  isPlainObject,
  isArray,
  sortBy,
  last,
  find,
  omit,
} from "lodash"

export default {
  props: ["value", "tree", "compact"],
  data: function () {
    return {
      search: {
        query: null,
      },
      products: [
        {
          label: "sdk",
          name: "Nibiru<br>SDKs",
          url: "https://nibiru.fi/docs/dev/",
          color: "#04D9D9",
        },
        {
          label: "hub",
          name: "Cosmos<br>Hub",
          url: "https://hub.cosmos.network/",
          color: "#04D9D9",
        },
        {
          label: "ibc",
          name: "Community<br>Hub",
          url: "https://nibiru.fi/docs/ecosystem/community/",
          color: "#04D9D9",
        },
        {
          label: "core",
          name: "Core Concepts",
          url: "https://nibiru.fi/docs/ecosystem/concepts/consensus.html",
          color: "#04D9D9",
        },
      ],
    }
  },
  computed: {
    searchResults() {
      return this.$site.pages.filter((page) => {
        const headers = page.headers ? page.headers.map((h) => h.title) : []
        const title = page.title
        return (
          title &&
          [title, ...headers]
            .join(" ")
            .toLowerCase()
            .match(this.search.query.toLowerCase())
        )
      })
    },
    logo() {
      return this.$themeConfig.logo
    },
    sidebar() {
      return this.value
    },
  },
}
</script>
