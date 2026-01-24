import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
      meta: { requiresAuth: false }
    },
    {
      path: '/',
      component: () => import('@/components/layout/AppLayout.vue'),
      meta: { requiresAuth: true },
      children: [
        {
          path: '',
          name: 'dashboard',
          component: () => import('@/views/DashboardView.vue')
        },
        {
          path: 'organizations',
          name: 'organizations',
          component: () => import('@/views/organizations/OrganizationsView.vue')
        },
        {
          path: 'users',
          name: 'users',
          component: () => import('@/views/users/UsersView.vue')
        },
        {
          path: 'groups',
          name: 'groups',
          component: () => import('@/views/groups/GroupsView.vue')
        },
        {
          path: 'organizations/:orgId/employees',
          name: 'employees',
          component: () => import('@/views/employees/EmployeesView.vue'),
          props: true
        },
        {
          path: 'organizations/:orgId/children',
          name: 'children',
          component: () => import('@/views/children/ChildrenView.vue'),
          props: true
        }
      ]
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: '/'
    }
  ]
})

router.beforeEach((to, _from, next) => {
  const authStore = useAuthStore()
  const requiresAuth = to.matched.some((record) => record.meta.requiresAuth !== false)

  if (requiresAuth && !authStore.isAuthenticated) {
    next('/login')
  } else if (to.path === '/login' && authStore.isAuthenticated) {
    next('/')
  } else {
    next()
  }
})

export default router
