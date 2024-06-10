<script lang="ts" setup>
import { onMounted, ref } from 'vue';
import { tenantClient } from '@/clients';
import { useAuth0 } from '@auth0/auth0-vue';
import CreateTenant from '@/components/CreateTenant.vue';
import ListTenants from '@/components/ListTenants.vue';

const { getAccessTokenSilently, user } = useAuth0();

const tenantsList = ref()

onMounted(async () => {
  const token = await getAccessTokenSilently();
  const auth = {
      headers: {
      Authorization: 'Bearer ' + token
      }
  }

  const { tenants } = await tenantClient.listTenants({}, auth)
  tenantsList.value = tenants
})
</script>

<template>
    <main class="choose-tenant">
      <ListTenants v-if="tenantsList" :tenants="tenantsList" />

      <CreateTenant v-else />
    </main>
</template>

<style lang="scss">
.choose-tenant {
  height: 100vh;
  width: 100vw;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
}
</style>
