<script setup lang="ts">
import { tenantClient } from '@/clients';
import { useAuth0 } from '@auth0/auth0-vue';
import { ref } from 'vue';
import { useRouter } from 'vue-router'
import { useTenantIdStore } from '@/stores/tenantId';

const { getAccessTokenSilently } = useAuth0();
const { tenantId, removeTenantId } = useTenantIdStore();
const open = ref(false);
const router = useRouter();

/**
 * Toggle the modal.
 */
const toggle = () => {
    open.value = !open.value;
};

/**
 * Remove the tenant.
 */
const removeTenant = async () => {
    const token = await getAccessTokenSilently();
    const auth = {
        headers: {
            Authorization: 'Bearer ' + token
        }
    }

    await tenantClient.removeTenant({
        id: tenantId
    }, auth);
    open.value = false;

    removeTenantId();
    router.push('/tenants');
};
</script>

<template>
    <main class="remove">
        <h2 class="remove__title">Ta bort organisation</h2>

        <p class="remove__text">
            Om du tar bort organisationen kommer all data att raderas och det går inte att ångra.
        </p>

        <button
            @click="toggle"
            class="btn btn--danger"
        >
            <font-awesome-icon icon="trash" />
            Ta bort organisation
        </button>
    </main>

    <div class="overlay" v-if="open"></div>

    <div class="modal" v-if="open">
        <h2>Är du helt säker?</h2>
        <p class="remove__modal__text">
            Om du tar bort organisationen kommer all data att raderas och det går inte att ångra.
        </p>

        <div class="modal__btns">
            <button
                @click="removeTenant"
                class="btn btn--danger"
            >
                <font-awesome-icon icon="trash" />
                Ja, jag är säker
            </button>

            <button
                @click="toggle"
                class="btn btn--outlined"
            >
                Avbryt
            </button>
        </div>
    </div>
</template>

<style lang="scss" scoped>
.remove {
    &__title {
        color: #ff3336;
        font-weight: 500;
    }

    &__text {
        margin-bottom: 2rem;
    }

    &__modal {
        position: fixed;
        top: 50%;
        left: 50%;
        transform: translate(-50%, -50%);
        background-color: white;
        padding: 3rem;
        border-radius: 5px;
        box-shadow: 0 1px 4px rgba(0, 0, 0, 0.1);
        z-index: 101;

        &__text {
            margin-top: .5rem;
            margin-bottom: 2rem;
        }

        &__btns {
            display: flex;
            gap: 1rem;
            justify-content: center;
        }
    }
}
</style>
