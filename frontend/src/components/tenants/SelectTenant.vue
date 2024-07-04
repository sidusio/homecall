<script lang="ts" setup>
import { ref, onMounted, watch } from 'vue';
import { tenantClient } from '@/clients';
import { useAuth0 } from '@auth0/auth0-vue';
import { useRouter } from 'vue-router'
import type { Tenant } from "./../../../gen/connect/homecall/v1alpha/tenant_service_pb";
// @ts-ignore
import { onClickOutside } from '@vueuse/core'

const router = useRouter()
const { getAccessTokenSilently } = useAuth0();
const tenantsList = ref<Tenant[]>([])
const currentTenant = ref<Tenant>({} as Tenant) // TODO: Change typing to a better solution.
const open = ref(false);
const dropdown = ref(null);

/**
 * Close the dropdown when clicking outside.
 */
onClickOutside(dropdown, () => {
    open.value = false;
});

/**
 * Watch the route.
 */
 watch(() => router.currentRoute.value, async () => {
    resetTenantList()
})

/**
 * Watch the tenantId.
 */

/**
 * Toggle the dropdown.
 *
 * @param options - The options to toggle (close or open). If no, toggle the dropdown.
 */
const toggle = (options: string) => {
    if(options === 'close') {
        open.value = false;
        return;
    }

    if(options === 'open') {
        open.value = true;
        return;
    }

    open.value = !open.value;
};

/**
 * Reset the tenant list.
 */
const resetTenantList = async () => {
    toggle('close')
    tenantsList.value = []
    await listTenants()
}

/**
 * Set the tenant.
 *
 * @param tenantId - The id of the tenant.
 */
const setTenant = (tenantId: string) => {
    localStorage.setItem('tenantId', tenantId)
    currentTenant.value = tenantsList.value.find(tenant => tenant.id === tenantId) as Tenant // TODO: Change typing to a better solution.
    open.value = false;
}

/**
 * List all tenants.
 */
const listTenants = async () => {
    const tenantId = localStorage.getItem('tenantId')
    const token = await getAccessTokenSilently();
    const auth = {
        headers: {
            Authorization: 'Bearer ' + token
        }
    }

    const { tenants } = await tenantClient.listTenants({}, auth)
    tenantsList.value = tenants
    currentTenant.value = tenants.find(tenant => tenant.id === tenantId) as Tenant // TODO: Change typing to a better solution.
}

onMounted(async () => {
    await listTenants()
})
</script>

<template>
    <div
        class="select-tenant"
        ref="dropdown"
    >
        <button
            class="link-btn"
            @click="toggle('')"
        >
            {{ currentTenant.name }}

            <font-awesome-icon
                class="link-btn__icon"
                :icon="!open ? 'fa-solid fa-chevron-up' : 'fa-solid fa-chevron-down'"
            />
        </button>

        <div
            class="select-tenant__dropdown"
            :class="{ 'select-tenant__dropdown--open': open }"
        >
            <button
                class="select-tenant__dropdown__item link-btn"
                :class="{ 'link-btn--active': currentTenant.id === tenant.id }"
                v-for="tenant in tenantsList"
                @click="setTenant(tenant.id)"
            >
                <span>{{ tenant.name }}</span>
                <font-awesome-icon
                    class="link-btn__icon"
                    icon="fa-solid fa-chevron-right"
                    v-if="currentTenant.id !== tenant.id"
                />
            </button>

            <router-link class="btn btn--outlined" href="#" to="/create-tenant">
                <font-awesome-icon
                    class="btn__icon"
                    icon="fa-solid fa-plus"
                />

                Skapa ny organisation
            </router-link>
        </div>
    </div>
</template>

<style lang="scss" scoped>
.select-tenant {
    position: relative;

    &__dropdown {
        position: absolute;
        width: max-content;
        top: 2.5rem;
        left: 0;
        padding: 1rem;
        background-color: #fff;
        box-shadow: 0 1px 4px rgba(0, 0, 0, 0.1);
        border-radius: 5px;
        display: none;

        &--open {
            display: flex;
            flex-direction: column;
            gap: 0.5rem;
        }

        &__item {
            width: 100%;
            display: flex;
            justify-content: space-between;
        }

        .btn {
            width: 100%;
        }
    }
}
</style>
