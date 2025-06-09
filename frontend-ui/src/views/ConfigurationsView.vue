<template>
  <div class="configurations-view container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>Configurations</span>
          <el-button type="primary" @click="handleCreateConfiguration" :disabled="!nacosStore.hasSelectedInstance" icon="Plus">
            Create Configuration
          </el-button>
        </div>
      </template>

      <el-row :gutter="20" class="mb-20 control-row">
        <el-col :span="8">
          <el-select
            v-model="selectedInstanceId"
            placeholder="Select Nacos Instance"
            class="full-width"
            @change="handleInstanceChange"
            filterable
            clearable
          >
            <el-option
              v-for="instance in nacosInstances"
              :key="instance.ID"
              :label="`${instance.Description || instance.InstanceURL} (${instance.InstanceURL})`"
              :value="instance.ID"
            />
          </el-select>
        </el-col>
        <el-col :span="16">
          <div v-if="nacosStore.hasSelectedInstance" class="namespace-tabs-container">
            <span class="namespace-label">Namespace:</span>
            <el-tabs v-model="activeNamespace" @tab-change="handleNamespaceChange" type="card" class="namespace-tabs">
              <el-tab-pane
                v-for="ns in nacosStore.availableNamespaces"
                :key="ns.namespaceId"
                :label="ns.namespaceShowName || ns.namespaceId || 'public'"
                :name="ns.namespaceId"
              />
              <!-- Add a specific tab for public if not already listed and if others exist -->
              <el-tab-pane
                label="public (Default)"
                name=""
                v-if="!nacosStore.availableNamespaces.some(ns => ns.namespaceId === '') && nacosStore.availableNamespaces.length > 0">
              </el-tab-pane>
               <el-tab-pane
                label="public"
                name=""
                v-if="nacosStore.availableNamespaces.length === 0"> <!-- Show public if no custom namespaces loaded -->
              </el-tab-pane>
              <el-tab-pane label="[All Namespaces]" name="_ALL_" v-if="nacosStore.availableNamespaces.length > 0"></el-tab-pane>
            </el-tabs>
          </div>
           <el-alert v-else title="Please select a Nacos instance to see namespaces and configurations." type="info" :closable="false" show-icon />
        </el-col>
      </el-row>

      <el-table :data="configurations" v-loading="isLoadingConfigurations" stripe style="width: 100%">
        <el-table-column prop="ID" label="ID" width="80" sortable />
        <el-table-column prop="DataID" label="Data ID" min-width="200" sortable show-overflow-tooltip/>
        <el-table-column prop="Group" label="Group" min-width="150" sortable show-overflow-tooltip/>
        <el-table-column prop="NamespaceID" label="Namespace ID" min-width="150" sortable show-overflow-tooltip>
            <template #default="scope">
                {{ scope.row.NamespaceID || 'public' }}
            </template>
        </el-table-column>
        <el-table-column prop="ContentType" label="Format" width="100" />
        <el-table-column prop="UpdatedAt" label="Last Modified (Local)" width="180" sortable>
          <template #default="scope">
            {{ new Date(scope.row.UpdatedAt).toLocaleString() }}
          </template>
        </el-table-column>
         <el-table-column prop="LastSyncTime" label="Last Synced (Nacos)" width="180" sortable>
          <template #default="scope">
            {{ scope.row.LastSyncTime ? new Date(scope.row.LastSyncTime).toLocaleString() : 'N/A' }}
          </template>
        </el-table-column>
        <el-table-column label="Actions" width="300" fixed="right">
          <template #default="scope">
            <el-button size="small" @click="handleEdit(scope.row.ID)" icon="Edit">Edit</el-button>
            <el-button size="small" type="info" @click="handleViewHistory(scope.row.ID)" icon="View">History</el-button>
            <el-button size="small" type="success" @click="openPublishDialog(scope.row)" icon="Promotion">Publish</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-if="!isLoadingConfigurations && configurations.length === 0"
        :description="nacosStore.hasSelectedInstance ? 'No configurations found for this instance/namespace.' : 'Please select a Nacos Instance.'">
      </el-empty>

    </el-card>

     <!-- Publish Dialog -->
    <el-dialog v-model="publishDialogVisible" title="Publish Configuration to Nacos" width="500px" @closed="resetPublishForm">
        <div v-if="configToPublish">
            <p><strong>Data ID:</strong> {{ configToPublish.DataID }}</p>
            <p><strong>Group:</strong> {{ configToPublish.Group }}</p>
            <p><strong>Target Namespace:</strong> {{ configToPublish.NamespaceID || 'public' }}</p>
            <el-form :model="publishForm" label-position="top" ref="publishFormRef" class="mt-20">
                <el-form-item label="Publish Type" prop="publishType" :rules="[{ required: true, message: 'Publish type is required'}]">
                    <el-radio-group v-model="publishForm.publishType">
                        <el-radio label="FULL">Full Publish</el-radio>
                        <el-radio label="GRAYSCALE" disabled>Grayscale (Not Implemented)</el-radio>
                    </el-radio-group>
                </el-form-item>
                <el-form-item label="Operator (Optional)" prop="operator">
                    <el-input v-model="publishForm.operator" placeholder="Your name or system ID" />
                </el-form-item>
            </el-form>
        </div>
        <template #footer>
            <el-button @click="publishDialogVisible = false">Cancel</el-button>
            <el-button type="primary" @click="confirmPublish" :loading="isPublishing">Confirm Publish</el-button>
        </template>
    </el-dialog>

  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted, watch, reactive } from 'vue';
