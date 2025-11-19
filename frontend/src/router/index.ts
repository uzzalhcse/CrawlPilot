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
          path: 'analytics',
          name: 'analytics',
          component: () => import('@/views/AnalyticsView.vue')
        }
      ]
    }
  ]
})

export default router
