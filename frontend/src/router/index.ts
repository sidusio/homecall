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
      path: '/enroll',
      name: 'Enroll Device',
      component: () => import('../views/office/EnrollView.vue')
    },
    {
      path: '/call/:deviceId',
      name: 'Call',
      component: () => import('../views/office/CallView.vue')
    },
    {
      path: '/device',
      name: 'Home (from device)',
      component: () => import('../views/device/HomeView.vue')
    },
    {
      path: '/device/enroll',
      name: 'Enroll Device (from device)',
      component: () => import('../views/device/EnrollView.vue')
    },
    {
      path: '/device/call',
      name: 'Call (from device)',
      component: () => import('../views/device/CallView.vue')
    },

  ]
})

export default router