import { useRouter } from 'vue-router';
import { ElCard, ElButton, ElIcon, ElEmpty, ElSelect, ElOption, ElTabs, ElTabPane, ElTable, ElTableColumn, ElRow, ElCol, ElAlert, ElMessage, ElDialog, ElForm, ElFormItem, ElRadioGroup, ElRadio, ElInput } from 'element-plus';
import { Plus, Edit, View, Promotion } from '@element-plus/icons-vue';
import { useNacosStore } from '@/store';
import * as api from '@/services/api';
import type { NacosInstance, Configuration as ConfigType, NacosNamespace, PublishPayload } from '@/types';

const router = useRouter();
const nacosStore = useNacosStore();

const nacosInstances = ref<NacosInstance[]>([]);
const selectedInstanceId = ref<number | null>(null); // Use local ref for el-select model
const activeNamespace = ref<string>(''); // Use local ref for el-tabs model
const configurations = ref<ConfigType[]>([]);
const isLoadingConfigurations = ref(false);

// Publish Dialog
const publishDialogVisible = ref(false);
const configToPublish = ref<ConfigType | null>(null);
const publishFormRef = ref<InstanceType<typeof ElForm>>();
const initialPublishFormState = {
    publishType: 'FULL' as 'FULL' | 'GRAYSCALE',
    operator: 'frontend-user',
};
const publishForm = reactive({ ...initialPublishFormState });
const isPublishing = ref(false);


const fetchNacosInstances = async () => {
  try {
    const response = await api.getNacosInstances();
    nacosInstances.value = response.data;
    // Restore selection from store if valid
    if (nacosStore.selectedInstanceId && nacosInstances.value.find(inst => inst.ID === nacosStore.selectedInstanceId)) {
      selectedInstanceId.value = nacosStore.selectedInstanceId;
      // If instance is restored, also try to restore namespaces and active namespace
      if (nacosStore.availableNamespaces.length > 0) {
         activeNamespace.value = nacosStore.currentNamespaceId || (nacosStore.availableNamespaces[0]?.namespaceId ?? '');
      }
      fetchConfigurations(); // Fetch configs for restored selection
    } else if (nacosInstances.value.length > 0) {
      // Default to first instance if no valid selection in store
      // selectedInstanceId.value = nacosInstances.value[0].ID;
      // handleInstanceChange(selectedInstanceId.value); // This will fetch namespaces and configs
    } else {
      // No instances, clear dependent state
      nacosStore.clearSelection();
      configurations.value = [];
    }
  } catch (error) {
    console.error('Failed to fetch Nacos instances:', error);
  }
};

