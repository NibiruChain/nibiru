<template>
  <div v-if="$themeConfig.footer">
    <div class="wrapper">
      <div class="container">
        <div class="footer__wrapper">
          <div class="questions" v-if="!full && !$themeConfig.custom">
            <div class="questions__wrapper">
              <h1 class="questions__h1">Questions?</h1>
              <p class="questions__p" v-if="$themeConfig.footer && $themeConfig.footer.question.text"
                v-html="md($themeConfig.footer.question.text)"></p>
            </div>
            <tm-newsletter-form v-if="$themeConfig.footer"></tm-newsletter-form>
          </div>
          <div class="links" v-if="$themeConfig.footer && $themeConfig.footer.links && full">
            <div class="links__item" v-for="item in $themeConfig.footer.links" :key="item.title">
              <div class="links__item__title">{{ item.title }}</div>
              <a class="links__item__link" v-for="link in item.children" v-if="link.title && link.url" :href="link.url"
                rel="noreferrer noopener" target="_blank">{{ link.title }}</a>
            </div>
          </div>
          <div class="logo">
            <div class="logo__item">
              <a class="logo__image" :href="$themeConfig.footer.textLink.url" target="_blank" rel="noreferrer noopener">
                <component :is="`logo-${$themeConfig.label}-text`" v-if="$themeConfig.label" fill="black"></component>
                <img v-else-if="$themeConfig.custom" :src="$themeConfig.footer.logo" />
              </a>
            </div>
            <div class="logo__item logo__link" v-if="$themeConfig.footer && $themeConfig.footer.services">
              <a class="smallprint__item__links__item" v-for="item in $themeConfig.footer.services" :href="item.url"
                target="_blank" :title="item.service" rel="noreferrer noopener">
                <!-- If isSvg is true, render a div with text "is" -->
                <component v-if="serviceIcon(item.service).isSvg" :is="serviceIcon(item.service).icon"
                  class="social-icon" />

                <!-- Else, render the SVG block -->
                <svg v-else width="24" height="24" xmlns="http://www.w3.org/2000/svg" fill-rule="evenodd"
                  clip-rule="evenodd" fill="#aaa">
                  <path :d="serviceIcon(item.service).icon"></path>
                </svg>
              </a>
            </div>
          </div>
          <div class="smallprint" v-if="$themeConfig.footer">
            <div class="smallprint__item smallprint__item__links">
              <a v-if="$themeConfig.footer &&
                $themeConfig.footer.textLink &&
                $themeConfig.footer.textLink.text &&
                $themeConfig.footer.textLink.url
                " :href="$themeConfig.footer.textLink.url">{{ $themeConfig.footer.textLink.text }}</a>
            </div>
            <div class="smallprint__item__desc smallprint__item"
              v-if="$themeConfig.footer && $themeConfig.footer.smallprint" v-html="md($themeConfig.footer.smallprint)">
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style lang="stylus" scoped>
.container
  background-color var(--color-bg)
  color var(--color-text, black)
  padding-top 3.5rem
  padding-bottom 3.5rem

.wrapper
  --height 50px
  background var(--color-bg)

