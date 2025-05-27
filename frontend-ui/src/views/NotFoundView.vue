<template>
  <div class="not-found-view container">
    <el-result
      icon="error"
      title="404 - Page Not Found"
      sub-title="Sorry, the page you are looking for does not exist."
    >
      <template #extra>
        <el-button type="primary" @click="goHome">Go Home</el-button>
      </template>
    </el-result>
  </div>
</template>

<script lang="ts" setup>
import { useRouter } from 'vue-router';
import { ElResult, ElButton } from 'element-plus';

const router = useRouter();

const goHome = () => {
  router.push('/');
};
</script>

<style lang="scss" scoped>
.not-found-view {
  display: flex;
  justify-content: center;
  align-items: center;
  height: calc(100vh - 120px); // Adjust based on your layout's header/footer height
}
</style>
