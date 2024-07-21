<script setup lang="ts">
import { ref } from 'vue';
import { tenantClient } from '@/clients';
import { useAuth0 } from '@auth0/auth0-vue';
import { useTenantIdStore } from '@/stores/tenantId';

const { getAccessTokenSilently } = useAuth0();
const tenantIdStore = useTenantIdStore();

defineProps(['id']);

const emit = defineEmits(['remove']);
const open = ref(false);

/**
 * Toggle the modal.
 */
 const toggle = () => {
    open.value = !open.value;
};

/**
 * Remove a member from the tenant.
 */
const removeMember = async (id: string) => {
    const token = await getAccessTokenSilently();
    const auth = {
        headers: {
            Authorization: 'Bearer ' + token
        }
    }

    await tenantClient.removeTenantInvite({
        id: id
    }, auth)

    emit('remove') // To update the list of members.
}
</script>

<template>
    <button
        class="link-btn link-btn--round link-btn--danger"
        @click="toggle"
    >
        <font-awesome-icon icon="fa-solid fa-trash" />
    </button>

    <div class="overlay" v-if="open"></div>

    <div class="modal" v-if="open">
        <h2>Är du helt säker?</h2>

        <p class="modal__text">
            Om du tar bort inbjudan till denna medlem kommer all associerad data att raderas och det går inte att ångra.
        </p>

        <div class="modal__btns">
            <button
                @click="removeMember(id)"
                class="btn btn--danger"
            >
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
