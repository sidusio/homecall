<script setup lang="ts">
// There doesn't seem to be any TS support for Jitsi Meet.
import { onMounted } from 'vue';

onMounted(() => {
  const api = new JitsiMeetExternalAPI('8x8.vc', {
    roomName: import.meta.env.VITE_JITSI_ROOM_ID,
    parentNode: document.querySelector('#meeting'),
    height: '100vh',
  })
});
</script>

<template>
  <div>
    <div id="meeting"></div>
  </div>
</template>

<style scoped>
#meeting {
  width: 100%;
  height: 100%;
}
</style>
