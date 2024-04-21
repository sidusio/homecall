<script setup lang="ts">
import { ref } from 'vue'
import QrcodeVue from 'qrcode.vue'
import { officeClient } from '@/clients'
import { useRoute, useRouter } from 'vue-router';

const router = useRouter()

const deviceName = ref('')
const autoAnswer = ref(false)
const autoAnswerDelay = ref(0)

const enrollmentKey = ref('')
const showContinueBtn = ref(false)

const enrollDevice = async (e: Event) => {
    e.preventDefault()

    const res = await officeClient.createDevice({
        name: deviceName.value,
        defaultSettings: {
            autoAnswer: autoAnswer.value,
            autoAnswerDelaySeconds: BigInt(autoAnswerDelay.value)
        }
    })

    // Set the enrollment key from the response.
    enrollmentKey.value = res.device?.enrollmentKey || ''

    if(enrollmentKey.value) {
        showContinueBtn.value = true
    }
}
</script>

<template>
  <main class="enroll-device">
    <h1 class="enroll-device__title">
        Registrera enhet
    </h1>

    <form v-if="!enrollmentKey">
        <input type="text" placeholder="Enhetens namn" v-model="deviceName" />
        <span class="form-row">
            <label class="enroll-device__auto-answer" for="autoanswer">
                <input id="autoanswer" type="checkbox" v-model="autoAnswer" />
                Svara automatiskt
            </label>
            <span class="enroll-device__auto-answer-delay">
                <span>efter</span>
                <input type="number" placeholder="Auto Answer Delay" v-model="autoAnswerDelay" />
                <span>sek</span>
            </span>
        </span>

        <button class="btn" @click="enrollDevice">Registrera enhet</button>
    </form>

    <div class="enroll-device__qrcode-container" v-else>
        <p class="enroll-device__desc">
            Skanna QR-koden med enheten.
        </p>
        <qrcode-vue
            :value="enrollmentKey"
            :size="300"
            level="H"
        />

        <router-link class="btn" to="/" v-if="showContinueBtn">
            Jag har skannat QR-koden!
        </router-link>
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

    &__title {
        margin-bottom: 1.5rem;
    }

    &__desc {
        margin-bottom: 2rem;
        text-align: center;
    }

    form {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: 1rem;

        input {
            padding: 1rem;
            border-radius: 5px;
            border: 1px solid #ccc;
            width: 100%;
            font-size: 1rem;
        }
    }

    .form-row {
        display: flex;
        align-items: center;
        justify-content: space-between;
        width: 100%;
    }

    &__auto-answer {
        display: flex;
        align-items: center;
        gap: 1rem;
        width: 50%;

        #autoanswer {
            height: 1.25rem;
            width: 1.25rem;
        }
    }

    &__auto-answer-delay {
        display: flex;
        align-items: center;
        justify-content: flex-end;
        gap: .5rem;
        width: 50%;

        > input {
            width: 40% !important;
        }
    }

    &__qrcode-container {
        display: flex;
        flex-direction: column;
        align-items: center;
    }
}

.btn {
    background-color: rgb(67, 107, 177);
    color: rgb(255, 255, 255);
    padding: 1rem 2rem;
    margin-top: 2rem;
    text-align: center;
    border-radius: 30px;
    text-decoration: none;
    transition: all 0.3s;
    font-size: 1rem;
    border: none;

    &:hover {
      background-color: rgb(67, 107, 177, 0.9);
      color: white;
    }
}
</style>
