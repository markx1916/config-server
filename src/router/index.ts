import { createRouter, createWebHashHistory, RouteRecordRaw } from 'vue-router';
import NacosAccess from '../views/NacosAccess.vue';
import ConfigList from '../views/ConfigList.vue';
import ConfigDetailView from '../views/ConfigDetailView.vue'; // Renamed from ConfigEditor

const routes: Array<RouteRecordRaw> = [
  {
    path: '/',
    redirect: '/nacos-instances', // Default to Nacos Instance Management
  },
  {
    path: '/nacos-instances',
    name: 'nacos-instances', // Renamed from 'nacos-access' for clarity
    component: NacosAccess,
  },
  {
    path: '/configurations',
    name: 'config-list',
    component: ConfigList,
  },
  {
    // Route for creating a new configuration.
    // Query param `nacosInstanceId` is expected to be passed via router.push
    path: '/configurations/new',
    name: 'config-new',
    component: ConfigDetailView,
    props: (route) => ({ // Pass route params and query as props
        id: 'new', // Indicate 'new' mode to the component
        nacosInstanceId: route.query.nacosInstanceId ? Number(route.query.nacosInstanceId) : null,
    }),
  },
  {
    // Route for viewing/editing an existing configuration.
    path: '/configurations/:id',
    name: 'config-detail', // This covers view/edit/history tabs
    component: ConfigDetailView,
    props: (route) => ({
      id: route.params.id, // Pass ID as prop
      // No need for nacosInstanceId here as the component will fetch the config which includes it
    }),
  },
  // Fallback route for 404
  {
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    component: () => import('../components/NotFound.vue'), // Assuming a NotFound component
  }
];

const router = createRouter({
  history: createWebHashHistory(), // Using hash mode for simplicity
  routes,
});

// Navigation guard example (optional, can be added later if needed)
// router.beforeEach((to, from, next) => {
//   // Check for authentication, etc.
//   // const isAuthenticated = !!localStorage.getItem('token');
//   // if (to.name !== 'Login' && !isAuthenticated) next({ name: 'Login' })
//   // else next()
//   next();
// });

export default router;
