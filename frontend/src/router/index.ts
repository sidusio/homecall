import { createRouter, createWebHistory } from 'vue-router'
import { authGuard, useAuth0 } from "@auth0/auth0-vue";

/**
 * Check if the user has verified their email address.
 */
// @ts-ignore
const checkVerifiedEmail = (to, from, next) => {
  const { user } = useAuth0();

  if(!user.value) {
    return;
  }

  if (!user.value.email_verified) {
    next({ name: 'VerifyEmail' });
  } else {
    next();
  }
}

/**
 * Check if the user has selected a tenant.
 */
// @ts-ignore
const checkTenantId = (to, from, next) => {
  const tenantId = localStorage.getItem('tenantId');

  if(tenantId === null) {
    next({ name: 'SelectTenants' });
  } else {
    next();
  }
}

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'Login',
      component: () => import('../views/office/LoginView.vue'),
      beforeEnter: (to, from, next) => {
        const { user } = useAuth0();

        if(user !== undefined) {
          next({ name: 'Home' });
        }

        next({ name: 'Login' });
      },
    },
    {
      path: '/verify-email',
      name: 'VerifyEmail',
      component: () => import('../views/office/VerifyEmailView.vue'),
      beforeEnter: [authGuard],
    },
    {
      path: '/dashboard',
      name: 'Home',
      component: () => import('../views/office/HomeView.vue'),
      beforeEnter: [authGuard, checkVerifiedEmail, checkTenantId],
    },
    {
      path: '/invites',
      name: 'Invites',
      component: () => import('../views/office/members/InvitesView.vue'),
      beforeEnter: [authGuard, checkVerifiedEmail],
    },
    {
      path: '/tenants',
      name: 'SelectTenants',
      component: () => import('../views/office/tenants/ChooseView.vue'),
      beforeEnter: [authGuard, checkVerifiedEmail],
    },
    {
      path: '/create-tenant',
      name: 'CreateTenant',
      component: () => import('../views/office/tenants/CreateView.vue'),
      beforeEnter: [authGuard, checkVerifiedEmail],
    },
    {
      path: '/tenant-settings',
      name: 'TenantSettings',
      component: () => import('../views/office/tenants/SettingsView.vue'),
      beforeEnter: [authGuard, checkVerifiedEmail],
    },
    {
      path: '/call/:deviceId',
      name: 'Call',
      component: () => import('../views/office/CallView.vue'),
      beforeEnter: [authGuard, checkVerifiedEmail],
    },
    {
      path: '/enhet/:deviceId',
      name: 'Device',
      component: () => import('../views/office/DeviceView.vue'),
      beforeEnter: [authGuard, checkVerifiedEmail],
    },
    {
      path: '/device',
      name: 'Home (from device)',
      component: () => import('../views/device/HomeView.vue'),
    },
  ]
})

export default router
