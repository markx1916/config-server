<template>
  <div class="not-found-view">
    <el-result icon="error" title="404 - Page Not Found" sub-title="Sorry, the page you are looking for does not exist.">
      <template #extra>
        <el-button type="primary" @click="goHome">Go to Homepage</el-button>
      </template>
    </el-result>
  </div>
</template>

<script lang="ts" setup>
import { useRouter } from 'vue-router';

const router = useRouter();

const goHome = () => {
  router.push({ name: 'nacos-instances' }); // Or your primary landing page route name
};
</script>

<style scoped>
.not-found-view {
  display: flex;
  justify-content: center;
  align-items: center;
  height: calc(100vh - 200px); /* Adjust height as needed */
  padding: 20px;
}
</style>
