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
      path: '/call',
      name: 'Call',
      component: () => import('../views/office/CallView.vue')
    },
    {
      path: '/client',
      name: 'Client Home',
      component: () => import('../views/client/HomeView.vue')
    },
    {
      path: '/client/call',
      name: 'Client Call',
      component: () => import('../views/client/CallView.vue')
    },

  ]
})

export default router
