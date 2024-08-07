<script setup lang="ts">
import { ref } from 'vue';
import { useAuth0 } from '@auth0/auth0-vue';
import { officeClient } from '@/clients';

interface Device {
    id: string;
    name: string;
    enrollmentKey: string;
    online: boolean;
}

const props = defineProps<{
    device: Device;
}>();

const emit = defineEmits(['remove']);

const { getAccessTokenSilently } = useAuth0();
const removeDeviceModal = ref(false);

/**
 * Authenticate the user.
 */
const auth = async () => {
    const token = await getAccessTokenSilently();
    return {
        headers: {
            Authorization: 'Bearer ' + token
        }
    }
}

/**
 * Remove a device.
 */
const removeDevice = async () => {
    await officeClient.removeDevice({
        deviceId: props.device.id
    }, await auth()).then(() => {
        emit('remove')
        toggleRemoveDevice()
    }).catch((e) => {
        console.error(e)
    })
}

/**
 * Toggle the remove device modal.
 */
const toggleRemoveDevice = () => {
    removeDeviceModal.value = !removeDeviceModal.value
}
</script>

<template>
    <div class="device">
        <div>
            <header class="device__header">
                <h1>
                    <font-awesome-icon icon="tablet-screen-button" />
                    {{ device.name }}
                </h1>

                <div>
                    <div class="device__tags">
                        <p class="device__tag device__tag--success" v-if="!device.enrollmentKey">
                            Registrerad
                        </p>
                        <p class="device__tag device__tag--danger" v-else>
                            Inte registrerad
                        </p>

                        <p class="device__tag device__tag--success" v-if="device.online">
                            Online
                        </p>
                        <p class="device__tag device__tag--danger" v-else>
                            Offline
                        </p>
                    </div>
                </div>
            </header>

            <hr class="device__divider"/>

            <section class="device__dangerzone">
                <h2>Ta bort enhet</h2>
                <p>Om du tar bort enheten kommer du inte kunna ringa den längre.</p>

                <button class="btn btn--danger" @click="toggleRemoveDevice">
                    <font-awesome-icon icon="trash" />
                    Ta bort enhet
                </button>
            </section>

            <div class="overlay" v-if="removeDeviceModal"></div>

            <div class="modal" v-if="removeDeviceModal">
                <h1>Är du säker?</h1>
                <p class="modal__text">
                    Om du tar bort enheten kommer du inte kunna ringa den längre.
                </p>

                <div class="modal__btns">
                    <button class="btn btn--danger" @click="removeDevice">
                        <font-awesome-icon icon="trash" />
                        Ja, jag är säker
                    </button>
                    <button class="btn" @click="toggleRemoveDevice">
                        Avbryt
                    </button>
                </div>
            </div>
        </div>

        <footer>
            <p class="device__text--smaller">
                Enhetens id: {{ device.id }}
            </p>
        </footer>
    </div>
</template>

<style lang="scss" scoped>
@import "@/assets/styles/variables.scss";

.device {
    padding: 2rem 1rem 1.5rem 1rem;
    width: 100%;
    height: 100%;
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    gap: 1rem;

    &__header {
        display: flex;
        flex-direction: column;
        gap: 1rem;
    }

    h1 {
        display: flex;
        align-items: center;
        gap: 1rem;
        margin: 0;
        line-height: normal;
    }

    &__text {
        &--small {
            font-size: .8rem;
            margin: 0;
        }

        &--smaller {
            font-size: .7rem;
            opacity: 50%;
        }
    }

    &__tags {
        display: flex;
        gap: 1rem;
    }

    &__tag {
        padding: .3rem 1rem;
        border-radius: 30px;
        color: white;
        font-size: .8rem;

        &--success {
            background-color: $color-success;
        }

        &--danger {
            background-color: #a5a5a5;
        }
    }

    &__divider {
        margin: 2rem 0;
        border: none;
        border-top: 1px solid #e5e5e5;
    }

    &__dangerzone {
        h2 {
            font-size: 1.5rem;
            color: $color-danger;
            font-weight: 500;
        }

        .btn {
            margin-top: 2rem;
        }
    }
}
</style>
