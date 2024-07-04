<script lang="ts" setup>
import { ref, onMounted } from 'vue';
import { useAuth0 } from '@auth0/auth0-vue';
import SelectTenant from '@/components/tenants/SelectTenant.vue';
import UserMenu from '@/components/UserMenu.vue';
import TenantSettings from '@/components/tenants/TenantSettings.vue';
import { useRouter } from 'vue-router';

// Check if logged in
const { isAuthenticated } = useAuth0();

const router = useRouter();

// A value that listens to tenantId in localStorage
const tenantId = ref(false);

// On route change, check if tenantId is in localStorage
onMounted(() => {
    router.afterEach(() => {
        const tenantIdValue = localStorage.getItem('tenantId');
        tenantId.value = tenantIdValue ? true : false;
    });
});
</script>

<template>
  <header class="global-header">
    <div class="global-header__group">
        <router-link to="/dashboard" class="global-header__logo">Homecall</router-link>

        <div class="global-header__divider"></div>

        <SelectTenant v-if="tenantId && isAuthenticated" />

        <TenantSettings v-if="tenantId && isAuthenticated" />
    </div>

    <div class="global-header__group">
        <router-link class="link-btn link-btn--small" to="/dashboard" v-if="tenantId && isAuthenticated">
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
