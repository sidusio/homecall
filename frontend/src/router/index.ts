import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'Home',
      component: () => import('../views/office/HomeView.vue')
    },
    {
      path: '/enroll-device',
      name: 'Enroll Device',
      component: () => import('../views/office/EnrollDevice.vue')
    },
    {
      path: '/call',
      name: 'Call',
      component: () => import('../views/office/CallView.vue')
    },
    {
      path: '/device',
      name: 'Client Home',
      component: () => import('../views/device/HomeView.vue')
    },
    {
      path: '/device/call',
      name: 'Client Call',
      component: () => import('../views/device/CallView.vue')
    },

  ]
})

export default router
