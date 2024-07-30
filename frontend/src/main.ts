import './assets/styles/main.scss'
import { getConfig, allOptionsDefined, type Config } from './utils/config';

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
import { faChevronDown, faChevronUp, faChevronRight, faPlus, faGear, faPen, faTrash, faEnvelope, faCheck, faExclamationTriangle } from '@fortawesome/free-solid-svg-icons'

/* add icons to the library */
library.add([faGear, faPlus, faChevronRight, faChevronUp, faChevronDown, faPen, faTrash, faEnvelope, faCheck, faExclamationTriangle])

const app = createApp(App)

const config = await getConfig();

if(!allOptionsDefined(config)) {
  console.log({config})
  throw new Error('Not all options are defined in the config');
}
declare global {
  interface Window {
    config: Config;
  }
}
window.config = config;

Sentry.init({
    app,
    dsn: config.sentry.dsn,
    integrations: [
      Sentry.browserTracingIntegration(),
      Sentry.replayIntegration(),
      Sentry.feedbackIntegration({
        // Additional SDK configuration goes in here, for example:
        colorScheme: "system",
        showBranding: false,
        triggerLabel: "Ge oss feedback",
        formTitle: "Ge oss feedback",
        nameLabel: "Namn",
        namePlaceholder: "Skriv ditt namn här...",
        emailLabel: "E-post",
        emailPlaceholder: "Skriv din e-post här...",
        messageLabel: "Meddelande",
        messagePlaceholder: "Skriv ditt meddelande här...",
        isRequiredLabel: "*",
        addScreenshotButtonLabel: "Lägg till skärmbild",
        removeScreenshotButtonLabel: "Ta bort skärmbild",
        submitButtonLabel: "Skicka feedback",
        cancelButtonLabel: "Avbryt",
        successMessageText: "Tack för din feedback!",
      }),
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
    domain: config.auth0.domain,
    clientId: config.auth0.clientId,
    authorizationParams: {
      redirect_uri: config.auth0.redirectUri,
      audience: config.auth0.audience,
    }
  })
);
app.use(createPinia())
app.use(router)

app.component('font-awesome-icon', FontAwesomeIcon)

app.mount('#app')
