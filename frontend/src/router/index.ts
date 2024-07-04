import { createRouter, createWebHistory } from 'vue-router'
import { authGuard, useAuth0 } from "@auth0/auth0-vue";

/**
 * Guard for login page.
 *
 * @returns {Object} The route to redirect to.
 */
const guardLogin = () => {
  const { isAuthenticated } = useAuth0();

  // TODO: This blocks login currently, that is not the intention.
  if (isAuthenticated) {
    return { name: 'Home' };
  } else {
    return { name: 'Login' };
  }
}

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'Login',
      component: () => import('../views/office/LoginView.vue'),
      beforeEnter: guardLogin,
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
