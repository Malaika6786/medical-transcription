import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import vuetify from './plugins/vuetify'

// Material Design Icons — bundled locally so icons render on networks that
// block external font CDNs (was previously loaded from a CDN in index.html).
import '@mdi/font/css/materialdesignicons.css'
import './styles/main.scss'

const app = createApp(App)

app.use(createPinia())
app.use(router)
app.use(vuetify)

app.mount('#app')

