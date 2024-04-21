<script setup lang="ts">
import { officeClient } from '@/clients';
import { onMounted, ref } from 'vue';

interface Device {
  id: string;
  name: string;
}

const devices = ref<Device[]>([])

const listDevices = async () => {
  const res = await officeClient.listDevices()

  devices.value = res.devices
}

onMounted(() => {
  listDevices()
})
</script>

<template>
  <div class="home">
    <aside class="home__side">
      <div class="home__device-container">
        <div class="home__device-header">
          <h1>Enheter</h1>

          <p>{{ devices.length }} enheter</p>
        </div>

        <ul class="home__devices">
          <li
            class="home__device"
            v-for="device in devices"
            :key="device.id"
          >
            <div class="home__device-info">
              <span class="home__device--small">
                {{ device.id ? device.id : 'Ingen ID'}}
              </span>
              {{ device.name }}
            </div>

            <router-link
              class="home__device-call"
              :to="`/call/${device.id}`"
            >
              <img src="@/assets/icons/phone.svg">
            </router-link>
          </li>
        </ul>
      </div>

      <router-link
        class="home__enroll"
        to="/enroll"
      >
        Registrera enhet
      </router-link>
    </aside>

    <main class="home__dashboard">
      <p>
        Här till vänster hittar du alla enheter som är registerade.
        <br/>
        Klicka på grön lur tillhörande enheten för att ringa.
      </p>
    </main>
  </div>
</template>

<style lang="scss" scoped>
.home {
  height: 100vh;
  display: flex;
  gap: 1rem;

  &__side {
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    width: 400px;
    background-color: white;
    box-shadow: 0 0 7px rgba(0, 0, 0, 0.1);
  }

  &__device-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1rem 1rem 0 1rem;

    p {
      font-size: 0.8rem;
      color: gray;
    }
  }

  &__enroll {
    background-color: rgb(67, 107, 177);
    color: rgb(255, 255, 255);
    padding: 1rem;
    margin: 1rem;
    text-align: center;
    border-radius: 30px;
    text-decoration: none;
    transition: all 0.3s;

    &:hover {
      background-color: rgb(67, 107, 177, 0.9);
      color: white;
    }
  }

  &__devices {
    list-style: none;
    padding: 0;
    margin: 1rem 0 0 0;

    max-height: 80vh;
    overflow: auto;
  }

  &__device {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 2rem;
    border-top: 1px solid rgba(0, 0, 0, 0.1);
    padding: 1rem;

    &--small {
      font-size: 0.8rem;
      color: gray;
    }

    &-info {
      display: flex;
      flex-direction: column;
    }

    &-call {
      background-color: rgb(0, 166, 3);
      color: white;
      padding: 0.75rem;
      border-radius: 50%;
      display: flex;
      justify-content: center;
      align-items: center;
      transition: all 0.3s;

      img {
        width: 1.25rem;
        filter: invert(100%);
      }

      &:hover {
        background-color: rgb(0, 184, 3);
        box-shadow: 0 0 7px rgb(0, 184, 3);
      }
    }
  }

  @media only screen and (max-width: 768px) {
    &__dashboard {
      display: none !important;
    }
  }

  &__dashboard {
    display: flex;
    justify-content: center;
    align-items: center;
    flex: 1;
    padding: 2rem;
    text-align: center;
  }
}
</style>
