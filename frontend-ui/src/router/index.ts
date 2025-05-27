import { createRouter, createWebHistory, RouteRecordRaw } from 'vue-router';

// Import view components
import NacosManagementView from '../views/NacosManagementView.vue';
import ConfigurationsView from '../views/ConfigurationsView.vue';
import ConfigurationEditorView from '../views/ConfigurationEditorView.vue';
import ConfigurationHistoryView from '../views/ConfigurationHistoryView.vue';
import PublishRecordsView from '../views/PublishRecordsView.vue'; // Added
import NotFoundView from '../views/NotFoundView.vue';
// import ReleaseLogView from '../views/ReleaseLogView.vue'; // Optional

const routes: Array<RouteRecordRaw> = [
  {
    path: '/',
    redirect: '/nacos-management', // Default route
  },
  {
    path: '/nacos-management',
    name: 'NacosManagement',
    component: NacosManagementView,
    meta: { title: 'Nacos Instances' },
  },
  {
    path: '/configurations',
    name: 'Configurations',
    component: ConfigurationsView,
    meta: { title: 'Configurations' },
  },
  {
    path: '/configurations/edit/:id',
    name: 'ConfigurationEdit',
    component: ConfigurationEditorView,
    props: route => ({ configId: route.params.id, mode: 'edit' }),
    meta: { title: 'Edit Configuration' },
  },
  {
    path: '/configurations/create',
    name: 'ConfigurationCreate',
    component: ConfigurationEditorView,
    props: { mode: 'create' }, // Pass mode prop
    meta: { title: 'Create Configuration' },
  },
  {
    path: '/configurations/history/:id',
    name: 'ConfigurationHistory',
    component: ConfigurationHistoryView,
    props: true, // Passes route.params as props to the component (e.g., id)
    meta: { title: 'Configuration History' },
  },
  { // Added route for Publish Records
    path: '/publish-records',
    name: 'PublishRecords',
    component: PublishRecordsView,
    meta: { title: 'Publish Records' },
  },
  // { // Optional Release Log View
  //   path: '/release-log',
  //   name: 'ReleaseLog',
  //   component: ReleaseLogView, // Uncomment if ReleaseLogView is created
  //   meta: { title: 'Release Log' },
  // },
  // Catch-all for 404 Not Found - must be the last route
  {
    path: '/:catchAll(.*)*', // Matches any path not matched above
    name: 'NotFound',
    component: NotFoundView,
    meta: { title: 'Page Not Found' },
  }
];

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL), // Vite uses import.meta.env.BASE_URL
  routes,
});

// Global navigation guard (example: for setting document title)
router.beforeEach((to, from, next) => {
  document.title = to.meta.title ? `${to.meta.title} - Nacos Config Tool` : 'Nacos Config Tool';
  next();
});

export default router;
