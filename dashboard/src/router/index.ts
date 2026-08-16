import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { APP_TITLE } from '@/lib/app'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/auth/LoginView.vue'),
      meta: { public: true, title: 'Đăng nhập' },
    },
    {
      path: '/',
      component: () => import('@/layouts/AdminLayout.vue'),
      children: [
        {
          path: '',
          name: 'dashboard',
          component: () => import('@/views/dashboard/DashboardView.vue'),
          meta: { perm: 'dashboard.view', title: 'Tổng quan' },
        },
        {
          path: 'users',
          name: 'users',
          component: () => import('@/views/users/UsersView.vue'),
          meta: { perm: 'users.read', title: 'Người dùng' },
        },
        {
          path: 'roles',
          name: 'roles',
          component: () => import('@/views/roles/RolesView.vue'),
          meta: { perm: 'roles.read', title: 'Vai trò & Quyền' },
        },
        {
          path: 'api-keys',
          name: 'api-keys',
          component: () => import('@/views/api-keys/ApiKeysView.vue'),
          meta: { perm: 'api_keys.read', title: 'API Keys' },
        },
        {
          path: 'config',
          name: 'config',
          component: () => import('@/views/config/ConfigView.vue'),
          meta: { perm: 'config.read', title: 'Cấu hình động' },
        },
        {
          path: 'cache',
          name: 'cache',
          component: () => import('@/views/cache/CacheView.vue'),
          meta: { perm: 'cache.read', title: 'Cache' },
        },
        {
          path: 'settings',
          name: 'settings',
          component: () => import('@/views/profile/ProfileView.vue'),
          meta: { perm: 'settings.read', title: 'Cài đặt' },
        },
      ],
    },
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
})

// ---- Guard: auth + permission ----
router.beforeEach(async (to) => {
  const auth = useAuthStore()

  if (to.meta.public) return true

  if (!auth.isAuthenticated) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }

  // Có token nhưng chưa có user → fetchMe (fail thì logout)
  if (!auth.user) {
    try {
      await auth.fetchMe()
    } catch {
      auth.logout()
      return { name: 'login' }
    }
  }

  // Permission check — user không có quyền → dashboard (hoặc login)
  const perm = to.meta.perm as string | undefined
  if (perm) {
    const perms = new Set<string>()
    // skeleton: claims perms qua /auth/me trả user — role check đơn giản ở đây
    // (mở rộng: fetch permissions qua endpoint khi có)
    if (!perms.has(perm)) {
      // Tạm: cho phép mọi user đã đăng nhập (RBAC frontend hoàn thiện ở phase sau)
    }
  }

  return true
})

router.afterEach((to) => {
  document.title = to.meta.title ? `${to.meta.title} · ${APP_TITLE}` : APP_TITLE
})

export default router
