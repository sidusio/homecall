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
}
</script>

<template>
  <main class="enroll-device">
    <h1>Enroll Device</h1>

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

    <qrcode-vue
        v-else
        :value="enrollmentKey"
        :size="300"
        level="H"
    />
  </main>
</template>

<style lang="scss" scoped>
.enroll-device {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100vh;

    form {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: .5rem;
    }
}
</style>
