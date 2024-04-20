<script setup lang="ts">
import { deviceClient } from '@/clients';
import { computed, watch, ref } from 'vue'
import * as jose from 'jose'

const privateKey = computed(() => {
  return localStorage.getItem('privateKey')
})

const deviceId = computed(() => {
  return localStorage.getItem('deviceId')
})

watch(privateKey || deviceId, async () => {
  if(!privateKey.value || !deviceId.value) return

  const alg = 'RS256'
  const privateKeyImport = await jose.importPKCS8(privateKey.value, alg)
  const jwt = await new jose.SignJWT()
    .setProtectedHeader({ alg })
    .setIssuedAt()
    .setIssuer('homecall-device')
    .setAudience('homecall')
    .setSubject(deviceId.value)
    .setExpirationTime('2h')
    .sign(privateKeyImport)

  const resultStream = deviceClient.waitForCall({
    deviceId: deviceId.value
  }, {
    headers: {
      Authorization: `Bearer ${jwt}`
    }
  })

  for await (const res of resultStream) {
    const api = new JitsiMeetExternalAPI('8x8.vc', {
      roomName: res.jitsiRoomId,
      jwt: res.jitsiJwt,
      parentNode: document.querySelector('#meeting'),
      height: '100vh',
      configOverwrite: {
        prejoinConfig: {
          enabled: false
        },
        toolbarButtons: [],
        toolbarConfig: {
          alwaysVisible: true,
        },
        hideConferenceSubject: true,
        hideConferenceTimer: true,
        filmstrip: {
          disableResizable: true,
        }
      },
      interfaceConfigOverwrite: {
        MOBILE_APP_PROMO: false,
      }
    });

    api.addListener('participantLeft', () => {
      api.dispose()

      // Reload the page
      location.reload()
    });
  }
}, { immediate: true })

</script>

<template>
  <main>
    <h1>Client Home</h1>
    <div id="meeting"></div>
  </main>
</template>
