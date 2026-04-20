import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  { path: '/', component: () => import('../views/CalendarView.vue') },
  { path: '/assignments/:id', component: () => import('../views/AssignmentView.vue') },
  { path: '/history', component: () => import('../views/HistoryView.vue') },
  { path: '/providers', component: () => import('../views/ProvidersView.vue') },
  { path: '/settings', component: () => import('../views/SettingsView.vue') },
]

export default createRouter({
  history: createWebHistory(),
  routes,
})
