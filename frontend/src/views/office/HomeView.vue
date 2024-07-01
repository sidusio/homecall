<script setup lang="ts">
import { officeClient } from '@/clients';
import { onMounted, ref, watch } from 'vue';
import EnrollDevice from '@/components/EnrollDevice.vue';
import RegisterDevice from '@/components/RegisterDevice.vue';
import { useAuth0 } from '@auth0/auth0-vue';

const { getAccessTokenSilently } = useAuth0();

interface Device {
  id: string;
  name: string;
  enrollmentKey: string;
  online: boolean;
}

interface Enrollment {
  enrollmentKey: string;
  deviceId: string;
}

const registerDevice = ref(false)
const enrollment = ref<Enrollment | null>(null)
const devices = ref<Device[]>([])
const recentlyModifiedDeviceId = ref<string | null>(null)

/**
 * List all devices.
 */
const listDevices = async () => {
  const tenantId = localStorage.getItem('tenantId')

  if(!tenantId) {
    return;
  }

  const token = await getAccessTokenSilently();
  const auth = {
    headers: {
      Authorization: 'Bearer ' + token
    }
  }

  const res = await officeClient.listDevices({
    tenantId: tenantId
  }, auth)

  devices.value = res.devices.sort((a, b) => a.name.localeCompare(b.name))
}

/**
 * Clear the enrollment.
 */
const clearEnrollment = () => {
  enrollment.value = null
}

/**
 * Handle the registration of a device.
 *
 * @param event - The enrollment event.
 */
const handleRegistration = (event: Enrollment) => {
  enrollment.value = event
  registerDevice.value = false
  recentlyModifiedDeviceId.value = event.deviceId
  listDevices()
}

/**
 * Handle the enrollment of a device.
 *
 * @param deviceId - The id of the device.
 */
const handleEnrollment = (deviceId: string) => {
  clearEnrollment()
  recentlyModifiedDeviceId.value = deviceId
  listDevices()
}

/**
 * Toggle the removal of a device.
 *
 * @param deviceId - The id of the device.
 */
const toggleRemoveDevice = (deviceId: string) => {
  const removeDiv = document.getElementById(`remove-${deviceId}`)
  if (removeDiv) {
    removeDiv.style.display = removeDiv.style.display === 'none' ? 'flex' : 'none'
  }
}

/**
 * Remove a device.
 *
 * @param deviceId - The id of the device.
 */
const removeDevice = async (deviceId: string) => {
  const tenantId = localStorage.getItem('tenantId')

  if(!tenantId) {
    return;
  }

  const token = await getAccessTokenSilently();
  const auth = {
    headers: {
      Authorization: 'Bearer ' + token
    }
  }

  officeClient.removeDevice({ deviceId }, auth)
  listDevices()
}

onMounted(async () => {
  // Poll for devices every minute.
  setInterval(() => {
    listDevices()
  }, 10000)

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

        <ul class="home__devices" v-if="devices.length > 0">
          <li
            class="home__device"
            v-for="device in devices"
            :key="device.id"
            :class="recentlyModifiedDeviceId === device.id ? 'home__device--recently-modified' : ''"
          >
            <div class="home__device-info" @click="toggleRemoveDevice(device.id)">
              <span class="home__device--small">
                {{ device.id ? device.id : 'Ingen ID'}}
              </span>
              <div class="home__device--row">
                {{ device.name }}
              </div>

              <div class="home__device-remove" :id="'remove-' + device.id">
                <button @click="removeDevice(device.id)">Ta bort</button>
              </div>
            </div>

            <router-link
              v-if="!device.enrollmentKey"
              class="home__device-btn home__device-call"
              :class="device.online ? 'home__device-call--online' : 'home__device-call--offline'"
              :to="`/call/${device.id}`"
            >
              <img src="@/assets/icons/phone.svg">
            </router-link>

            <button
              v-else
              class="home__device-btn home__device-qrcode"
              @click="handleRegistration({ enrollmentKey: device.enrollmentKey, deviceId: device.id })"
            >
              <img src="@/assets/icons/qrcode-solid.svg">
            </button>
          </li>
        </ul>

        <p class="home__no-devices" v-else>
          Du har inga enheter
        </p>
      </div>

      <button
        class="home__register"
        @click="registerDevice = true"
      >
        Registrera enhet
      </button>
    </aside>

    <main class="home__dashboard">
      <RegisterDevice
        v-if="registerDevice"
        @registered="handleRegistration"
      ></RegisterDevice>

      <EnrollDevice
        v-else-if="enrollment"
        :enrollment="enrollment"
        @close="clearEnrollment"
        @enrolled="handleEnrollment"
      ></EnrollDevice>

      <div v-else>
        <h1>Välkommen till Homecall</h1>
        <p>Välj en enhet att ringa till eller registrera en ny enhet.</p>

        <p class="tip">För att ta bort en enhet, klicka på den och välj "Ta bort".</p>
      </div>
    </main>
  </div>
</template>

<style lang="scss" scoped>
@import "@/assets/styles/variables.scss";

.home {
  height: $viewport-height;
  display: flex;
  gap: 1rem;

  &__side {
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    width: 350px;
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

  &__register {
    background-color: #002594;
    color: rgb(255, 255, 255);
    padding: 1rem;
    margin: 1rem;
    text-align: center;
    font-size: 1rem;
    border-radius: 30px;
    text-decoration: none;
    transition: all 0.3s;
    border: none;

    &:hover {
      background-color: #001f6d;
      color: white;
      cursor: pointer;
    }
  }

  &__no-devices {
    padding: 1rem;
    opacity: .6;
    text-align: center;
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

    &--row {
      display: flex;
      align-items: center;
    }

    &--recently-modified {
      background-color: rgb(67, 107, 177, 0.1);
    }

    &-info {
      position: relative;
      display: flex;
      flex-direction: column;
    }

    &-btn {
      color: white;
      padding: 0.75rem;
      border-radius: 50%;
      display: flex;
      justify-content: center;
      align-items: center;
      transition: all 0.3s;
      border: none;

      img {
        width: 1.25rem;
        filter: invert(100%);
      }
    }

    &-qrcode {
      background-color: rgb(67, 107, 177);

      &:hover {
        cursor: pointer;
        background-color: rgb(67, 107, 177, 0.9);
      }
    }

    &-call {
      background-color: rgb(0, 166, 3);

      &:hover {
        background-color: rgb(0, 184, 3);
        box-shadow: 0 0 7px rgb(0, 184, 3);
      }

      &--offline {
        background-color: rgb(165, 165, 165);

        &:hover {
          background-color: rgb(165, 165, 165, 0.9);
          box-shadow: none;
        }
      }
    }

    &-remove {
      display: none;
      position: absolute;
      top: 0;
      right: 0;
      width: 100%;
      height: 100%;
      justify-content: center;
      align-items: center;
      background-color: rgba(255, 255, 255, 0.5);

      button {
        background-color: rgb(255, 0, 0);
        color: white;
        padding: 0.5rem 1rem;
        border-radius: 5px;
        border: none;
        transition: all 0.3s;

        &:hover {
          background-color: rgb(255, 0, 0, 0.9);
          cursor: pointer;
        }
      }
    }
  }

  .tip {
    font-size: 0.8rem;
    background-color: rgb(67, 107, 177, 0.1);
    padding: 1rem;
    border-radius: 10px;
    margin-top: 3rem;
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
