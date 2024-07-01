<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { tenantClient } from '@/clients';
import { useAuth0 } from '@auth0/auth0-vue';
import EditMember from '@/components/tenants/members/EditMember.vue';
import InviteMember from '@/components/tenants/members/InviteMember.vue';
import RemoveMember from '@/components/tenants/members/RemoveMember.vue';
import { Role } from "./../../../gen/connect/homecall/v1alpha/tenant_service_pb";

const { getAccessTokenSilently, user } = useAuth0();
const allMembers = ref<any[]>([]);

/**
 * Get all members of the tenant.
 */
const getAllMembers = async () => {
    allMembers.value = []
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

    const { tenantMembers } = await tenantClient.listTenantMembers({
        tenantId: tenantId
    }, auth)

    allMembers.value.push(...tenantMembers)
}

/**
 * Set the role of a member.
 *
 * @param role - The role of the member.
 */
const setRole = (role: Role) => {
    switch(role) {
        case Role.ADMIN:
            return 'Admin'
        case Role.MEMBER:
            return 'Medlem'
        case Role.UNSPECIFIED:
            return 'Okänd'
        default:
            return 'Okänd'
    }
}

onMounted(() => {
    getAllMembers()
})
</script>

<template>
  <div class="add-members">
    <div class="add-members__header">
        <h2>Medlemmar</h2>

        <InviteMember @invite="getAllMembers" />
    </div>

    <div>
        <article v-for="member in allMembers" class="add-members__member">
            <p class="add-members__member__info">
                <span>{{ member.email }}</span>
                <span class="add-members__member__role">-</span>
                <span class="add-members__member__role">{{ setRole(member.role) }}</span>
            </p>

            <div
                v-if="user && user.email !== member.email"
                class="add-members__member__btns"
            >
                <EditMember :email="member.email" :role="member.role" @edit="getAllMembers" />

                <RemoveMember :email="member.email" @remove="getAllMembers" />
            </div>
        </article>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.add-members {
    &__header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 1rem;
    }

    &__member {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 1rem 0;

        h3 {
            margin: 0;
        }

        &__role {
            color: #7e7e7e;
        }

        &__info {
            display: flex;
            align-items: center;
            gap: 1rem;
        }

        &__btns {
            display: flex;
            gap: .5rem;
        }
    }
}
</style>
