<script lang="ts" setup>
import { onMounted, ref } from 'vue';
import { useAuth0 } from '@auth0/auth0-vue';
import { useTenantIdStore } from '@/stores/tenantId';
import SelectTenant from '@/components/tenants/SelectTenant.vue';
import UserMenu from '@/components/UserMenu.vue';
import TenantSettings from '@/components/tenants/TenantSettings.vue';
import { tenantClient } from '@/clients';

const { isAuthenticated, getAccessTokenSilently, user } = useAuth0(); // Check if logged in.
const tenantIdStore = useTenantIdStore(); // Get the tenantId.
const hasInvites = ref(false);
const role = ref<string>('member')

const auth = async () => {
    const token = await getAccessTokenSilently();
    const auth = {
        headers: {
            Authorization: 'Bearer ' + token
        }
    }

    return auth;
}

/**
 * Subscribe to tenantId changes.
 */
 useTenantIdStore().$subscribe(() => {
    isAdmin()
})

/**
 * Check if the user is an admin.
 */
const isAdmin = async () => {
    const res = await tenantClient.listTenantMembers({
        tenantId: tenantIdStore.tenantId
    }, await auth()).then(res => true).catch(err => false)

    if(res) {
        role.value = 'admin'
    } else {
        role.value = 'member'
    }
}

/**
 * Check if the user has invites.
 */
const checkIfHasInvites = async () => {
    const { tenantInvites } = await tenantClient.listTenantInvites({}, await auth());

    return tenantInvites.length > 0;
}

onMounted(async () => {
    if(!user.value || !user.value.email_verified) {
        return;
    }

    if(isAuthenticated) {
        hasInvites.value = await checkIfHasInvites();
    }

    isAdmin()
});
</script>

<template>
  <header class="global-header">
    <div class="global-header__group">
        <router-link to="/dashboard" class="global-header__logo">Homecall</router-link>

        <div class="global-header__divider" v-if="tenantIdStore.tenantId && isAuthenticated"></div>

        <SelectTenant v-if="tenantIdStore.tenantId && isAuthenticated" />

        <TenantSettings v-if="tenantIdStore.tenantId && isAuthenticated && role === 'admin'" />

        <router-link class="link-btn link-btn--small" to="/dashboard" v-if="tenantIdStore.tenantId && isAuthenticated">
            Enheter
        </router-link>
    </div>

    <div class="global-header__group">
        <router-link class="link-btn global-header__invites" to="/invites" v-if="isAuthenticated">
            <span class="global-header__invites__notif" v-if="hasInvites"></span>
            <font-awesome-icon icon="fa-solid fa-envelope" />
        </router-link>

        <UserMenu v-if="isAuthenticated" />
    </div>
  </header>
</template>

<style lang="scss" scoped>
@import "@/assets/styles/variables.scss";

.global-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: .5rem 1rem;
    background-color: #ffffff;
    box-shadow: 0 1px 4px rgba(0, 0, 0, 0.15);
    z-index: 100;
    position: sticky;
    top: 0;
    width: 100%;

    &__logo {
        font-size: 1.2rem;
        margin: 4px 0;
        font-weight: 500;
        color: #333;
        text-decoration: none;
    }

    &__divider {
        height: 1.5rem;
        width: 1px;
        background-color: #ccc;
        margin: 0 0 0 1rem;
    }

    &__group {
        display: flex;
        align-items: center;
        gap: .5rem;
    }

    &__invites {
        position: relative;
        display: block;
        color: $color-primary;
        font-size: 1.05rem;
        padding: .5rem .8rem;
        margin-right: .6rem;

        &__notif {
            position: absolute;
            top: 10px;
            right: 8px;
            width: 7px;
            height: 7px;
            background-color: red;
            border-radius: 50%;
        }
    }
}
</style>