const fetchNamespacesForInstance = async (instanceId: number) => {
  if (!instanceId) {
    nacosStore.setNamespaces([]);
    activeNamespace.value = ''; // Default to public for tabs
    configurations.value = [];
    return;
  }
  try {
    const response = await api.getNacosInstanceNamespaces(instanceId);
    let fetchedNamespaces = response.data.map((ns: any) => ({
        namespaceId: ns.namespace,
        namespaceShowName: ns.namespaceShowName || ns.namespace,
    }));

    // Ensure 'public' (empty string ID) is handled correctly.
    // If the API doesn't return an entry for public explicitly, but it's the default.
    // For now, assume API returns all relevant namespaces including one for public if it has configs.
    // The UI will provide a "public" tab if no specific public entry is returned but others exist.

    nacosStore.setNamespaces(fetchedNamespaces);
    // If currentNamespaceId from store is valid within new namespaces, use it. Otherwise, default.
    if (fetchedNamespaces.some(ns => ns.namespaceId === nacosStore.currentNamespaceId)) {
        activeNamespace.value = nacosStore.currentNamespaceId!;
    } else if (fetchedNamespaces.length > 0) {
        activeNamespace.value = fetchedNamespaces[0].namespaceId; // Default to first
    } else {
        activeNamespace.value = ''; // Default to public if no custom namespaces
    }
    nacosStore.setCurrentNamespace(activeNamespace.value); // Update store
    fetchConfigurations();
  } catch (error) {
    console.error(`Failed to fetch namespaces for instance ${instanceId}:`, error);
    nacosStore.setNamespaces([]);
    activeNamespace.value = '';
    configurations.value = [];
  }
};

const fetchConfigurations = async () => {
  if (!selectedInstanceId.value) {
    configurations.value = [];
    return;
  }
  isLoadingConfigurations.value = true;
  try {
    const params: api.ListConfigurationsParams = {
      nacosInstanceId: selectedInstanceId.value,
    };
    // Only add namespaceId to params if it's not the "All Namespaces" placeholder
    if (activeNamespace.value !== '_ALL_') {
        params.namespaceId = activeNamespace.value;
    }

    const response = await api.getConfigurations(params);
    configurations.value = response.data;
  } catch (error) {
    console.error('Failed to fetch configurations:', error);
    configurations.value = [];
  } finally {
    isLoadingConfigurations.value = false;
  }
};

const handleInstanceChange = (instanceId: number | null) => {
  selectedInstanceId.value = instanceId; // Update local ref for el-select
  if (instanceId) {
    const selected = nacosInstances.value.find(inst => inst.ID === instanceId);
    if (selected) {
      nacosStore.selectInstance({ id: selected.ID, instanceUrl: selected.InstanceURL });
      fetchNamespacesForInstance(selected.ID); // This will also trigger fetchConfigurations
    }
  } else {
    nacosStore.clearSelection();
    configurations.value = [];
    activeNamespace.value = ''; // Reset tabs
  }
};

const handleNamespaceChange = (namespaceId: string | number | undefined) => { // name can be string or number or undefined from el-tabs
  const nsId = (namespaceId === undefined || namespaceId === null) ? '' : String(namespaceId);
  activeNamespace.value = nsId; // Update local ref for el-tabs
  nacosStore.setCurrentNamespace(nsId);
  fetchConfigurations();
};

const handleCreateConfiguration = () => {
  if (nacosStore.selectedInstanceId) {
    let targetNamespace = activeNamespace.value;
    if (targetNamespace === '_ALL_') { // If "All Namespaces" is selected, default new config to public
        targetNamespace = '';
    }
    router.push({
        name: 'ConfigurationCreate',
        query: {
            nacosInstanceId: nacosStore.selectedInstanceId.toString(),
            namespaceId: targetNamespace
        }
    });
  } else {
    ElMessage.warning('Please select a Nacos instance first.');
  }
};

