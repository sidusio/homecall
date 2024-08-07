<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { officeClient } from '@/clients'
import { useAuth0 } from '@auth0/auth0-vue';

const { getAccessTokenSilently } = useAuth0();

interface Enrollment {
  enrollmentKey: string;
  deviceId: string;
}

const deviceName = ref('')
const autoAnswer = ref(false)
const autoAnswerDelay = ref(0)
const error = ref(false)

const emit = defineEmits({
    registered(enrollment: Enrollment) {
        return enrollment
    }
})

const enrollDevice = async (e: Event) => {
    e.preventDefault()

    const tenantId = localStorage.getItem('tenantId')

    if(!tenantId) {
        return;
    }

    if(!deviceName.value) {
        error.value = true;
        return;
    }

    const token = await getAccessTokenSilently();
    const auth = {
        headers: {
            Authorization: 'Bearer ' + token
        }
    }

    const res = await officeClient.createDevice({
        name: deviceName.value,
        defaultSettings: {
            autoAnswer: autoAnswer.value,
            autoAnswerDelaySeconds: BigInt(autoAnswerDelay.value)
        },
        tenantId: tenantId
    }, auth)

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
            <div class="register-device__input-container">
                <label for="name">
                    Enhetens namn <span class="mandatory">*</span>
                </label>
                <input type="text" placeholder="Fyll i enhetens namn..." v-model="deviceName" />
                <p v-if="error" class="mandatory">Enheten måste ha ett namn.</p>
            </div>

            <span class="form-row">
                <label class="register-device__auto-answer" for="autoanswer">
                    <input id="autoanswer" type="checkbox" v-model="autoAnswer" />
                    Svara automatiskt
                </label>
                <span class="register-device__auto-answer-delay">
                    <span>efter</span>
                    <input type="number" placeholder="Auto Answer Delay" v-model="autoAnswerDelay" min="0"/>
                    <span>sek</span>
                </span>
            </span>

            <button class="btn btn--filled" @click="enrollDevice">Registrera enhet</button>
        </form>
    </div>
</template>

<style lang="scss" scoped>
@import '@/assets/styles/variables.scss';

.register-device {
    height: $viewport-height;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;

    &__title {
        margin-bottom: 1.5rem;
    }

    &__input-container {
        width: 100%;

        label {
            text-align: left;
        }
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
    padding: 1rem 2rem;
    margin-top: 2rem;
}
</style>
