import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'

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
          props: true,
          meta: { orgScoped: true }
        },
        {
          path: 'organizations/:orgId/children',
          name: 'children',
          component: () => import('@/views/children/ChildrenView.vue'),
          props: true,
          meta: { orgScoped: true }
        }
      ]
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: '/'
    }
  ]
})

router.beforeEach(async (to, _from, next) => {
  const authStore = useAuthStore()
  const requiresAuth = to.matched.some((record) => record.meta.requiresAuth !== false)

  // Handle authentication
  if (requiresAuth && !authStore.isAuthenticated) {
    next('/login')
    return
  } else if (to.path === '/login' && authStore.isAuthenticated) {
    next('/')
    return
  }

  // Handle org-scoped routes: sync URL org to store
  const isOrgScoped = to.matched.some((record) => record.meta.orgScoped)
  if (isOrgScoped && to.params.orgId) {
    const uiStore = useUiStore()
    const orgId = Number(to.params.orgId)

    // Ensure organizations are loaded before validation
    if (uiStore.organizations.length === 0) {
      await uiStore.fetchOrganizations()
    }

    // Validate org exists
    if (!uiStore.isValidOrganization(orgId)) {
      // Invalid org ID - redirect to dashboard
      next('/')
      return
    }

    // Sync the org selection from URL to store
    uiStore.syncFromRoute(orgId)
  }

  next()
})

export default router
