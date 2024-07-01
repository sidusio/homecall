<script setup lang="ts">
import { tenantClient } from '@/clients';
import { useAuth0 } from '@auth0/auth0-vue';

const { getAccessTokenSilently } = useAuth0();

defineProps(['email']);

const emit = defineEmits(['remove']);

/**
 * Remove a member from the tenant.
 */
const removeMember = async (email: string) => {
    const tenantId = localStorage.getItem('tenantId')

    if(!tenantId) {
        return;
    }

    const token = await getAccessTokenSilently();
    const auth = {
        headers: {
            Authorization: 'Bearer ' + token
        }
    }

    await tenantClient.removeTenantMember({
        tenantId: tenantId,
        email: email
    }, auth)

    emit('remove') // To update the list of members.
}
</script>

<template>
    <button
        class="link-btn link-btn--round link-btn--danger"
        @click="removeMember(email)"
    >
        <font-awesome-icon icon="fa-solid fa-trash" />
    </button>
</template>
