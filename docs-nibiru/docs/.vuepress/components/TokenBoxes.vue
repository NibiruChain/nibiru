<template>
  <div class="doc-token-boxes">
    <div :href="box.href" class="token-box" v-for="box in boxes" :key="box.id">
      <a :href="box.href">
        <h2 class="token-box__title">
          <img v-if="box.icon" :src="box.icon" alt="" class="hero__icon" />
          {{ box.title }}
        </h2>
      </a>
      <p class="token-box__body" v-if="box.text">{{ box.text }}</p>
      <div class="token-box__address" v-if="box.address">
        <code>{{ box.address }}</code>
        <!-- <span class="copy-icon">📋</span> -->
      </div>
    </div>
  </div>
</template>

<script>
export default {
  props: {
    boxes: {
      type: Array,
      required: true,
    },
  },
}
</script>

<style scoped>
.doc-token-boxes {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 10px 0;
}

.token-box {
  width: 100%;
  color: var(--color-text);
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 1.5rem 1rem;
  text-decoration: none;
  /* Removes underline from links */
  border: 1px solid var(--color-sidebar-dim);
  border-radius: 20px;
  box-shadow: 0 8px 15px 2px #565b731a;
}

h2.token-box__title {
  display: flex;
  align-items: center;
  gap: 0.5rem;

  margin: 0;
  padding: 0;
  color: var(--color-link);
  font-size: 1.5rem;
  line-height: 1.1;
}

img.hero__icon {
  width: 32px;
  height: 32px;
  margin: 0;
  object-fit: contain;
}

.token-box__body,
p {
  padding: 0.25rem 0;
  margin: 0;
  line-height: 1.3;
  font-size: 0.875rem;
  color: var(--color-text);
}

.token-box__address {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  word-break: break-all;
}

.token-box__address code {
  margin: 0;
}

/* .copy-icon { */
/*   font-size: 1.2rem; */
/*   opacity: 0.6; */
/*   cursor: pointer; */
/* } */
</style>
