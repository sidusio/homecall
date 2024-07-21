<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useAuth0 } from '@auth0/auth0-vue';
const { user, logout } = useAuth0();

const alert = ref(false);
const countDown = ref(5);

/**
 * Set interval to count down.
 */
const countDownInterval = setInterval(() => {
    countDown.value -= 1;
}, 1000);

onMounted(() => {
    // Set interval to reload page every 5 seconds
    const intervalId = setInterval(() => {
        window.location.reload();

        if(!user.value) {
            return;
        }

        if(user.value.email_verified) {
            clearInterval(intervalId);
            logout({ logoutParams: { returnTo: window.location.origin + "?verified_email=true" } });
        }
    }, 5000);

    countDownInterval;
})
</script>

<template>
    <div class="verify-email">
        <div class="verify-email__container">
            <h1>Verifiera din e-post!</h1>

            <p>
                Vi har skickat ett verifieringsmail till din e-postadress. Klicka på länken i mailet för att verifiera din e-postadress.
            </p>

            <p>Kollar ifall du har verifierat om {{ countDown }} sek.</p>

            <div class="verify-email__notif" v-if="alert">
                <p>
                    <font-awesome-icon icon="fa-solid fa-exclamation-triangle" />
                    Din mejl verkar inte vara verifierad. Kolla din inkorg för att verifiera din mejl.
                </p>
            </div>
        </div>
    </div>
</template>

<style lang="scss" scoped>
@import "@/assets/styles/variables.scss";

.verify-email {
    height: 100vh;
    display: flex;
    justify-content: center;
    align-items: center;

    &__notif {
        position: absolute;
        bottom: 2rem;
        animation: fadeIn 1s ease-in-out;

        p {
            display: flex;
            align-items: center;
            gap: 1rem;
            margin: 0;
            padding: .5rem 1.5rem;
            background-color: $color-danger;
            color: white;
            border-radius: 30px;
        }
    }

    &__container {
        max-width: 50%;
        display: flex;
        flex-direction: column;
        align-items: center;
        background-color: #ffffff;
        padding: 2rem;
        text-align: center;

        h1 {
            margin-bottom: 1rem;
        }

        p {
            margin-bottom: 2rem;
        }
    }
}

@keyframes fadeIn {
    from {
        opacity: 0;
    }

    to {
        opacity: 1;
    }
}
</style>
