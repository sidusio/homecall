<script setup lang="ts">
import { useTenantIdStore } from '@/stores/tenantId';
import { ref, onMounted } from 'vue';
import { tenantClient } from '@/clients';
import { useAuth0 } from '@auth0/auth0-vue';
import EditMember from '@/components/tenants/members/EditMember.vue';
import InviteMember from '@/components/tenants/members/InviteMember.vue';
import RemoveMember from '@/components/tenants/members/RemoveMember.vue';
import RemoveInvite from '@/components/tenants/members/RemoveInvite.vue';
import { Role } from "./../../../gen/connect/homecall/v1alpha/tenant_service_pb";

const { getAccessTokenSilently, user } = useAuth0();
const tenantIdStore = useTenantIdStore();
const allMembers = ref<any[]>([]);
const allInvites = ref<any[]>([]);

/**
 * Subscribe to tenantId changes.
 */
useTenantIdStore().$subscribe(() => {
    getAllMembers()
})

const auth = async () => {
    const token = await getAccessTokenSilently();
    const auth = {
        method: 'GET',
        redirect: 'follow',
        headers: {
            Authorization: 'Bearer ' + token
        }
    }

    return auth;
}

const getAllInvites = async () => {
    allInvites.value = []

    const { tenantInvites } = await tenantClient.listTenantInvites({
        tenantId: tenantIdStore.tenantId
    }, await auth())

    allInvites.value.push(...tenantInvites)
}

/**
 * Get all members of the tenant.
 */
const getAllMembers = async () => {
    allMembers.value = []

    const { tenantMembers } = await tenantClient.listTenantMembers({
        tenantId: tenantIdStore.tenantId
    }, await auth())

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

onMounted(async () => {
    getAllMembers()
    getAllInvites()
})
</script>

<template>
  <div class="add-members">
    <div class="add-members__header">
        <h2>Medlemmar</h2>

        <InviteMember @invite="getAllInvites()" />
    </div>

    <div class="add-members__members">
        <article v-for="member in allMembers" class="add-members__member">
            <p class="add-members__member__info">
                <span>{{ member.displayName }} ({{ member.verifiedEmail }})</span>
                <span class="add-members__member__role">-</span>
                <span class="add-members__member__role">{{ setRole(member.role) }}</span>
            </p>

            <div
                v-if="user && user.email !== member.email"
                class="add-members__member__btns"
            >
                <EditMember :id="member.id" :role="member.role" @edit="getAllMembers" />

                <RemoveMember :id="member.id" @remove="getAllMembers" />
            </div>
        </article>

        <h3>
            Inbjudningar
        </h3>

        <p class="add-members__no-invites" v-if="allInvites.length === 0">
            <font-awesome-icon class="add-members__no-invites__icon" icon="fa-solid fa-envelope" />
            Inga inbjudningar.
        </p>

        <article v-for="invite in allInvites" class="add-members__member" v-else>
            <p class="add-members__member__info">
                <span>{{ invite.email }}</span>
                <span class="add-members__member__role">-</span>
                <span class="add-members__member__role">{{ setRole(invite.role) }}</span>
            </p>

            <div class="add-members__member__btns">
                <RemoveInvite :id="invite.id" @remove="getAllInvites" />
            </div>
        </article>
    </div>
  </div>
</template>

<style lang="scss" scoped>
@import "@/assets/styles/variables.scss";

.add-members {
    &__header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 1rem;
    }

    &__members {
        h3 {
            margin: 3rem 0 1rem 0;
            font-weight: 500;
        }
    }

    &__no-invites {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: .5rem;
        text-align: center;
        font-size: 1.2rem;
        color: $color-primary;

        &__icon {
            font-size: 2rem;
        }
    }

    &__member {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 0 0 1rem 0;

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