.questions
  display grid
  grid-template-columns 1fr 1fr
  margin-bottom 3rem
  column-gap 10%
  margin-right 10%
  align-items flex-start

  & >>> a[href]
    color var(--color-link, #ccc)

  &__wrapper
    margin-bottom 2rem

  &__h1
    font-size 1.5rem
    line-height 2rem
    margin-bottom 0.5rem
    font-weight 600
    color var(--color-text, black)

  &__p
    font-size 0.875rem
    color rgba(22, 25, 49, 0.9)
    line-height 1.25rem

.links
  display grid
  grid-template-columns repeat(auto-fit, minmax(250px, 1fr))

  &__item
    display flex
    flex-direction column
    margin 1.5rem 0

    &__title
      font-size 0.75rem
      letter-spacing 0.2em
      text-transform uppercase
      font-weight 700
      margin-bottom 1rem

    &__link
      font-size 0.875rem
      line-height 1.25rem
      margin-top 0.5rem
      margin-bottom 0.5rem
      align-self flex-start
      color var(--color-text-dim, inherit)

      &:hover,
      &:focus
        color var(--color-link, inherit)

.footer__wrapper
  margin 0 auto
  // padding-left 2.5rem
  // padding-right 0.5rem

.social-icon
  width 36px
  height 36px

.logo
  display grid
  grid-template-columns repeat(auto-fit, minmax(200px, 1fr))

  &__item
    padding 1.5rem 0
    display flex
    align-items flex-start
    justify-content flex-start

  &__image
    display inline-block
    min-height 2rem
    max-height 3rem
    max-width 12.5rem
    cursor pointer

    img
      max-height 100%
      max-width 100%

  &__link
    grid-column span 2
    font-weight 600
    align-items center

.smallprint
  display grid
  grid-template-columns repeat(auto-fit, minmax(200px, 1fr))
  align-items flex-end

  & >>> a[href]
    color var(--color-link, #ccc)

  &__item
    padding 1rem 0
    font-weight 600

    &__links
      color var(--color-link)
      font-size 0.875rem

      &__item
        margin-right 1rem

        svg
          transition fill .15s ease-out
          fill var(--color-link)

        &:hover svg,
        &:focus svg
          transition filter 0.3s ease
          filter brightness(115%)

    &__desc
      grid-column span 2
      font-size 0.8125rem
      line-height 1rem
      font-weight normal
      color var(--color-text-dim)

@media screen and (max-width: 732px)
  .questions
    display block
    margin-right 0

@media screen and (max-width: 480px)
  .footer__links
    margin-left 1.5rem
    margin-right 1.5rem
</style>

<script>
import { find } from "lodash"
import SocialLinkedIn from "./socials/LinkedIn.vue"
import SocialDiscord from "./socials/Discord.vue"
import SocialGitHub from "./socials/GitHub.vue"
import SocialTwitter from "./socials/Twitter.vue"
import SocialYouTube from "./socials/YouTube.vue"
import SocialTelegram from "./socials/Telegram.vue"

export default {
  props: ["tree", "full"],
  components: {
    SocialLinkedIn,
    SocialDiscord,
    SocialGitHub,
    SocialTwitter,
    SocialYouTube,
    SocialTelegram,
  },
  methods: {
    logItem(item) {
      console.debug("item %o", item)
    },
    serviceIcon(service) {
      // icons from https://iconmonstr.com
      const icons = [
        {
          service: "github",
          isSvg: true,
          icon: "SocialGitHub",
        },
        {
          service: "discord",
          isSvg: true,
          icon: "SocialDiscord",
        },
        {
          service: "medium",
          icon: "M24 24h-24v-24h24v24zm-4.03-5.649v-.269l-1.247-1.224c-.11-.084-.165-.222-.142-.359v-8.998c-.023-.137.032-.275.142-.359l1.277-1.224v-.269h-4.422l-3.152 7.863-3.586-7.863h-4.638v.269l1.494 1.799c.146.133.221.327.201.523v7.072c.044.255-.037.516-.216.702l-1.681 2.038v.269h4.766v-.269l-1.681-2.038c-.181-.186-.266-.445-.232-.702v-6.116l4.183 9.125h.486l3.593-9.125v7.273c0 .194 0 .232-.127.359l-1.292 1.254v.269h6.274z",
          isSvg: false,
        },
        {
          service: "twitter",
          icon: "M24 4.557c-.883.392-1.832.656-2.828.775 1.017-.609 1.798-1.574 2.165-2.724-.951.564-2.005.974-3.127 1.195-.897-.957-2.178-1.555-3.594-1.555-3.179 0-5.515 2.966-4.797 6.045-4.091-.205-7.719-2.165-10.148-5.144-1.29 2.213-.669 5.108 1.523 6.574-.806-.026-1.566-.247-2.229-.616-.054 2.281 1.581 4.415 3.949 4.89-.693.188-1.452.232-2.224.084.626 1.956 2.444 3.379 4.6 3.419-2.07 1.623-4.678 2.348-7.29 2.04 2.179 1.397 4.768 2.212 7.548 2.212 9.142 0 14.307-7.721 13.995-14.646.962-.695 1.797-1.562 2.457-2.549z",
          isSvg: true,
          icon: "SocialTwitter",
        },
        {
          service: "linkedin",
          isSvg: true,
          icon: "SocialLinkedIn",
        },
        {
          service: "reddit",
          icon: "M14.238 15.348c.085.084.085.221 0 .306-.465.462-1.194.687-2.231.687l-.008-.002-.008.002c-1.036 0-1.766-.225-2.231-.688-.085-.084-.085-.221 0-.305.084-.084.222-.084.307 0 .379.377 1.008.561 1.924.561l.008.002.008-.002c.915 0 1.544-.184 1.924-.561.085-.084.223-.084.307 0zm-3.44-2.418c0-.507-.414-.919-.922-.919-.509 0-.923.412-.923.919 0 .506.414.918.923.918.508.001.922-.411.922-.918zm13.202-.93c0 6.627-5.373 12-12 12s-12-5.373-12-12 5.373-12 12-12 12 5.373 12 12zm-5-.129c0-.851-.695-1.543-1.55-1.543-.417 0-.795.167-1.074.435-1.056-.695-2.485-1.137-4.066-1.194l.865-2.724 2.343.549-.003.034c0 .696.569 1.262 1.268 1.262.699 0 1.267-.566 1.267-1.262s-.568-1.262-1.267-1.262c-.537 0-.994.335-1.179.804l-2.525-.592c-.11-.027-.223.037-.257.145l-.965 3.038c-1.656.02-3.155.466-4.258 1.181-.277-.255-.644-.415-1.05-.415-.854.001-1.549.693-1.549 1.544 0 .566.311 1.056.768 1.325-.03.164-.05.331-.05.5 0 2.281 2.805 4.137 6.253 4.137s6.253-1.856 6.253-4.137c0-.16-.017-.317-.044-.472.486-.261.82-.766.82-1.353zm-4.872.141c-.509 0-.922.412-.922.919 0 .506.414.918.922.918s.922-.412.922-.918c0-.507-.413-.919-.922-.919z",
          isSvg: false,
        },
        {
          service: "telegram",
          isSvg: true,
          icon: "SocialTelegram",
        },
        {
          service: "youtube",
          isSvg: true,
          icon: "SocialYouTube",
        },
        {
          service: "unknown_service",
          icon: "M19.615 3.184c-3.604-.246-11.631-.245-15.23 0-3.897.266-4.356 2.62-4.385 8.816.029 6.185.484 8.549 4.385 8.816 3.6.245 11.626.246 15.23 0 3.897-.266 4.356-2.62 4.385-8.816-.029-6.185-.484-8.549-4.385-8.816zm-10.615 12.816v-8l8 3.993-8 4.007z",
          isSvg: false,
        },
      ]
      const knownService = icons.filter((s) => {
        return (
          s.service.toLowerCase().match(service.toLowerCase()) ||
          service.toLowerCase().match(s.service.toLowerCase())
        )
      })[0]
      const defaultIcon = find(icons, ["service", "unknown_service"])
      return knownService || defaultIcon
    },
  },
}
</script>
