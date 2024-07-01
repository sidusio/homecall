<script setup lang="ts">
import { useAuth0 } from '@auth0/auth0-vue';

const { loginWithRedirect } = useAuth0();

const login = () => {
    loginWithRedirect();
}

const signup = () => {
    loginWithRedirect({authorizationParams: {
        screen_hint: "signup",
    }});
}
</script>

<template>
    <main class="login">
        <div class="login__left">
            <h1 class="login__title">
                Logga in / Skapa konto
            </h1>

            <p class="login__text">
                Du kommer att bli omdirigerad till en inloggningssida där du kan logga in med ditt konto.
            </p>

            <div class="login__btns">
                <button
                    class="login__btn login__btn--filled"
                    @click="login"
                >
                    Logga in
                </button>

                <button
                    class="login__btn login__btn--outlined"
                    @click="signup"
                >
                    Registrera dig
                </button>
            </div>
        </div>

        <div class="login__right">
        </div>
    </main>
</template>

<style lang="scss" scoped>
@import "@/assets/styles/variables.scss";

.login {
    height: $viewport-height;
    width: 100vw;
    display: flex;

    &__left {
        width: 60%;
        height: 100%;
        display: flex;
        flex-direction: column;
        justify-content: center;
        padding-left: 5rem;
    }

    &__right {
        width: 40%;
        height: 100%;
        background-color: #f0f0f0;
    }

    &__title {
        margin-bottom: 1rem;
        font-size: 3rem;
        line-height: 1.3;
        color: #000A2E;

        span {
            font-weight: bolder;
        }
    }

    &__text {
        max-width: 60%;
        margin-bottom: 1.5rem;
        color: #000A2E;
    }

    &__btns {
        display: flex;
        gap: 1rem;
    }

    &__btn {
        width: fit-content;
        padding: .8rem 2rem;
        font-size: 1.1rem;
        border: none;
        border-radius: 30px;
        cursor: pointer;

        &--filled {
            background-color: #002594;
            color: white;
            transition: all 0.3s ease-in-out;

            &:hover {
                background-color: #001f6d;
            }
        }

        &--outlined {
            background-color: white;
            color: #002594;
            border: 1px solid #002594;
            margin-left: 1rem;
            transition: all 0.3s ease-in-out;

            &:hover {
                background-color: #002594;
                color: white;
            }
        }
    }
}
</style>
