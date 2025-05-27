<template>
  <el-container class="main-layout">
    <el-aside width="220px" class="sidebar">
      <el-menu
        :default-active="activeRoute"
        class="el-menu-vertical-demo"
        router
        :collapse="isCollapsed"
      >
        <div class="sidebar-header">
          <img src="/logo.svg" alt="Nacos Logo" class="logo" v-if="!isCollapsed" />
          <h1 v.if="!isCollapsed">Nacos Tool</h1>
        </div>

        <el-menu-item index="/nacos-management">
          <el-icon><Setting /></el-icon>
          <span>Nacos Instances</span>
        </el-menu-item>
        <el-menu-item index="/configurations">
          <el-icon><Files /></el-icon>
          <span>Configurations</span>
        </el-menu-item>
        <el-menu-item index="/publish-records" route="/api/publish-records"> <!-- Corrected: route prop is not standard, use index -->
           <el-icon><Memo /></el-icon>
           <span>Publish Records</span>
        </el-menu-item>
        <!-- <el-menu-item index="/release-log">
          <el-icon><Promotion /></el-icon>
          <span>Release Log</span>
        </el-menu-item> -->
         <div class="collapse-button-container">
          <el-button @click="toggleCollapse" type="text" class="collapse-button">
            <el-icon>
              <Expand v-if="isCollapsed" />
              <Fold v-else />
            </el-icon>
          </el_button>
        </div>
      </el-menu>
    </el-aside>

    <el-container>
      <el-header class="app-header">
        <div>{{ currentRouteTitle }}</div>
        <!-- Add any header content here, like user profile, notifications -->
      </el-header>
      <el_main class="content-area">
        <router-view v-slot="{ Component }">
          <template v-if="Component">
            <suspense>
              <component :is="Component"></component>
              <template #fallback>
                <div>Loading...</div>
              </template>
            </suspense>
          </template>
        </router-view>
      </el_main>
    </el-container>
  </el-container>
</template>

<script lang="ts" setup>
import { ref, computed, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { Setting, Files, Memo, Promotion, Expand, Fold } from '@element-plus/icons-vue'; // Import icons

const route = useRoute();
const router = useRouter(); // Get router instance

const isCollapsed = ref(false);

const activeRoute = computed(() => {
  return route.path;
});

const currentRouteTitle = computed(() => {
  return route.meta.title || 'Nacos Configuration Tool';
});

const toggleCollapse = () => {
  isCollapsed.value = !isCollapsed.value;
};

// Ensure logo.svg is present in the public directory
// Create a placeholder if it does not exist
watch(isCollapsed, (newVal) => {
  // Adjust sidebar width if needed, though Element Plus handles it with collapse attribute
});

</script>

<style lang="scss">
.main-layout {
  height: 100vh;
  display: flex;
}

.sidebar {
  background-color: #f8f9fa; // Lighter sidebar color
  border-right: 1px solid #e0e0e0;
  transition: width 0.3s ease;
  display: flex;
  flex-direction: column; // Ensure items stack vertically

  .el-menu {
    border-right: none; // Remove default border as sidebar has its own
    height: 100%; // Make menu take full height of sidebar
    display: flex;
    flex-direction: column; // Stack menu items and collapse button
  }
  .el-menu-item {
    &:hover {
      background-color: #e9ecef;
    }
    &.is-active {
      background-color: var(--el-color-primary-light-9) !important; // Element Plus primary color for active item
      color: var(--el-color-primary);
      border-right: 3px solid var(--el-color-primary);
    }
  }
}

.sidebar-header {
  display: flex;
  align-items: center;
  padding: 15px 20px; // Consistent padding
  margin-bottom: 10px; // Space before first menu item
  // border-bottom: 1px solid #dee2e6; // Subtle separator

  .logo {
    height: 32px; // Adjust as needed
    width: 32px;
    margin-right: 12px; // Space between logo and title
    object-fit: contain;
  }

  h1 {
    font-size: 1.25rem; // Slightly larger title
    font-weight: 600;
    color: #343a40; // Darker title color
    margin: 0;
    white-space: nowrap; // Prevent title from wrapping
    overflow: hidden;
    text-overflow: ellipsis;
  }
}


.app-header {
  background-color: #ffffff;
  padding: 0 20px;
  display: flex;
  align-items: center;
  justify-content: space-between; // For title on left, other items (e.g. user menu) on right
  border-bottom: 1px solid #dcdfe6; // Element Plus default border color
  height: 60px; // Standard header height
  font-size: 1.1rem;
  font-weight: 500;
}

.content-area {
  padding: 20px;
  background-color: #f0f2f5; // Light background for content area
  // overflow-y: auto; // Allow scrolling for content area if it overflows
  // height: calc(100vh - 60px); // Full height minus header
}

// Ensure router-view and its content take up available space
// #app > .el-container > .el-container { // Target the main content container
//   flex-direction: column;
//   .el-main {
//     flex-grow: 1;
//     display: flex;
//     flex-direction: column;
//     > * { // Make the direct child of el-main (router-view's content) also grow
//       flex-grow: 1;
//     }
//   }
// }

.collapse-button-container {
  margin-top: auto; // Pushes the button to the bottom
  padding: 10px 0; // Some padding around the button
  // border-top: 1px solid #e0e0e0; // Optional separator
  display: flex;
  justify-content: center;
}
.collapse-button .el-icon {
  font-size: 20px; // Make icon larger
  color: #606266;
}

/* Fade transition for router view */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
