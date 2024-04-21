<script setup lang="ts">
import { deviceClient } from '@/clients';
import { useRouter } from 'vue-router';
import * as jose from 'jose'

import { QrcodeStream } from 'vue-qrcode-reader'

const router = useRouter()

const onDetect = async (content: Array<any>) => {
  // Create key pair for the device.
  const keyPair = await window.crypto.subtle.generateKey(
    {
      name: 'RSA-OAEP',
      modulusLength: 2048,
      publicExponent: new Uint8Array([1, 0, 1]),
      hash: 'SHA-256',
    },
    true,
    ['encrypt', 'decrypt']
  )

  // Export the public key.
  const publicKeyExport = await jose.exportSPKI(keyPair.publicKey)
  const privateKeyExport = await jose.exportPKCS8(keyPair.privateKey)

  // Enroll the device.
  const res = await deviceClient.enroll({
    publicKey: publicKeyExport,
    enrollmentKey: content[0].rawValue,
  })

  // Save private key and deviceid in local storage.
  localStorage.setItem('privateKey', privateKeyExport)
  localStorage.setItem('deviceId', res.deviceId)
  localStorage.setItem('settings', JSON.stringify(res.settings))

  // Redirect to the device view.
  router.push('/device')
}
</script>

<template>
  <main class="enroll-device">
    <div class="enroll-device__qrcode">
      <p class="enroll-device__heading">
        Skanna QR-koden för att registrera enheten.
      </p>
      <QrcodeStream
        class="enroll-device__qrcode-stream"
        @detect="onDetect"
      />
    </div>
  </main>
</template>

<style lang="scss" scoped>
.enroll-device {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100vh;


  &__qrcode {
    height: 100%;
    width: 100%;
    position: relative;
  }

  &__heading {
    z-index: 100;
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    padding: 1rem;
    color: black;
    background-color: white;
    text-align: center;
    font-size: 1rem;
  }

  &__registered {
    text-align: center;

    h1, p, a {
      display: block;
    }

    p {
      font-size: 1rem;
    }
  }

  &__back {
    margin-top: 3rem;
    font-size: 1.25rem;
    color: white;
    background-color: rgb(0, 122, 2);
    text-decoration: none;
    padding: 1rem 2rem;
    border-radius: 30px;
  }
}
</style>
