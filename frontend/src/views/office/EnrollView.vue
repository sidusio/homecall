<script setup lang="ts">
import { ref } from 'vue'
import QrcodeVue from 'qrcode.vue'
import { officeClient } from '@/clients'

const deviceName = ref('')
const usersName = ref('')
const autoAnswer = ref(false)
const autoAnswerDelay = ref(0)

const enrollmentKey = ref('')

const enrollDevice = async (e: Event) => {
    e.preventDefault()

    const res = await officeClient.enrollDevice({
        name: deviceName.value,
        settings: {
            usersName: usersName.value,
            autoAnswer: autoAnswer.value,
            autoAnswerDelaySeconds: BigInt(autoAnswerDelay.value)
        }
    })

    // Set the enrollment key from the response.
    enrollmentKey.value = res.enrollmentKey
    console.log(enrollmentKey.value)
}
</script>

<template>
  <main class="enroll-device">
    <h1 class="enroll-device__title">
        Registrera enhet
    </h1>

    <form v-if="!enrollmentKey">
        <input type="text" placeholder="Device Name" v-model="deviceName" />
        <input type="text" placeholder="User's Name" v-model="usersName" />
        <label for="autoanswer">
            <input id="autoanswer" type="checkbox" v-model="autoAnswer" />
            Auto Answer
        </label>
        <input type="number" placeholder="Auto Answer Delay" v-model="autoAnswerDelay" />
        <button @click="enrollDevice">Enroll Device</button>
    </form>

    <div v-else>
        <p class="enroll-device__desc">
            Skanna QR-koden med enheten.
        </p>
        <qrcode-vue
            :value="enrollmentKey"
            :size="300"
            level="H"
        />
    </div>
  </main>
</template>

<style lang="scss" scoped>
.enroll-device {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100vh;

    &__desc {
        margin-bottom: 2rem;
        text-align: center;
    }

    form {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: .5rem;
    }
}
</style>
