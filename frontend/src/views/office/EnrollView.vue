<script setup lang="ts">
import { ref } from 'vue'
import QrcodeVue from 'qrcode.vue'
import { officeClient } from '@/clients'
import { useRoute, useRouter } from 'vue-router';

const router = useRouter()

const deviceName = ref('')
const usersName = ref('')
const autoAnswer = ref(false)
const autoAnswerDelay = ref(0)

const enrollmentKey = ref('')
const showContinueBtn = ref(false)

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
        <input type="text" placeholder="Device Name" v-model="deviceName" />
        <input type="text" placeholder="User's Name" v-model="usersName" />
        <span class="form-row">
            <label for="autoanswer">
                <input id="autoanswer" type="checkbox" v-model="autoAnswer" />
                Svara automatiskt
            </label>
            <input type="number" placeholder="Auto Answer Delay" v-model="autoAnswerDelay" />
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

        input {
            padding: 1rem;
            border-radius: 5px;
            border: 1px solid #ccc;
            width: 100%;
            font-size: 1rem;

            &[type="number"] {
                width: 20%;
            }
        }
    }

    .form-row {
        display: flex;
        align-items: center;
        width: 100%;
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
