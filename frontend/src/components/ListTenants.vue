<script lang="ts" setup>
import { useRouter } from 'vue-router'

const router = useRouter()
const props = defineProps(['tenants'])

const chooseTenant = (id: string) => {
    localStorage.setItem('tenantId', id)

    router.push({ path: '/dashboard' })
}
</script>

<template>
    <div class="list-tenants">
        <h1>Välj grupp</h1>

        <section v-if="tenants.length > 0">
            <article v-for="tenant in tenants">
                <div>
                    {{ tenant.name }}
                </div>

                <button @click="chooseTenant(tenant.id)">Välj</button>
            </article>
        </section>
    </div>
</template>

<style lang="scss">
</style>
