<template>
  <el-container style="height: 100vh;">
    <el-aside width="200px" style="background-color: #545c64;">
      <el-menu
        :default-active="activeIndex"
        class="el-menu-vertical-demo"
        @select="handleSelect"
        background-color="#545c64"
        text-color="#fff"
        active-text-color="#ffd04b">
        <el-menu-item index="nacos-instances">
          <el-icon><SetUp /></el-icon> <!-- Changed icon -->
          <span>Nacos Instances</span>
        </el-menu-item>
        <el-menu-item index="config-list">
          <el-icon><Document /></el-icon>
          <span>Configurations</span> <!-- Changed label -->
        </el-menu-item>
      </el-menu>
    </el-aside>
    <el-main>
      <router-view></router-view>
    </el-main>
  </el-container>
</template>

<script lang="ts" setup>
import { ref } from 'vue'
import { SetUp, Document } from '@element-plus/icons-vue' // Removed Time, Added SetUp
import { useRouter, useRoute } from 'vue-router'

const router = useRouter()
const route = useRoute()

// activeIndex will now be computed based on the current route's name
const activeIndex = computed(() => {
  // Handle cases where route.name might be null or part of a nested route
  if (route.name === 'config-new' || route.name === 'config-detail') {
    return 'config-list'; // Highlight 'Configurations' when viewing/editing a config
  }
  return route.name || 'nacos-instances';
});
const handleSelect = (key: string) => {
  // activeIndex is now computed, so we just navigate
  router.push({ name: key })
}

// No need to manually push on mount, router handles initial navigation based on URL or redirect
// If you want a specific default page if no route matches, ensure your router's redirect handles it.
// The current redirect is from '/' to '/nacos-instances'.
</script>

<style>
body {
  margin: 0;
  font-family: 'Helvetica Neue', Helvetica, 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', '\\5FAE\\8F6F\\96C5\\9ED1', Arial, sans-serif;
}
.el-menu-vertical-demo:not(.el-menu--collapse) {
  width: 200px;
  min-height: 400px;
}
</style>
