<script lang="ts" setup>
import { useAuth0 } from '@auth0/auth0-vue';
import { useTenantIdStore } from '@/stores/tenantId';
import SelectTenant from '@/components/tenants/SelectTenant.vue';
import UserMenu from '@/components/UserMenu.vue';
import TenantSettings from '@/components/tenants/TenantSettings.vue';

const { isAuthenticated } = useAuth0(); // Check if logged in.
const tenantIdStore = useTenantIdStore(); // Get the tenantId.
</script>

<template>
  <header class="global-header">
    <div class="global-header__group">
        <router-link to="/dashboard" class="global-header__logo">Homecall</router-link>

        <div class="global-header__divider" v-if="tenantIdStore.tenantId && isAuthenticated"></div>

        <SelectTenant v-if="tenantIdStore.tenantId && isAuthenticated" />

        <TenantSettings v-if="tenantIdStore.tenantId && isAuthenticated" />
    </div>

    <div class="global-header__group">
        <router-link class="link-btn link-btn--small" to="/dashboard" v-if="tenantIdStore.tenantId && isAuthenticated">
            Enheter
        </router-link>

        <UserMenu v-if="isAuthenticated" />
    </div>
  </header>
</template>

<style lang="scss" scoped>
.global-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: .5rem 1rem;
    background-color: #ffffff;
    box-shadow: 0 1px 4px rgba(0, 0, 0, 0.1);
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
}
</style>
