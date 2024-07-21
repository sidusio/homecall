<script lang="ts" setup>
    import Office from '@/templates/Office.vue';
    import { tenantClient } from '@/clients';
    import { onMounted, ref } from 'vue';
    import { useAuth0 } from '@auth0/auth0-vue';
    import { useRouter } from 'vue-router';
    import roleTranslation from '@/utils/roles';
    import { useTenantIdStore } from '@/stores/tenantId';

    const { getAccessTokenSilently } = useAuth0();
    const tenantIdStore = useTenantIdStore();
    const router = useRouter();
    const invites = ref<any[]>([]); // TODO: Better type

    /**
     * Get the auth header.
     *
     * @returns The auth header.
     */
    const auth = async () => {
        const token = await getAccessTokenSilently();
        const auth = {
            headers: {
                Authorization: 'Bearer ' + token
            }
        }

        return auth;
    }

    /**
     * Accept an invite. Reloads the page after accepting.
     *
     * @param id - The id of the invite.
     */
    const acceptInvite = async (invite: any) => {
        await tenantClient.acceptTenantInvite({
            id: invite.id
        }, await auth());

        // Add tenantId to the store.
        tenantIdStore.setTenantId(invite.tenantId);

        // Go to dashboard.
        router.push('/dashboard');
    }

    /**
     * Get all invites. Adds the invites to the invites ref.
     */
    const getAllInvites = async () => {
        const { tenantInvites } = await tenantClient.listTenantInvites({}, await auth());

        invites.value = tenantInvites;
    }

    onMounted(async () => {
        await getAllInvites();
    });
</script>

<template>
    <Office>
        <div class="invites">
            <h1 class="invites__title">
                Inbjudningar
            </h1>

            <div v-if="invites.length === 0">
                <p class="invites__no-invites">
                    <font-awesome-icon class="invites__no-invites__icon" icon="fa-solid fa-envelope" />
                    Inga inbjudningar.
                </p>
            </div>

            <div class="invites__cards" v-else>
                <div class="invites__card" v-for="invite in invites" :key="invite.id">
                    <div>
                        <p>Du har blivit inbjuden att bli <strong>{{ roleTranslation(invite.role) }}</strong> i organisationen <strong>{{ invite.tenantName }}</strong>.</p>
                    </div>

                    <button class="btn btn--filled" @click="acceptInvite(invite)">
                        <font-awesome-icon icon="fa-solid fa-check" />
                        Acceptera
                    </button>
                </div>
            </div>
        </div>
    </Office>
</template>

<style lang="scss">
@import "@/assets/styles/variables.scss";

.invites {
    padding: 2rem;

    &__title {
        margin-bottom: 2rem;
    }

    &__no-invites {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: .5rem;
        text-align: center;
        font-size: 1.2rem;
        color: $color-primary;

        &__icon {
            font-size: 2rem;
        }
    }

    &__cards {
        display: flex;
        flex-direction: column;
        gap: 1rem;
    }

    &__card {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 1rem;
        border: 1px solid #ccc;
        border-radius: 5px;
        margin-bottom: 1rem;
    }

    strong {
        font-weight: 700;
    }
}
</style>
