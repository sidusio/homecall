<script setup lang="ts">
</script>

<template>
    <div class="loading">
        <div class="loading__spinner"></div>
        <p>Laddar...</p>
    </div>
</template>

<style lang="scss" scoped>
@import "@/assets/styles/variables.scss";
.loading {
    height: 100vh;
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
    gap: .5rem;

    &__spinner {
        border: 8px solid $color-primary-opacity-10;
        border-left: 8px solid $color-primary;
        border-radius: 50%;
        width: 50px;
        height: 50px;
        animation: spin 1s linear infinite;
    }

    p {
        color: $color-primary;
    }

    @keyframes spin {
        0% {
            transform: rotate(0deg);
        }
        100% {
            transform: rotate(360deg);
        }
    }
}
</style>
