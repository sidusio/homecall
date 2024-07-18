<script lang="ts" setup>
import Office from '@/templates/Office.vue';
import { onMounted, ref } from 'vue';
import { tenantClient } from '@/clients';
import ListTenants from '@/components/tenants/ListTenants.vue';
import { useAuth0 } from '@auth0/auth0-vue';
import { useRouter } from 'vue-router';
import { useTenantIdStore } from '@/stores/tenantId';

const router = useRouter()
const { getAccessTokenSilently } = useAuth0();
const { setTenantId } = useTenantIdStore();
const tenantsList = ref()

onMounted(async () => {
  const token = await getAccessTokenSilently();
  const auth = {
      headers: {
          Authorization: 'Bearer ' + token
      }
  }

  const { tenants } = await tenantClient.listTenants({}, auth)

  if(tenants.length === 1) {
    setTenantId(tenants[0].id)
    router.push('/dashboard')
  }

  tenantsList.value = tenants
})
</script>

<template>
    <Office>
      <main class="choose-tenant">
        <h1>Välj organisation</h1>

        <div class="choose-tenant__content">
          <ListTenants v-if="tenantsList && tenantsList.length > 0" :tenants="tenantsList" />

          <div class="choose-tenant__empty-container" v-else>
            <p class="choose-tenant__empty">
              Inga organisationer hittades.
            </p>

            <button class="btn btn--filled" @click="$router.push('/create-tenant')">Skapa ny organisation</button>
          </div>
        </div>
      </main>
    </Office>
</template>

<style lang="scss">
@import '@/assets/styles/variables.scss';

.choose-tenant {
  height: $viewport-height;
  width: 100vw;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;

  &__content {
    margin-top: 2rem;
  }

  &__empty-container {
    display: flex;
    flex-direction: column;
    align-items: center;
  }

  &__empty {
    margin-bottom: 2.5rem;
    font-size: 1.1rem;
  }
}
</style>
