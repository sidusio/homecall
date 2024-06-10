<script setup lang="ts">
import { officeClient } from '@/clients';
import QrcodeVue from 'qrcode.vue'
import { useAuth0 } from '@auth0/auth0-vue';
import { computed, onMounted, onUnmounted } from 'vue';

const { getAccessTokenSilently } = useAuth0();

interface Enrollment {
  enrollmentKey: string;
  deviceId: string;
}

const props = defineProps<{
    enrollment: Enrollment;
}>()

const qrcode = computed(() => {
    const currentUrlOrigin = window.location.origin

    return 'homecall://' + JSON.stringify({
        deviceId: props.enrollment.deviceId,
        enrollmentKey: props.enrollment.enrollmentKey,
        instanceUrl: currentUrlOrigin + '/api',
        audience: 'homecall'
    })
})

// Define emit enrolled
const emit = defineEmits({
    enrolled(deviceId: string) {
        return deviceId
    },
    close() {
        return true
    }
})

const abort = new AbortController();

const waitForEnrollment = async (): Promise<string> => {
    const token = await getAccessTokenSilently();

    const devices = officeClient.waitForEnrollment({
        deviceId: props.enrollment.deviceId,
    },
    {
        signal: abort.signal,
        headers: {
            Authorization: 'Bearer ' + token
        }
    })

    for await (const res of devices) {
        abort.abort()
        return res.device?.id || ''
    }

    return waitForEnrollment()
}

onMounted(async () => {
    const deviceId = await waitForEnrollment()
    emit('enrolled', deviceId)
})

onUnmounted(() => {
    abort.abort()
})
</script>

<template>
    <div class="qrcode-container">
        <p class="qrcode-container__desc">
            Skanna QR-koden med enheten.
        </p>

        <qrcode-vue
            :value="qrcode"
            :size="300"
            level="H"
        />

        <button class="btn" @click="emit('close')">
            Avbryt
        </button>
    </div>
</template>

<style lang="scss" scoped>
.qrcode-container {
    display: flex;
    flex-direction: column;
    align-items: center;

    &__desc {
        margin-bottom: 2rem;
        text-align: center;
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
        cursor: pointer;
    }
}
</style>
