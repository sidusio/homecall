<script lang="ts" setup>
import { useRouter } from 'vue-router'

const router = useRouter()
defineProps(['tenants'])

const chooseTenant = (id: string) => {
    localStorage.setItem('tenantId', id)

    router.push({ path: '/dashboard' })
}
</script>

<template>
    <div class="list-tenants">
        <section class="list-tenants__tenants" v-if="tenants && tenants.length > 0">
            <button
                @click="chooseTenant(tenant.id)"
                class="list-tenants__tenant link-btn"
                v-for="tenant in tenants"
            >
                <div>
                    {{ tenant.name }}
                </div>

                <font-awesome-icon
                    class="link-btn__icon"
                    icon="fa-solid fa-chevron-right"
                />
            </button>
        </section>
    </div>
</template>

<style lang="scss">
.list-tenants {
    &__title {
        margin-bottom: 1rem;
    }

    &__tenants {
        display: flex;
        flex-direction: column;
        gap: 1rem;
    }

    &__tenant {
        width: 100%;
        justify-content: space-between;
    }
}
</style>
