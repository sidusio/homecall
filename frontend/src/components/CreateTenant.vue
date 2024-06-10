<script lang="ts" setup>
import { ref } from 'vue';
import { tenantClient } from '@/clients';
import { useAuth0 } from '@auth0/auth0-vue';
import { useRouter } from 'vue-router'

const { getAccessTokenSilently } = useAuth0();
const router = useRouter()
const tenantName = ref<string>('')

/**
 * Create a new tenant.
 */
const createTenant = async () => {
    const token = await getAccessTokenSilently();
    const auth = {
        headers: {
            Authorization: 'Bearer ' + token
        }
    }

    await tenantClient.createTenant({
        name: tenantName.value
    }, auth)

    router.push({ path: '/tenants' })
}
</script>

<template>
    <div>
        <h1>Skapa ny grupp</h1>

        <form class="create-tenant__form">
            <input type="text" v-model="tenantName" placeholder="Skriv in namn..." />
            <input type="submit" value="Skapa" @click="createTenant" />
        </form>
    </div>
</template>

<style lang="scss">
.create-tenant {
    &__form {
        display: flex;
        flex-direction: column;
    }
}
</style>
