<template>
  <div class="doc-hero">
    <a :href="box.href" class="hero__box" v-for="box in boxes" :key="box.id">
      <h2 class="hero__box__title">{{ box.title }}</h2>
      <p class="hero__box__body" v-if="box.text">{{ box.text }}</p>
    </a>
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

<style>
.doc-hero {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  padding: 10px 0px;
  gap: 10px;
  /* Controls the space between rectangles */
}

.hero__box {
  flex: 1 0 100%;

  @media (min-width: 640px) {
    /* sm */
    /* 
    flex-basis: 45%;
    */
  }

  @media (min-width: 768px) {
    /* md */
    min-height: 90px;
    /* Height of each rectangle */
    flex-basis: 45%;
  }

  height: auto;
  color: var(--color-text);
  /* Text color */
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 1.5rem 1rem;
  /* padding: 0.875rem 1.0rem; */
  text-decoration: none;
  /* Removes underline from links */
  border: 1px solid var(--color-sidebar-dim);
  border-radius: 20px;
  /* Optional: Adds rounded corners */
  box-shadow: 0 8px 15px 2px #565b731a;

  .hero__box__title,
  h2 {
    margin: 0;
    padding: 0;
    color: var(--color-link);
    font-size: 1.5rem;
    line-height: 1.1;
  }

  .hero__box__body,
  p {
    padding: 0.25rem 0;
    margin: 0;
    line-height: 1.3;
    font-size: 0.875rem;
    color: var(--color-text);
  }
}

.hero__box:hover {
  filter: brightness(1.8);
  cursor: pointer;
}
</style>
