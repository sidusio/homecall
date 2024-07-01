import './assets/styles/main.scss'

import { createAuth0 } from '@auth0/auth0-vue';
import { createApp } from 'vue'
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

app.component('font-awesome-icon', FontAwesomeIcon)

app.mount('#app')
