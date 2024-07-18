<script lang="ts" setup>
import RemoveTenant from '@/components/tenants/RemoveTenant.vue';
import AddMembers from '@/components/tenants/AddMembers.vue';
import Office from '@/templates/Office.vue';
</script>

<template>
    <Office>
        <main class="settings fill-height">
            <h1 class="settings__title">
                Inställningar
            </h1>

            <AddMembers />

            <hr class="settings__divider" />

            <RemoveTenant />
        </main>
    </Office>
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
