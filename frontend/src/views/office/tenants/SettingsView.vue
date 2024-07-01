<script lang="ts" setup>
import RemoveTenant from '@/components/tenants/RemoveTenant.vue';
import AddMembers from '@/components/tenants/AddMembers.vue';
// Be able to change name

// Add members to tenant
</script>

<template>
    <main class="settings fill-height">
        <h1 class="settings__title">
            Inställningar
        </h1>

        <AddMembers />

        <hr class="settings__divider" />

        <RemoveTenant />
    </main>
</template>

<style lang="scss">
.settings {
    padding: 2rem;

    &__title {
        margin-bottom: 2rem;
    }

    &__divider {
        margin: 4rem 0;
        border: 1px solid rgba(0, 0, 0, 0.05);
    }
}
</style>
