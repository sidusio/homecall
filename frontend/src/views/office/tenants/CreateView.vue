<script setup lang="ts">
import { ref } from 'vue';
import { tenantClient } from '@/clients';
import { useRouter } from 'vue-router'
import { useAuth0 } from '@auth0/auth0-vue';

const { getAccessTokenSilently } = useAuth0();
const router = useRouter()
const tenantName = ref<string>('')

/**
 * Create a new tenant.
 */
const createTenant = async (e: Event) => {
    e.preventDefault()

    const token = await getAccessTokenSilently();
    const auth = {
        headers: {
            Authorization: 'Bearer ' + token
        }
    }

    const { tenant } = await tenantClient.createTenant({
        name: tenantName.value
    }, auth)

    if(!tenant) {
        return;
    }

    localStorage.setItem('tenantId', tenant.id)

    router.push({ path: '/dashboard' })
}
</script>

<template>
    <main class="create fill-height">
        <h1>Skapa ny organisation</h1>

        <form class="create__form">
            <input type="text" v-model="tenantName" placeholder="Skriv in namn..." />
            <button type="submit" @click="createTenant"> + </button>
        </form>
    </main>
</template>

<style lang="scss" scoped>
@import '@/assets/styles/variables.scss';

.create {
    height: $viewport-height;
    width: 100vw;
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;

    &__form {
        display: flex;
        width: 50%;
        gap: 1rem;
        margin-top: 2rem;

        button {
            height: 50px;
            width: 50px;
            font-size: 2.5rem;
            border-radius: 100%;
            border: none;
            background-color: #002594;
            color: #fff;
            cursor: pointer;
            transition: all 0.3s ease-in-out;

            &:hover {
                background-color: #001f4d;
            }
        }
    }
}
</style>
