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
          path: 'probes',
          name: 'probes',
          component: () => import('@/views/ProbesView.vue')
        },
        {
          path: 'probes/:workflowId/:probeId',
          name: 'probe-detail',
          component: () => import('@/views/ProbeDetailView.vue')
        },
        {
          path: 'schedules',
          name: 'schedules',
          component: () => import('@/views/SchedulesView.vue')
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
          path: 'error-recovery',
          name: 'error-recovery-settings',
          component: () => import('@/views/ErrorRecoverySettings.vue')
        },
        {
          path: 'proxies',
          name: 'proxies',
          component: () => import('@/views/ProxyManagementView.vue')
        },
        {
          path: 'incidents',
          name: 'incidents',
          component: () => import('@/views/IncidentsView.vue')
        },
        {
          path: 'incidents/:id',
          name: 'incident-detail',
          component: () => import('@/views/IncidentDetailView.vue')
        },
        {
          path: 'recovery-history',
          name: 'recovery-history',
          component: () => import('@/views/RecoveryHistoryView.vue')
        },
        {
          path: 'domain-strategies',
          name: 'domain-strategies',
          component: () => import('@/views/DomainStrategiesView.vue')
        },
        {
          path: 'universal-scraper',
          name: 'universal-scraper',
          component: () => import('@/views/UniversalScraperView.vue')
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
