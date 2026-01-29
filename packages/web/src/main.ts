import { createApp } from 'vue'
// @ts-ignore
import './style.css'
import App from './App.vue'
import router from './router'
import { createPinia } from 'pinia'
import i18n from './i18n'
import { PiniaColada } from '@pinia/colada'
import piniaPluginPersistedstate from 'pinia-plugin-persistedstate'
// @ts-ignore
import 'markstream-vue/index.css'
// @ts-ignore
import 'katex/dist/katex.min.css'

createApp(App)
  .use(createPinia().use(piniaPluginPersistedstate))
  .use(PiniaColada)
  .use(router)
  .use(i18n)
  .mount('#app')