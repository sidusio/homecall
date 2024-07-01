<script lang="ts" setup>
import { ref } from 'vue';
import { useAuth0 } from '@auth0/auth0-vue';
import LogoutButton from '@/components/LogoutButton.vue';

const { user } = useAuth0();

const open = ref(false);

const toggle = () => {
    open.value = !open.value;
};
</script>

<template>
    <div class="user-menu">
        <button
            class="user-menu__user"
            @click="toggle"
        >
            <img
                class="user-menu__user__avatar"
                :src="user.picture"
                alt="User avatar"
            />
        </button>

        <div
            class="user-menu__dropdown"
            :class="{ 'user-menu__dropdown--open': open }"
        >
            <LogoutButton />
        </div>
    </div>
</template>

<style lang="scss" scoped>
.user-menu {
    position: relative;

    &__user {
        background-color: transparent;
        border: none;
        border-radius: 50%;

        &:hover {
            cursor: pointer;
        }

        &__avatar {
            width: 2rem;
            height: 2rem;
            border-radius: 50%;
        }
    }

    &__dropdown {
        position: absolute;
        width: 13rem;
        top: 2.5rem;
        right: 0;
        padding: 1rem;
        background-color: #fff;
        box-shadow: 0 1px 4px rgba(0, 0, 0, 0.1);
        border-radius: 5px;
        display: none;

        &--open {
            display: block;
        }
    }
}
</style>
