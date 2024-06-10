import './assets/main.css'

import { createAuth0 } from '@auth0/auth0-vue';
import { createApp } from 'vue'
import { createPinia } from 'pinia'

import App from './App.vue'
import router from './router'

const app = createApp(App)

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

app.mount('#app')
