<script setup lang="ts">
    import { useAuth0 } from '@auth0/auth0-vue';
    const { logout } = useAuth0();

    const logoutUser = () => {
        logout({ logoutParams: { returnTo: window.location.origin } });
    }
</script>

<template>
    <button
        class="logout-btn"
        @click="logoutUser"
    >
        Log out
    </button>
</template>

<style lang="scss">
.logout-btn {
    position: absolute;
    top: 1rem;
    right: 1rem;
}
</style>
