import './assets/styles/main.scss'

import { createAuth0 } from '@auth0/auth0-vue';
import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import * as Sentry from "@sentry/vue";
import { createPinia } from 'pinia'

/* import the fontawesome core */
import { library } from '@fortawesome/fontawesome-svg-core'

/* import font awesome icon component */
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'

/* import specific icons */
import { faChevronDown, faChevronUp, faChevronRight, faPlus, faGear, faPen, faTrash } from '@fortawesome/free-solid-svg-icons'

/* add icons to the library */
library.add(faChevronDown)
library.add(faChevronUp)
library.add(faChevronRight)
library.add(faPlus)
library.add(faGear)
library.add(faPen)
library.add(faTrash)

const app = createApp(App)

Sentry.init({
    app,
    dsn: "https://9639d3406a54364151d90077a1a2020b@o4507538136170496.ingest.de.sentry.io/4507538144755792",
    integrations: [
      Sentry.browserTracingIntegration(),
      Sentry.replayIntegration(),
    ],
    // Performance Monitoring
    tracesSampleRate: 1.0, //  Capture 100% of the transactions
    // Set 'tracePropagationTargets' to control for which URLs distributed tracing should be enabled
    tracePropagationTargets: ["localhost", /^https:\/\/yourserver\.io\/api/],
    // Session Replay
    replaysSessionSampleRate: 0.1, // This sets the sample rate at 10%. You may want to change it to 100% while in development and then sample at a lower rate in production.
    replaysOnErrorSampleRate: 1.0, // If you're not already sampling the entire session, change the sample rate to 100% when sampling sessions where errors occur.
});

app.use(
    createAuth0({
        domain: "homecall.eu.auth0.com",
        clientId: "t0SCUNEPaCWyLVzVMbgj7TmVVgqDqJEN",
        authorizationParams: {
            redirect_uri: window.location.origin + '/tenants',
            audience: "https://office-api.homecall.sidus.io",
        }
    })
);
app.use(createPinia())
app.use(router)

app.component('font-awesome-icon', FontAwesomeIcon)

app.mount('#app')
