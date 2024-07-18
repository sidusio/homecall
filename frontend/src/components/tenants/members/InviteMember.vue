<script setup lang="ts">
import { ref, defineEmits } from 'vue';
import { Role } from "./../../../../gen/connect/homecall/v1alpha/tenant_service_pb";
import { useAuth0 } from '@auth0/auth0-vue';
import { tenantClient } from '@/clients';
import { useTenantIdStore } from '@/stores/tenantId';
import Select from '@/templates/Select.vue';

const { getAccessTokenSilently, user } = useAuth0();
const tenantIdStore = useTenantIdStore();
const open = ref(false);
const invitedEmail = ref<string>('');
const invitedRole = ref<Role>(Role.UNSPECIFIED);
const error = ref<boolean>(false);

const emit = defineEmits(['invite']);

/**
 * Add a member to the tenant.
 */
const addMember = async () => {
    if(!invitedEmail.value && !invitedRole.value) {
        error.value = true;

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
        tenantId: tenantIdStore.tenantId,
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
    error.value = false;
}
</script>

<template>
    <button
        @click="toggle"
        class="btn btn--outlined"
    >
        Bjud in medlem
    </button>

    <div class="overlay" v-if="open"></div>

    <div class="modal" v-if="open">
        <h2>Bjud in medlem</h2>

        <div class="input-container">
            <label for="email">
                E-post <span class="mandatory">*</span>
            </label>

            <input
                v-model="invitedEmail"
                type="email"
                id="email"
                placeholder="E-post"
            />

            <p class="mandatory" v-if="error && !invitedEmail">
                Du måste ange en e-postadress.
            </p>
        </div>

        <div class="input-container">
            <label for="role">
                Roll <span class="mandatory">*</span>
            </label>

            <Select>
                <select v-model="invitedRole" id="role">
                    <option :value="Role.UNSPECIFIED" disabled>Välj roll</option>
                    <option :value="Role.ADMIN">Admin</option>
                    <option :value="Role.MEMBER">Medlem</option>
                </select>
            </Select>

            <p class="mandatory" v-if="error && !invitedEmail">
                Du måste välja en roll.
            </p>
        </div>

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
