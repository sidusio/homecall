<script setup lang="ts">
import { ref } from 'vue'
import { officeClient } from '@/clients'

interface Enrollment {
  enrollmentKey: string;
  deviceId: string;
}

const deviceName = ref('')
const autoAnswer = ref(false)
const autoAnswerDelay = ref(0)

const emit = defineEmits({
    registered(enrollment: Enrollment) {
        return enrollment
    }
})

const enrollDevice = async (e: Event) => {
    e.preventDefault()

    const res = await officeClient.createDevice({
        name: deviceName.value,
        defaultSettings: {
            autoAnswer: autoAnswer.value,
            autoAnswerDelaySeconds: BigInt(autoAnswerDelay.value)
        }
    })

    if(res.device?.enrollmentKey) {
        emit('registered', {
            enrollmentKey: res.device.enrollmentKey,
            deviceId: res.device.id
        })
    }
}
</script>

<template>
    <div class="register-device">
        <h1 class="register-device__title">
            Registrera enhet
        </h1>

        <form>
            <input type="text" placeholder="Enhetens namn" v-model="deviceName" />
            <span class="form-row">
                <label class="register-device__auto-answer" for="autoanswer">
                    <input id="autoanswer" type="checkbox" v-model="autoAnswer" />
                    Svara automatiskt
                </label>
                <span class="register-device__auto-answer-delay">
                    <span>efter</span>
                    <input type="number" placeholder="Auto Answer Delay" v-model="autoAnswerDelay" />
                    <span>sek</span>
                </span>
            </span>

            <button class="btn" @click="enrollDevice">Registrera enhet</button>
        </form>
    </div>
</template>

<style lang="scss" scoped>
.register-device {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100vh;

    &__title {
        margin-bottom: 1.5rem;
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
