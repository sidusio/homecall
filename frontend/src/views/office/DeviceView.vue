<script setup lang="ts">
import { ref, onMounted } from 'vue';
import Office from '@/templates/Office.vue';
import { useRoute, useRouter } from 'vue-router';
import { useAuth0 } from '@auth0/auth0-vue';
import { officeClient } from '@/clients';
import { useTenantIdStore } from '@/stores/tenantId';
import Loading from '@/components/Loading.vue';

interface Device {
    id: string;
    name: string;
    enrollmentKey: string;
    online: boolean;
    tenantId: string;
}

const route = useRoute();
const router = useRouter();
const { getAccessTokenSilently } = useAuth0();
const deviceId = ref(route.params.deviceId);
const tenantIdStore = useTenantIdStore()
const device = ref<Device | null>(null)
const removeDeviceModal = ref(false)
const loading = ref(false)

const auth = async () => {
    const token = await getAccessTokenSilently();
    return {
        headers: {
            Authorization: 'Bearer ' + token
        }
    }
}

const removeDevice = async () => {
    await officeClient.removeDevice({
        deviceId: device.value?.id
    }, await auth()).then(() => {
        router.push('/dashboard')
    }).catch((e) => {
        console.error(e)
    })
}

const toggleRemoveDevice = () => {
    removeDeviceModal.value = !removeDeviceModal.value
}

onMounted(async() => {
    loading.value = true

    device.value = await officeClient.listDevices({
        tenantId: tenantIdStore.tenantId
    }, await auth()).then((res) => {
        return res.devices.find((device) => device.id === deviceId.value)
    }).catch((e) => {
        console.error(e)
    }) || null

    loading.value = false
})
</script>

<template>
    <Office>
        <div class="loading" v-if="loading">
            <Loading/>
        </div>

        <div v-else>
            <div class="device" v-if="device">
                <header class="device__header">
                    <div class="device__header--left">
                        <p class="device__text--small">
                            {{ device.id }}
                        </p>
                        <h1>{{ device?.name }}</h1>
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

                    <div class="device__call">
                        <button
                            class="device__call-btn"
                            :class="device.online ? 'device__call-btn--success' : 'device__call-btn--danger'"
                            :disabled="!device.online"
                        >
                            <font-awesome-icon icon="phone" />
                            Ring
                        </button>

                        <p class="device__call-btn__note" v-if="!device.online">
                            Enheter som är offline kan inte ringas till.
                        </p>
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

            <div v-else>
                <h1>Enhet hittades inte</h1>
            </div>
        </div>
    </Office>
</template>

<style lang="scss" scoped>
@import "@/assets/styles/variables.scss";

.loading {
    height: $viewport-height;
    display: flex;
    justify-content: center;
    align-items: center;

    p {
        font-size: 1.5rem;
    }
}

.device {
    padding: 2rem;
    height: $viewport-height;


    &__header {
        display: flex;
        justify-content: space-between;
        align-items: center;

        &--left {
            display: flex;
            flex-direction: column;
            gap: .8rem;
        }
    }

    &__divider {
        margin: 4rem 0;
        border: none;
        border-top: 1px solid #e5e5e5;
    }

    &__text--small {
        font-size: .8rem;
        color: gray;
        margin: 0;
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

    &__call {
        position: relative;
    }

    &__call-btn {
        display: flex;
        align-items: center;
        gap: .5rem;
        padding: 1rem 1.5rem;
        border-radius: 30px;
        color: white;
        font-size: 1.2rem;
        border: none;
        transition: all .3s;
        cursor: pointer;

        &--success {
            background-color: $color-success;
        }

        &--danger {
            background-color: #a5a5a5;
            cursor: not-allowed;

            &:hover {
                ~ .device__call-btn {
                    &__note {
                        opacity: 1;
                        transition: all .3s;
                    }
                }
            }
        }

        svg {
            font-size: 1.1rem;
        }

        &:hover {
            transform: scale(1.05);
        }

        &__note {
            position: absolute;
            right: 0;
            top: calc(100% + 10px);
            opacity: 0;
            font-size: .8rem;
            color: gray;
            text-align: right;
            width: max-content;
            transition: all .3s;
        }
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
