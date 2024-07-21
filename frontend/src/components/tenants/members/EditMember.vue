<script setup lang="ts">
import { ref } from 'vue';
import { Role } from "./../../../../gen/connect/homecall/v1alpha/tenant_service_pb";
import { tenantClient } from '@/clients';
import { useAuth0 } from '@auth0/auth0-vue';
import Select from '@/templates/Select.vue';

const emit = defineEmits(['edit']);
const props = defineProps(['id', 'role']);

const { getAccessTokenSilently } = useAuth0();
const open = ref(false);
const role = ref<Role>(props.role);

/**
 * Edit a member.
 */
const editMember = async () => {
    const token = await getAccessTokenSilently();
    const auth = {
        headers: {
            Authorization: 'Bearer ' + token
        }
    }

    await tenantClient.updateTenantMember({
        id: props.id,
        role: role.value
    }, auth)

    toggle();
    emit('edit');
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
        class="link-btn link-btn--round"
    >
        <font-awesome-icon icon="fa-solid fa-pen" />
    </button>

    <div class="overlay" v-if="open"></div>

    <div class="modal" v-if="open">
        <h2 class="edit-member__title">
            Redigera medlem
        </h2>

        <div class="edit-member__column">
            <div class="input-container">
                <label for="role">
                    Roll
                </label>

                <Select>
                    <select v-model="role" id="role">
                        <option :value="Role.UNSPECIFIED">Okänd</option>
                        <option :value="Role.ADMIN">Admin</option>
                        <option :value="Role.MEMBER">Medlem</option>
                    </select>
                </Select>
            </div>
        </div>

        <div class="edit-member__btns">
            <button
                class="btn btn--outlined"
                @click="toggle"
            >
                Avbryt
            </button>

            <button
                class="btn btn--filled"
                @click="editMember"
            >
                Spara
            </button>
        </div>
    </div>
</template>

<style lang="scss">
@import "@/assets/styles/mixins.scss";

.edit-member {
    &__title {
        margin-bottom: 1rem;
    }

    &__column {
        display: flex;
        flex-direction: column;
    }

    &__btns {
        display: flex;
        justify-content: flex-end;
        align-items: center;
        gap: 1rem;
    }
}
</style>
