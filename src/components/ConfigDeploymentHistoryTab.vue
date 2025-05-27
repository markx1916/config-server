<template>
  <div class="config-deployment-history-tab">
    <el-alert
      v-if="!configIdProp"
      title="No Configuration Selected"
      type="warning"
      description="Cannot load deployment history without a configuration ID."
      show-icon
      :closable="false"
      style="margin-bottom: 20px;"
    ></el-alert>

    <div v-if="configIdProp">
      <el-button @click="fetchDeploymentHistory" :loading="configStore.loading" type="primary" plain size="small" style="margin-bottom: 15px;">
          <el-icon><Refresh /></el-icon> Refresh History
      </el-button>

      <el-table :data="configStore.currentConfigDeploymentHistory" v-loading="configStore.loading" border style="width: 100%">
        <el-table-column prop="id" label="ID" width="80" sortable></el-table-column>
        <el-table-column prop="configuration_version" label="Version Deployed" width="150" align="center" sortable>
            <template #default="scope">
                <el-tag size="small">v{{ scope.row.configuration_version }}</el-tag>
            </template>
        </el-table-column>
        <el-table-column prop="deployment_type" label="Type" width="100" align="center">
            <template #default="scope">
                <el-tag :type="scope.row.deployment_type === 'FULL' ? 'success' : 'warning'" size="small">
                    {{ scope.row.deployment_type }}
                </el-tag>
            </template>
        </el-table-column>
        <el-table-column prop="status" label="Status" width="120" align="center" sortable>
             <template #default="scope">
                <el-tag
                    :type="scope.row.status === 'SUCCESS' ? 'success' : (scope.row.status === 'FAILURE' ? 'danger' : 'info')"
                    size="small"
                >
                    {{ scope.row.status }}
                </el-tag>
            </template>
        </el-table-column>
        <el-table-column prop="deployed_by" label="Deployed By" width="150"></el-table-column>
        <el-table-column prop="deployed_at" label="Deployed At" width="180" sortable>
          <template #default="scope">{{ new Date(scope.row.deployed_at).toLocaleString() }}</template>
        </el-table-column>
        <el-table-column prop="message" label="Message" min-width="200">
            <template #default="scope">
                <el-popover
                    placement="top-start"
                    title="Deployment Message"
                    width="400"
                    trigger="hover"
                    :content="scope.row.message || 'No message.'"
                    v-if="scope.row.message"
                >
                    <template #reference>
                        <span class="ellipsis">{{ scope.row.message || '-' }}</span>
                    </template>
                </el-popover>
                 <span v-else>-</span>
            </template>
        </el-table-column>
      </el-table>
      <el-empty v-if="!configStore.loading && configStore.currentConfigDeploymentHistory.length === 0" description="No deployment history found."></el-empty>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { defineProps, toRefs, onMounted, watch } from 'vue';
import { useConfigStore } from '../store/configStore';
import { Refresh } from '@element-plus/icons-vue';
// No longer need useRoute or useRouter if it's purely a display component driven by props

const props = defineProps<{
  configIdProp: number | null; // Renamed to avoid conflict with potential 'configId' from route if this were a view
}>();

const { configIdProp } = toRefs(props);
const configStore = useConfigStore();

const fetchDeploymentHistory = () => {
  if (configIdProp.value) {
    configStore.fetchDeploymentHistory(configIdProp.value);
  }
};

onMounted(() => {
  // Initial fetch when component is mounted with a valid configIdProp
  fetchDeploymentHistory();
});

// Watch for changes in configIdProp if the parent component might change it
watch(configIdProp, (newId) => {
  if (newId) {
    fetchDeploymentHistory();
  } else {
    configStore.currentConfigDeploymentHistory = []; // Clear history if prop becomes null
  }
});

</script>

<style scoped>
.config-deployment-history-tab {
  padding: 10px;
}
.ellipsis {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  display: block;
}
</style>
