<template>
  <a href="/">
    <div
      class="img-fill"
      role="img"
      arial-label='A lost woman standing in front of a vast, futuristic city scape. The text reads, "You seem lost, fellow Nibirun. Let us take you home." Tapping the screen send you to the Nibiru documentation home page.'
    >
      <h1 class="invis">
        You seem lost, fellow Nibirun. Let us take you back home.
      </h1>
    </div>
  </a>
</template>

<style scoped>
img {
  width: 100%;
  margin-bottom: 1rem;
  margin-top: 1rem;
}

.img-fill {
  height: 100vh;
  background-image: url("./images/nibiru-lost-image.png");
  background-size: cover;
  background-position: center;
}

@media (max-width: 640px) {
  .img-fill {
    background-image: url("./images/nibiru-lost-image-mobile.png");
  }
}

.invis {
  visibility: hidden;
}
</style>
