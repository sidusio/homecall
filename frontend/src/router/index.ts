import { createRouter, createWebHistory } from 'vue-router'
import { authGuard } from "@auth0/auth0-vue";

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'Login',
      component: () => import('../views/office/LoginView.vue')
    },
    {
      path: '/dashboard',
      name: 'Home',
      component: () => import('../views/office/HomeView.vue'),
      beforeEnter: authGuard,
    },
    {
      path: '/tenants',
      name: 'SelectTenants',
      component: () => import('../views/office/tenants/ChooseView.vue'),
      beforeEnter: authGuard,
    },
    {
      path: '/create-tenant',
      name: 'CreateTenant',
      component: () => import('../views/office/tenants/CreateView.vue'),
      beforeEnter: authGuard,
    },
    {
      path: '/tenant-settings',
      name: 'TenantSettings',
      component: () => import('../views/office/tenants/SettingsView.vue'),
      beforeEnter: authGuard,
    },
    /*{
      path: '/groups',
      name: 'SelectTenants',
      component: () => import('../views/office/HandleTenantView.vue'),
      beforeEnter: authGuard,
    },*/
    {
      path: '/call/:deviceId',
      name: 'Call',
      component: () => import('../views/office/CallView.vue'),
      beforeEnter: authGuard,
    },
    {
      path: '/device',
      name: 'Home (from device)',
      component: () => import('../views/device/HomeView.vue'),
    },
  ]
})

export default router
