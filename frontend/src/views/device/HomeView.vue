<script setup lang="ts">
import { deviceClient } from '@/clients';
import { computed, watch, ref, onMounted } from 'vue'

// Types.
interface Settings {
  autoAnswer: boolean;
  autoAnswerDelaySeconds: bigint;
}

interface Call {
  jitsiRoomId: string;
  jitsiJwt: string;
}

const settings = computed((): Settings => {
  // @ts-ignore - DeviceId is set in the HTML.
  return (window.deviceData.settings || '{}')
})

const incomingCall = ref<Call | null>(null)
const activeCall = ref<Call | null>(null)

let jitsiAPI: {
  addListener(arg0: string, arg1: () => void): unknown;
  dispose: () => void;
} | null = null

// Handle auto answering.
watch(incomingCall, async () => {
  if (!incomingCall.value) {
    return
  }
  if(settings.value.autoAnswer) {
    console.log('Auto answering call (scheduling)')
    setTimeout(() => {
      answerCall()
    }, Number(settings.value.autoAnswerDelaySeconds) * 1000)
  }
})

// Answer the call.
const answerCall = () => {
  console.log('Answering call')
  if(incomingCall.value) {
    activeCall.value = incomingCall.value
    incomingCall.value = null
  }
}

// Watch for active call, handle Jitsi.
watch(activeCall, async () => {
  if(!activeCall.value) {
    if(jitsiAPI !== null) {
      jitsiAPI.dispose()
    }
    jitsiAPI = null
    return
  }

  //@ts-ignore - Jitsi is not typed.
  jitsiAPI = new JitsiMeetExternalAPI(window.config.jitsi.domain, {
    roomName: activeCall.value.jitsiRoomId,
    jwt: activeCall.value.jitsiJwt,
    parentNode: document.querySelector('#meeting'),
    height: '100vh',
    lang: 'sv',
    configOverwrite: {
      requireDisplayName: false,
      prejoinPageEnabled: false,
      disableDeepLinking: true,
      toolbarButtons: [],
      toolbarConfig: {
        alwaysVisible: true,
      },
      hideConferenceSubject: true,
      hideConferenceTimer: true,
      filmstrip: {
        disableResizable: true,
      },
    },
    interfaceConfigOverwrite: {
      MOBILE_APP_PROMO: false,
      LANG_DETECTION: false
    }
  });

  if (jitsiAPI === null) {
    return
  }
  jitsiAPI.addListener('participantLeft', () => {
    activeCall.value = null
  });
})

interface CallData {
  type: string;
  callId: string;
}

const pickupCall = async (data: CallData) => {
  try {
      const token = localStorage.getItem('token')

      const abort = new AbortController()

      const resultStream = await deviceClient.getCallDetails({
        callId: data.callId
      }, {
        headers: {
          Authorization: `Bearer ${token}`
        },
        signal: abort.signal
      })

      /**
       * TODO: Unsure if abort should be used.
      for await (const res of resultStream) {
        abort.abort()
        return res
      }
      */

      return resultStream
    } catch (e) {
      console.error(e)
    }
}

onMounted(async () => {
  document.querySelector('body')?.classList.add('remove-feedback')

  // Listen to fcm event.
  window.addEventListener('fcm', async (event) => {
    // @ts-ignore - Data is set in the injected JS.
    incomingCall.value = await pickupCall(event.detail.data) || null
  })
})
</script>

<template>
  <main class="home device">
    <article class="home__awaiting-call" v-if="!incomingCall && !activeCall">
      <h2>Inget samtal just nu</h2>
    </article>

    <article class="home__awaiting-call" v-if="incomingCall">
      <h2>Nu ringer det!</h2>

      <p v-if="settings.autoAnswer">Samtalet startas automatisk efter {{ settings.autoAnswerDelaySeconds }} sekunder.</p>

      <button class="home__answer" @click="answerCall">
        <img src="@/assets/icons/phone.svg" alt="Svara">
        Svara
      </button>
    </article>

    <div id="meeting"></div>

    <button v-if="activeCall" class="home__hangup" @click="activeCall = null">Avsluta samtal</button>
  </main>
</template>

<style lang="scss">
// Hide the feedback button.
.remove-feedback {
  #sentry-feedback {
    display: none;
  }
}
</style>

<style lang="scss" scoped>
.home {
  height: 100vh;
  position: relative;
  background-color: #002594;

  h1 {
    font-size: 4rem;
  }

  &__hangup {
    position: absolute;
    width: fit-content;
    bottom: 1rem;
    left: 50%;
    transform: translateX(-50%);
    padding: 1rem 1.5rem;
    font-size: 1.2rem;
    background-color: rgb(166, 0, 0);
    color: white;
    border-radius: 30px;
    border: none;
    transition: all 0.3s;

    &:hover {
      background-color: rgb(184, 0, 0);
      box-shadow: 0 0 7px rgb(184, 0, 0);
      cursor: pointer;
    }
  }

  &__awaiting-call {
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    font-size: 2rem;
    text-align: center;
    background-color: white;
    padding: 2rem 4rem;
    border-radius: 30px;
  }

  &__answer {
    display: flex;
    align-items: center;
    padding: 1rem 2rem;
    font-size: 1.5rem;
    background-color: rgb(0, 166, 3);
    color: white;
    border-radius: 30px;
    border: none;
    margin-top: 2rem;
    transition: all 0.3s;
    animation: blink 2s infinite;

    img {
      height: 2rem;
      margin-right: 1rem;
      filter: invert(1);
    }

    &:hover {
      background-color: rgb(0, 184, 3);
      box-shadow: 0 0 7px rgb(0, 184, 3);
      cursor: pointer;
    }

    // Pulsating animation of the green box shadow
    @keyframes blink {
      0% {
        box-shadow: 0 0 20px rgba(0, 166, 3, 0.5);
      }
      50% {
        box-shadow: 0 0 20px rgba(0, 166, 3, 1);
      }
      100% {
        box-shadow: 0 0 20px rgba(0, 166, 3, 0.5);
      }
    }
  }
}
</style>
