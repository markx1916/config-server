<template>
  <div class="config-save-history-tab">
    <el-alert
      v-if="!configId"
      title="No Configuration Selected"
      type="warning"
      description="Cannot load save history without a configuration ID."
      show-icon
      :closable="false"
      style="margin-bottom: 20px;"
    ></el-alert>

    <div v-if="configId">
        <el-button @click="fetchHistory" :loading="configStore.loading" type="primary" plain size="small" style="margin-bottom: 15px;">
            <el-icon><Refresh /></el-icon> Refresh History
        </el-button>

        <el-timeline v-if="configStore.currentConfigSaveHistory.length > 0">
            <el-timeline-item
            v-for="item in configStore.currentConfigSaveHistory"
            :key="item.id"
            :timestamp="new Date(item.saved_at).toLocaleString()"
            placement="top"
            :type="item.version === configStore.currentConfiguration?.version ? 'primary' : 'info'"
            >
            <el-card>
                <h4>Version {{ item.version }} <el-tag size="small" :type="item.version === configStore.currentConfiguration?.version ? 'success' : 'info'">{{ item.version === configStore.currentConfiguration?.version ? 'Current' : 'Historical' }}</el-tag></h4>
                <p>Saved by: {{ item.saved_by || 'N/A' }}</p>
                <p>Format: {{ item.format }}</p>
                <el-button size="small" @click="viewContent(item)">View Content</el-button>
                <el-button
                    size="small"
                    type="warning"
                    @click="handleRollback(item.id)"
                    :disabled="item.version === configStore.currentConfiguration?.version"
                    :loading="configStore.loading"
                >
                    Rollback to this Version
                </el-button>
            </el-card>
            </el-timeline-item>
        </el-timeline>
        <el-empty v-else-if="!configStore.loading" description="No save history found for this configuration."></el-empty>
    </div>

    <el-dialog v-model="contentDialogVisible" title="View Historical Content" width="60%">
      <pre class="historical-content-viewer">{{ selectedHistoryContent }}</pre>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="contentDialogVisible = false">Close</el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted, watch, defineProps, toRefs } from 'vue';
import { useConfigStore } from '../store/configStore';
import type { ConfigurationHistory } from '../services/api';
import { ElMessage, ElMessageBox } from 'element-plus';
import { Refresh } from '@element-plus/icons-vue';

const props = defineProps<{
  configId: number | null;
}>();

const { configId } = toRefs(props);
const configStore = useConfigStore();

const contentDialogVisible = ref(false);
const selectedHistoryContent = ref('');

const fetchHistory = () => {
  if (configId.value) {
    configStore.fetchSaveHistory(configId.value);
  }
};

onMounted(() => {
  fetchHistory();
});

watch(configId, (newId) => {
  if (newId) {
    fetchHistory();
  } else {
    configStore.currentConfigSaveHistory = []; // Clear history if no configId
  }
});

const viewContent = (historyItem: ConfigurationHistory) => {
  selectedHistoryContent.value = historyItem.content;
  contentDialogVisible.value = true;
};

const handleRollback = async (historyEntryId: number) => {
  if (!configId.value) return;

  ElMessageBox.confirm(
    'Are you sure you want to rollback to this version? This will create a new version with the content of the selected historical version. It will not automatically publish to Nacos.',
    'Confirm Rollback',
    {
      confirmButtonText: 'Rollback',
      cancelButtonText: 'Cancel',
      type: 'warning',
    }
  ).then(async () => {
    try {
      await configStore.rollbackConfiguration(configId.value!, historyEntryId);
      // The store action already shows a success message and updates currentConfiguration.
      // Optionally, emit an event to parent if parent needs to react, e.g., refresh editor content.
    } catch (error) {
      // Error already handled by store/API service
    }
  }).catch(() => {
    ElMessage.info('Rollback canceled.');
  });
};
</script>

<style scoped>
.config-save-history-tab {
  padding: 10px;
}
.historical-content-viewer {
  background-color: #f5f5f5;
  padding: 15px;
  border-radius: 4px;
  max-height: 60vh;
  overflow-y: auto;
  white-space: pre-wrap; /* Ensures line breaks and spaces are preserved */
  word-break: break-all; /* Ensures long strings without spaces wrap */
}
</style>
