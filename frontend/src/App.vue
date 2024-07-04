<script setup lang="ts">
import GlobalHeader from '@/components/GlobalHeader.vue';
</script>

<template>
  <div>
    <GlobalHeader />

    <router-view />
  </div>
</template>

<style scoped>
#meeting {
  width: 100%;
  height: 100%;
}
</style>
