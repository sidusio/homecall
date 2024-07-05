<script setup lang="ts">
import GlobalHeader from '@/components/GlobalHeader.vue';
import { useRoute } from 'vue-router';

const isDeviceRoute = () => {
  const route = useRoute();
  return route.path.includes('device');
};
</script>

<template>
  <div>
    <GlobalHeader v-if="!isDeviceRoute()" />

    <router-view />
  </div>
</template>

<style lang="scss" scoped>
#meeting {
  width: 100%;
  height: 100%;
}
</style>