const handleEdit = (configId: number) => {
  router.push({ name: 'ConfigurationEdit', params: { id: configId.toString() } });
};

const handleViewHistory = (configId: number) => {
  router.push({ name: 'ConfigurationHistory', params: { id: configId.toString() } });
};


const openPublishDialog = (config: ConfigType) => {
    configToPublish.value = config;
    publishDialogVisible.value = true;
};

const resetPublishForm = () => {
    Object.assign(publishForm, initialPublishFormState);
    if (publishFormRef.value) {
        publishFormRef.value.clearValidate();
    }
    configToPublish.value = null;
};

const confirmPublish = async () => {
    if (!configToPublish.value || !publishFormRef.value) return;

    await publishFormRef.value.validate(async (valid) => {
        if (valid) {
            isPublishing.value = true;
            try {
                await api.publishConfiguration(configToPublish.value!.ID, {
                    publishType: publishForm.publishType,
                    operator: publishForm.operator,
                });
                ElMessage.success('Configuration published successfully!');
                publishDialogVisible.value = false;
                fetchConfigurations(); // Refresh the list to show updated sync status
            } catch (error) {
                console.error('Failed to publish configuration:', error);
                // Error handled by global interceptor
            } finally {
                isPublishing.value = false;
            }
        }
    });
};


onMounted(() => {
  // Restore selections from Pinia store if available
  if (nacosStore.selectedInstanceId) {
    selectedInstanceId.value = nacosStore.selectedInstanceId;
  }
  if (nacosStore.currentNamespaceId !== null) { // Can be empty string for public
    activeNamespace.value = nacosStore.currentNamespaceId;
  }

  fetchNacosInstances(); // This will also trigger dependent fetches if selections are restored
});


// Watch for external changes to Pinia store (e.g., if another component changes the instance)
watch(() => nacosStore.selectedInstanceId, (newId) => {
  if (newId !== selectedInstanceId.value) { // Avoid loop if change originated here
    selectedInstanceId.value = newId; // Update local model for el-select
    if (newId) {
      fetchNamespacesForInstance(newId); // This will also update store's namespaces and trigger config fetch
    } else {
      // Instance cleared
      nacosStore.setNamespaces([]);
      nacosStore.setCurrentNamespace(null);
      configurations.value = [];
      activeNamespace.value = '';
    }
  }
});

watch(() => nacosStore.currentNamespaceId, (newNsId) => {
  const currentTabNs = (activeNamespace.value === undefined || activeNamespace.value === null) ? '' : String(activeNamespace.value);
  const storeNs = (newNsId === undefined || newNsId === null) ? '' : String(newNsId);

  if (storeNs !== currentTabNs) { // Avoid loop if change originated here
    activeNamespace.value = storeNs; // Update local model for el-tabs
    if (nacosStore.hasSelectedInstance) {
      fetchConfigurations();
    }
  }
});

</script>

<style lang="scss" scoped>
.configurations-view {
  // View specific styles
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.control-row {
  align-items: flex-end; // Align items to bottom for better visual with tabs
  margin-bottom: 20px;
}
.full-width {
  width: 100%;
}
.mb-20 {
  margin-bottom: 20px;
}
.mt-20 {
    margin-top: 20px;
}
.namespace-tabs-container {
  display: flex;
  align-items: center;
  .namespace-label {
    margin-right: 10px;
    font-size: 14px;
    color: #606266;
    white-space: nowrap;
  }
  .namespace-tabs {
    flex-grow: 1; // Allow tabs to take available space
    // Reset bottom margin for tabs inside this container if needed
    :deep(.el-tabs__header) {
      margin-bottom: 0;
    }
  }
}
</style>
