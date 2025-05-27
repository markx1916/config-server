<template>
  <div class="configuration-history-view container">
    <el-page-header @back="goBack" :content="pageTitle" class="mb-20">
    </el-page-header>

    <el-card v-if="!isLoadingConfig && config">
        <el-descriptions :title="`Details for: ${config.DataID} (${config.Group})`" :column="2" border class="mb-20">
            <el-descriptions-item label="Data ID">{{ config.DataID }}</el-descriptions-item>
            <el-descriptions-item label="Group">{{ config.Group }}</el-descriptions-item>
            <el-descriptions-item label="Namespace ID">{{ config.NamespaceID || 'public' }}</el-descriptions-item>
            <el-descriptions-item label="Nacos Instance">
                {{ config.NacosInstance?.Description || config.NacosInstance?.InstanceURL || 'N/A' }}
            </el-descriptions-item>
        </el-descriptions>
    </el-card>
     <el-skeleton :rows="2" animated v-else-if="isLoadingConfig" />


    <el-card>
      <template #header>
        <span>Version History</span>
      </template>
      <el-empty v-if="isLoadingHistory" description="Loading history..."></el-empty>
      <el-empty v-else-if="historyEntries.length === 0 && !isLoadingHistory" description="No history found for this configuration."></el-empty>
      <el-table :data="historyEntries" stripe style="width: 100%" v-else>
        <el-table-column prop="ID" label="History ID" width="100" sortable />
        <el-table-column prop="CreatedAt" label="Saved At" sortable>
          <template #default="scope">
            {{ new Date(scope.row.CreatedAt).toLocaleString() }}
          </template>
        </el-table-column>
        <el-table-column prop="Operator" label="Operator" width="150" />
        <el-table-column prop="ChangeSource" label="Change Source" width="200" />
        <el-table-column prop="MD5" label="MD5 Checksum" width="280" />
        <el-table-column label="Actions" width="280" fixed="right">
          <template #default="scope">
            <el-button size="small" @click="viewContent(scope.row)" icon="View">View Content</el-button>
            <el-button size="small" type="warning" @click="handleRollback(scope.row)" icon="RefreshLeft">Rollback to this</el-button>
            <!-- <el-button size="small" type="info" @click="diffWithCurrent(scope.row)">Diff with Current</el-button> -->
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- Dialog to View Historical Content -->
    <el-dialog v-model="contentDialogVisible" title="Historical Content" width="70%" top="5vh">
      <div v-if="selectedHistoryEntry">
        <p><strong>History ID:</strong> {{ selectedHistoryEntry.ID }}</p>
        <p><strong>Saved At:</strong> {{ new Date(selectedHistoryEntry.CreatedAt).toLocaleString() }}</p>
        <p><strong>Operator:</strong> {{ selectedHistoryEntry.Operator }}</p>
        <p><strong>Content Type:</strong> {{ selectedHistoryEntry.ContentType }}</p>
        <div class="editor-container-readonly mt-20">
             <codemirror
              v-model="selectedHistoryEntry.Content"
              :style="{ height: '400px', width: '100%' }"
              :extensions="codemirrorExtensionsDialog"
              :disabled="true" 
            />
        </div>
      </div>
      <template #footer>
        <el-button @click="contentDialogVisible = false">Close</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed, onMounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { ElCard, ElButton, ElIcon, ElEmpty, ElTable, ElTableColumn, ElPageHeader, ElDialog, ElMessage, ElDescriptions, ElDescriptionsItem, ElSkeleton } from 'element-plus';
import { View, RefreshLeft } from '@element-plus/icons-vue';
import { Codemirror } from 'vue-codemirror';
import { javascript } from '@codemirror/lang-javascript';
import { yaml } from '@codemirror/lang-yaml';
import { json } from '@codemirror/lang-json';
import { html } from '@codemirror/lang-html';
import { oneDark } from '@codemirror/theme-one-dark'; // Example theme


import * as api from '@/services/api';
import type { Configuration, ConfigurationHistory } from '@/types';

const props = defineProps<{
  id: string; // Configuration ID from route params
}>();

const route = useRoute();
const router = useRouter();

const config = ref<Configuration | null>(null);
const historyEntries = ref<ConfigurationHistory[]>([]);
const isLoadingConfig = ref(false);
const isLoadingHistory = ref(false);

const contentDialogVisible = ref(false);
const selectedHistoryEntry = ref<ConfigurationHistory | null>(null);

const pageTitle = computed(() => `History for Configuration ID: ${props.id}`);


const codemirrorExtensionsDialog = computed(() => {
  if (!selectedHistoryEntry.value) return [oneDark];
  const langExtMap: { [key: string]: () => any } = {
    json: json,
    yaml: yaml,
    javascript: javascript,
    html: html,
    text: () => [],
  };
  const selectedLangExtension = langExtMap[selectedHistoryEntry.value.ContentType || 'text'] || (() => []);
  return [selectedLangExtension(), oneDark];
});


const fetchConfigurationDetails = async () => {
  isLoadingConfig.value = true;
  try {
    const response = await api.getConfigurationById(Number(props.id));
    config.value = response.data;
  } catch (error) {
    console.error(`Failed to fetch configuration details for ID ${props.id}:`, error);
    ElMessage.error('Failed to load configuration details.');
  } finally {
    isLoadingConfig.value = false;
  }
};

const fetchHistory = async () => {
  isLoadingHistory.value = true;
  try {
    const response = await api.getConfigurationHistory(Number(props.id));
    historyEntries.value = response.data;
  } catch (error) {
    console.error(`Failed to fetch history for config ID ${props.id}:`, error);
    ElMessage.error('Failed to load configuration history.');
    historyEntries.value = [];
  } finally {
    isLoadingHistory.value = false;
  }
};

const viewContent = (entry: ConfigurationHistory) => {
  selectedHistoryEntry.value = entry;
  contentDialogVisible.value = true;
};

const handleRollback = async (historyEntry: ConfigurationHistory) => {
  try {
    // Ask for confirmation before rollback
    await ElMessage.confirm(
      `Are you sure you want to rollback to version from ${new Date(historyEntry.CreatedAt).toLocaleString()}? This will update the current configuration content. You will still need to publish the changes.`,
      'Confirm Rollback',
      {
        confirmButtonText: 'Rollback',
        cancelButtonText: 'Cancel',
        type: 'warning',
      }
    );

    // Proceed with rollback
    await api.rollbackConfiguration(Number(props.id), historyEntry.ID, { operator: 'frontend-user-rollback' });
    ElMessage.success('Configuration rolled back successfully! The content is now updated to the selected historical version. Please review and publish if needed.');
    // Optionally, redirect to editor or refresh current configuration data if displayed on this page.
    router.push({ name: 'ConfigurationEdit', params: { id: props.id } });

  } catch (error: any) {
    if (error !== 'cancel' && error.message !== 'cancel') { // Error from API or unhandled promise rejection
        console.error('Rollback failed:', error);
        // API error already shown by global interceptor
    } else {
        ElMessage.info('Rollback cancelled.');
    }
  }
};

const goBack = () => {
  // Go back to configurations list, potentially for the same instance/namespace if stored in Pinia
  router.push('/configurations');
};

onMounted(() => {
  fetchConfigurationDetails();
  fetchHistory();
});
</script>

<style lang="scss" scoped>
.configuration-history-view {
  // Styles for this view
}
.editor-container-readonly {
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  overflow: hidden;
}
.mt-20 {
    margin-top: 20px;
}
</style>
