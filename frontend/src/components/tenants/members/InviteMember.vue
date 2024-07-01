<script setup lang="ts">
import { ref, defineEmits } from 'vue';
import { Role } from "./../../../../gen/connect/homecall/v1alpha/tenant_service_pb";
import { useAuth0 } from '@auth0/auth0-vue';
import { tenantClient } from '@/clients';
import Select from '@/templates/Select.vue';

const { getAccessTokenSilently, user } = useAuth0();
const open = ref(false);
const invitedEmail = ref<string>('');
const invitedRole = ref<Role>(Role.UNSPECIFIED);

const emit = defineEmits(['invite']);

/**
 * Add a member to the tenant.
 */
const addMember = async () => {
    const tenantId = localStorage.getItem('tenantId')

    if(!tenantId) {
        return;
    }

    const token = await getAccessTokenSilently();
    const auth = {
        method: 'GET',
        redirect: 'follow',
        headers: {
            Authorization: 'Bearer ' + token
        }
    }

    await tenantClient.createTenantMember({
        tenantId: tenantId,
        email: invitedEmail.value,
        role: invitedRole.value
    }, auth)

    toggle()
    emit('invite') // To update the list of members.
}

/**
 * Toggle the modal.
 */
const toggle = () => {
    open.value = !open.value;
}
</script>

<template>
    <button
        @click="toggle"
        class="btn btn--outlined"
    >
        Bjud in medlem
    </button>

    <div class="modal" v-if="open">
        <h2>Bjud in medlem</h2>

        <input
            v-model="invitedEmail"
            type="email"
            placeholder="E-post"
        />

        <Select>
            <select v-model="invitedRole">
                <option :value="Role.UNSPECIFIED">Okänd</option>
                <option :value="Role.ADMIN">Admin</option>
                <option :value="Role.MEMBER">Medlem</option>
            </select>
        </Select>

        <div class="modal__btns">
            <button
                @click="toggle"
                class="btn btn--outlined"
            >
                Avbryt
            </button>

            <button
                @click="addMember"
                class="btn btn--filled"
            >
                Bjud in
            </button>
        </div>
    </div>
</template>

<style lang="scss" scoped>
</style>
