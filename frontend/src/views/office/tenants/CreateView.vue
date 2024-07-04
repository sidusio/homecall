<script setup lang="ts">
import { ref } from 'vue';
import { tenantClient } from '@/clients';
import { useRouter } from 'vue-router'
import { useAuth0 } from '@auth0/auth0-vue';

const { getAccessTokenSilently } = useAuth0();
const router = useRouter()
const tenantName = ref<string>('')
const error = ref(false)

/**
 * Create a new tenant.
 */
const createTenant = async (e: Event) => {
    e.preventDefault()

    if(!tenantName.value) {
        error.value = true;
        return;
    }

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
            <div class="">
                <label for="tenantName">
                    Organisationsnamn <span class="mandatory">*</span>
                </label>
                <input type="text" v-model="tenantName" placeholder="Skriv in namn..." />
                <p class="mandatory" v-if="error">
                    Du måste fylla i ett namn.
                </p>
            </div>

            <button class="btn btn--filled" type="submit" @click="createTenant"> Skapa </button>
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
        flex-direction: column;
        width: 50%;
        gap: 1rem;
        margin-top: 2rem;
    }

    .btn {
        align-self: flex-end;
        width: fit-content;
        padding-left: 3rem;
        padding-right: 3rem;
    }
}
</style>
