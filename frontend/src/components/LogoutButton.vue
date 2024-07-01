<script setup lang="ts">
    import { useAuth0 } from '@auth0/auth0-vue';
    const { logout } = useAuth0();

    const logoutUser = () => {
        localStorage.removeItem('tenantId');
        logout({ logoutParams: { returnTo: window.location.origin } });
    }
</script>

<template>
    <button
        class="link-btn"
        @click="logoutUser"
    >
        Logga ut
    </button>
</template>

<style lang="scss" scoped>
</style>
