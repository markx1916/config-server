<template>
  <div class="config-list-view">
    <h1>Configuration List</h1>

    <el-row :gutter="20" style="margin-bottom: 20px;">
      <el-col :span="8">
        <el-select
          v-model="nacosStore.selectedInstanceId"
          placeholder="Select Nacos Instance"
          @change="handleNacosInstanceChange"
          filterable
          style="width: 100%;"
        >
          <el-option
            v-for="instance in nacosStore.instanceOptions"
            :key="instance.value"
            :label="instance.label"
            :value="instance.value"
          ></el-option>
        </el-select>
      </el-col>
      <el-col :span="5">
        <el-input v-model="filterDataId" placeholder="Filter by Data ID" @input="debouncedFetchConfigs"></el-input>
      </el-col>
      <el-col :span="5">
        <el-input v-model="filterGroup" placeholder="Filter by Group" @input="debouncedFetchConfigs"></el-input>
      </el-col>
      <el-col :span="6" style="text-align: right;">
        <el-button
          type="primary"
          @click="navigateToCreateConfig"
          :disabled="!nacosStore.selectedInstanceId"
        >
          <el-icon><DocumentAdd /></el-icon> Create Configuration
        </el-button>
      </el-col>
    </el-row>

    <div v-if="!nacosStore.selectedInstanceId" class="empty-state">
      <el-empty description="Please select a Nacos Instance to view configurations."></el-empty>
    </div>
    <div v-else>
      <el-table :data="configStore.configurations" v-loading="configStore.loading || nacosStore.loading" border style="width: 100%">
        <el-table-column prop="id" label="ID" width="80" sortable></el-table-column>
        <el-table-column prop="data_id" label="Data ID" sortable>
            <template #default="scope">
                <el-link type="primary" @click="navigateToViewConfig(scope.row.id)">{{ scope.row.data_id }}</el-link>
            </template>
        </el-table-column>
        <el-table-column prop="group_name" label="Group" sortable></el-table-column>
        <el-table-column prop="format" label="Format" width="100"></el-table-column>
        <el-table-column prop="version" label="Version" width="100" align="center"></el-table-column>
        <el-table-column prop="updated_at" label="Last Modified" width="180" sortable>
          <template #default="scope">{{ new Date(scope.row.updated_at).toLocaleString() }}</template>
        </el-table-column>
        <el-table-column label="Actions" width="180" align="center">
          <template #default="scope">
            <el-tooltip content="Edit" placement="top">
              <el-button size="small" circle @click="navigateToEditConfig(scope.row.id)">
                <el-icon><Edit /></el-icon>
              </el-button>
            </el-tooltip>
            <el-tooltip content="Publish/History" placement="top">
                 <el-button size="small" circle @click="navigateToViewConfig(scope.row.id)" type="info">
                    <el-icon><View /></el-icon>
                </el-button>
            </el-tooltip>
            <el-tooltip content="Delete (Local DB only)" placement="top">
              <el-popconfirm
                title="Delete this configuration (local DB only)?"
                confirm-button-text="Yes"
                cancel-button-text="No"
                @confirm="handleDeleteConfig(scope.row.id)"
              >
                <template #reference>
                  <el-button size="small" type="danger" circle>
                    <el-icon><Delete /></el-icon>
                  </el-button>
                </template>
              </el-popconfirm>
            </el-tooltip>
          </template>
        </el-table-column>
      </el-table>
       <el-pagination
        v-if="configStore.configurations.length"
        layout="prev, pager, next, sizes, total"
        :total="configStore.configurations.length"
        :page-sizes="[10, 20, 50, 100]"
        style="margin-top: 20px; justify-content: flex-end;"
      ></el-pagination>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted, watch, computed } from 'vue';
import { useRouter } from 'vue-router';
import { useNacosStore } from '../store/nacosStore';
import { useConfigStore } from '../store/configStore';
import { ElMessage, ElMessageBox }  from 'element-plus';
import { DocumentAdd, Edit, Delete, View } from '@element-plus/icons-vue';
import { debounce } from 'lodash-es'; // Using lodash-es for debounce

const router = useRouter();
const nacosStore = useNacosStore();
const configStore = useConfigStore();

const filterDataId = ref('');
const filterGroup = ref('');

// Fetch instances when component mounts
onMounted(async () => {
  if (nacosStore.instances.length === 0) {
    await nacosStore.fetchInstances();
  }
  // If an instance is already selected (e.g. from localStorage or previous navigation)
  if (nacosStore.selectedInstanceId) {
    fetchConfigsForSelectedInstance();
  }
});

const fetchConfigsForSelectedInstance = () => {
  if (nacosStore.selectedInstanceId) {
    configStore.fetchConfigurations(nacosStore.selectedInstanceId, filterDataId.value, filterGroup.value);
  } else {
    configStore.configurations = []; // Clear configurations if no instance is selected
  }
};

// Debounce fetching configurations to avoid rapid API calls on filter input
const debouncedFetchConfigs = debounce(fetchConfigsForSelectedInstance, 500);


const handleNacosInstanceChange = (instanceId: number | null) => {
  nacosStore.selectInstance(instanceId); // Update store's selectedInstanceId
  fetchConfigsForSelectedInstance();
};

// Watch for changes in selectedInstanceId from other components/actions
watch(() => nacosStore.selectedInstanceId, (newId) => {
  if (newId) {
    fetchConfigsForSelectedInstance();
  } else {
    configStore.configurations = [];
  }
});


const navigateToCreateConfig = () => {
  if (!nacosStore.selectedInstanceId) {
    ElMessage.warning('Please select a Nacos instance first.');
    return;
  }
  // Pass nacos_instance_id to the editor, so it knows which instance this new config belongs to.
  router.push({ name: 'config-editor', params: { id: 'new' }, query: { nacosInstanceId: nacosStore.selectedInstanceId }});
};

const navigateToEditConfig = (configId: number) => {
  router.push({ name: 'config-editor', params: { id: configId } });
};

const navigateToViewConfig = (configId: number) => {
  // This will navigate to a detailed view that includes history, publish options, etc.
  // I'll create/rename this view later. For now, let's assume 'config-detail'.
  router.push({ name: 'config-detail', params: { id: configId } });
};

const handleDeleteConfig = async (configId: number) => {
  // This should be a store action if it involves API call.
  // For now, this is a placeholder as backend doesn't have delete config yet.
  ElMessageBox.confirm(
    'This will delete the configuration from the local database. Are you sure?',
    'Warning',
    {
      confirmButtonText: 'OK',
      cancelButtonText: 'Cancel',
      type: 'warning',
    }
  ).then(async () => {
    // await configStore.deleteConfiguration(configId); // Assuming this action exists
    ElMessage.info('Local delete functionality not yet fully implemented with backend.');
    // After deletion, refetch:
    // fetchConfigsForSelectedInstance();
  }).catch(() => {
    ElMessage.info('Delete canceled');
  });
};

</script>

<style scoped>
.config-list-view {
  padding: 20px;
}
.el-row {
  align-items: center;
}
.empty-state {
  margin-top: 40px;
  text-align: center;
}
</style>
