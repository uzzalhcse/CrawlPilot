import { createRouter, createWebHistory } from 'vue-router'
import DashboardLayout from '@/layouts/DashboardLayout.vue'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      component: DashboardLayout,
      children: [
        {
          path: '',
          name: 'dashboard',
          component: () => import('@/views/DashboardView.vue')
        },
        {
          path: 'workflows',
          name: 'workflows',
          component: () => import('@/views/WorkflowsView.vue')
        },
        {
          path: 'workflows/create',
          name: 'workflow-create',
          component: () => import('@/views/WorkflowCreateView.vue')
        },
        {
          path: 'workflows/:id',
          name: 'workflow-detail',
          component: () => import('@/views/WorkflowDetailView.vue')
        },
        {
          path: 'executions',
          name: 'executions',
          component: () => import('@/views/ExecutionsView.vue')
        },
        {
          path: 'executions/:id',
          name: 'execution-detail',
          component: () => import('@/views/ExecutionDetailView.vue')
        },
        {
          path: 'monitoring',
          name: 'monitoring-overview',
          component: () => import('@/views/HealthChecksOverview.vue')
        },
        {
          path: 'monitoring/:id',
          name: 'monitoring-detail',
          component: () => import('@/views/HealthCheckView.vue')
        },
        {
          path: 'analytics',
          name: 'analytics',
          component: () => import('@/views/AnalyticsView.vue')
        },
        {
          path: 'plugins',
          name: 'plugins',
          component: () => import('@/views/PluginMarketplace.vue')
        },
        {
          path: 'plugins/:id',
          name: 'plugin-detail',
          component: () => import('@/views/PluginDetailView.vue')
        },
        {
          path: 'browser-profiles',
          name: 'browser-profiles',
          component: () => import('@/views/BrowserProfilesView.vue')
        },
        {
          path: 'browser-profiles/create',
          name: 'browser-profile-create',
          component: () => import('@/views/BrowserProfileCreateView.vue')
        },
        {
          path: 'browser-profiles/:id/edit',
          name: 'browser-profile-edit',
          component: () => import('@/views/BrowserProfileCreateView.vue')
        },
        {
          path: 'browser-profiles/:id',
          name: 'browser-profile-detail',
          component: () => import('@/views/BrowserProfileDetailView.vue')
        },
        {
          path: '/:pathMatch(.*)*',
          name: 'not-found',
          component: () => import('@/views/NotFoundView.vue')
        }
      ]
    }
  ]
})

export default router
