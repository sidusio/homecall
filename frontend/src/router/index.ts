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
      path: '/call/:deviceId',
      name: 'Call',
      component: () => import('../views/office/CallView.vue')
    },
    {
      path: '/device',
      name: 'Home (from device)',
      component: () => import('../views/device/HomeView.vue'),
      beforeEnter: (to, from, next) => {
        // Redirect to /device/enroll if no device ID is set.
        if (!localStorage.getItem('deviceId')) {
          next('/device/enroll')
          return
        }
        next()
      }
    },
    {
      path: '/device/enroll',
      name: 'Enroll Device (from device)',
      component: () => import('../views/device/EnrollView.vue'),
      beforeEnter: (to, from, next) => {
        // Redirect to /device if device ID is already set.
        if (localStorage.getItem('deviceId')) {
          next('/device')
          return
        }
        next()
      }
    },
  ]
})

export default router
