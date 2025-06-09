<template>
  <div class="publish-records-view container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>Publish Records</span>
          <!-- Add any controls like refresh button if needed -->
        </div>
      </template>

      <el-form :model="filterParams" inline class="mb-20 filter-form">
        <el-form-item label="Config ID">
          <el-input v-model.number="filterParams.configurationId" placeholder="Enter Configuration ID" clearable />
        </el-form-item>
        <el-form-item label="Instance ID">
          <el-input v-model.number="filterParams.nacosInstanceId" placeholder="Enter Nacos Instance ID" clearable />
        </el-form-item>
        <el-form-item label="Status">
          <el-select v-model="filterParams.status" placeholder="Select Status" clearable>
            <el-option label="SUCCESS" value="SUCCESS"></el-option>
            <el-option label="FAILED" value="FAILED"></el-option>
            <el-option label="PENDING" value="PENDING"></el-option>
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="fetchRecords" icon="Search">Filter</el-button>
          <el-button @click="resetFilters" icon="RefreshRight">Reset</el-button>
        </el-form-item>
      </el-form>

      <el-table :data="publishRecords" v-loading="isLoading" stripe style="width: 100%">
        <el-table-column prop="ID" label="Record ID" width="100" sortable />
        <el-table-column label="Configuration" min-width="250" show-overflow-tooltip>
            <template #default="scope">
                <div v-if="scope.row.Configuration">
                    <strong>ID:</strong> {{ scope.row.ConfigurationID }} <br/>
                    <strong>DataID:</strong> {{ scope.row.Configuration.DataID }} <br/>
                    <strong>Group:</strong> {{ scope.row.Configuration.Group }} <br/>
                    <strong>Namespace:</strong> {{ scope.row.Configuration.NamespaceID || 'public' }}
                </div>
                <span v-else>Config ID: {{ scope.row.ConfigurationID }}</span>
            </template>
        </el-table-column>
         <el-table-column label="Nacos Instance" min-width="180" show-overflow-tooltip>
            <template #default="scope">
                 <div v-if="scope.row.NacosInstance">
                    <strong>ID:</strong> {{ scope.row.NacosInstanceID }} <br/>
                    {{ scope.row.NacosInstance.Description || scope.row.NacosInstance.InstanceURL }}
                </div>
                <span v-else>Instance ID: {{ scope.row.NacosInstanceID }}</span>
            </template>
        </el-table-column>
        <el-table-column prop="PublishType" label="Type" width="100" />
        <el-table-column prop="Status" label="Status" width="120" sortable>
            <template #default="scope">
                <el-tag :type="getStatusTagType(scope.row.Status)">
                    {{ scope.row.Status }}
                </el-tag>
            </template>
        </el-table-column>
        <el-table-column prop="Operator" label="Operator" width="150" />
        <el-table-column prop="PublishedAt" label="Published At" width="180" sortable>
          <template #default="scope">
            {{ new Date(scope.row.PublishedAt).toLocaleString() }}
          </template>
        </el-table-column>
        <el-table-column prop="Details" label="Details" min-width="200" show-overflow-tooltip/>
        <el-table-column label="Actions" width="100" fixed="right">
             <template #default="scope">
                <el-button size="small" type="info" @click="viewRecordDetails(scope.row)" icon="View">Details</el-button>
            </template>
        </el-table-column>
      </el-table>
      <el-empty v-if="!isLoading && publishRecords.length === 0" description="No publish records found."></el-empty>
    </el-card>

    <!-- Dialog to View Publish Record Details -->
    <el-dialog v-model="detailsDialogVisible" title="Publish Record Details" width="60%" top="5vh">
      <div v-if="selectedRecord">
        <el-descriptions :column="2" border>
            <el-descriptions-item label="Record ID">{{ selectedRecord.ID }}</el-descriptions-item>
            <el-descriptions-item label="Status">
                 <el-tag :type="getStatusTagType(selectedRecord.Status)">
                    {{ selectedRecord.Status }}
                </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="Publish Type">{{ selectedRecord.PublishType }}</el-descriptions-item>
            <el-descriptions-item label="Operator">{{ selectedRecord.Operator }}</el-descriptions-item>
            <el-descriptions-item label="Published At">{{ new Date(selectedRecord.PublishedAt).toLocaleString() }}</el-descriptions-item>

            <el-descriptions-item label="Config ID">{{ selectedRecord.ConfigurationID }}</el-descriptions-item>
            <el-descriptions-item label="Config Data ID" v-if="selectedRecord.Configuration">{{ selectedRecord.Configuration.DataID }}</el-descriptions-item>
            <el-descriptions-item label="Config Group" v-if="selectedRecord.Configuration">{{ selectedRecord.Configuration.Group }}</el-descriptions-item>
            <el-descriptions-item label="Config Namespace" v-if="selectedRecord.Configuration">{{ selectedRecord.Configuration.NamespaceID || 'public' }}</el-descriptions-item>

            <el-descriptions-item label="Instance ID">{{ selectedRecord.NacosInstanceID }}</el-descriptions-item>
            <el-descriptions-item label="Instance URL" v-if="selectedRecord.NacosInstance">{{ selectedRecord.NacosInstance.InstanceURL }}</el-descriptions-item>
        </el-descriptions>
        <h4 class="mt-20">Details:</h4>
        <pre class="details-pre">{{ selectedRecord.Details }}</pre>
      </div>
      <template #footer>
        <el-button @click="detailsDialogVisible = false">Close</el-button>
      </template>
    </el-dialog>

  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted, reactive } from 'vue';
import { ElCard, ElTable, ElTableColumn, ElButton, ElIcon, ElTag, ElEmpty, ElForm, ElFormItem, ElInput, ElSelect, ElOption, ElDialog, ElDescriptions, ElDescriptionsItem } from 'element-plus';
import { Search, RefreshRight, View } from '@element-plus/icons-vue';
import * as api from '@/services/api';
import type { PublishRecord, ListPublishRecordsParams } from '@/types';

const publishRecords = ref<PublishRecord[]>([]);
const isLoading = ref(false);
const filterParams = reactive<ListPublishRecordsParams>({
  configurationId: undefined,
  nacosInstanceId: undefined,
  status: '',
});

const detailsDialogVisible = ref(false);
const selectedRecord = ref<PublishRecord | null>(null);

const fetchRecords = async () => {
  isLoading.value = true;
  try {
    // Prepare params, remove empty/undefined values
    const queryParams: ListPublishRecordsParams = {};
    if (filterParams.configurationId) queryParams.configurationId = filterParams.configurationId;
    if (filterParams.nacosInstanceId) queryParams.nacosInstanceId = filterParams.nacosInstanceId;
    if (filterParams.status) queryParams.status = filterParams.status;

    const response = await api.getPublishRecords(queryParams);
    publishRecords.value = response.data;
  } catch (error) {
    console.error('Failed to fetch publish records:', error);
    publishRecords.value = []; // Clear on error
  } finally {
    isLoading.value = false;
  }
};

const resetFilters = () => {
  filterParams.configurationId = undefined;
  filterParams.nacosInstanceId = undefined;
  filterParams.status = '';
  fetchRecords();
};

const getStatusTagType = (status: string) => {
  switch (status) {
    case 'SUCCESS': return 'success';
    case 'FAILED': return 'danger';
    case 'PENDING': return 'warning';
    default: return 'info';
  }
};

const viewRecordDetails = (record: PublishRecord) => {
    selectedRecord.value = record;
    detailsDialogVisible.value = true;
};

onMounted(() => {
  fetchRecords();
});
</script>

<style lang="scss" scoped>
.publish-records-view {
  // Styles for this view
}
.filter-form .el-form-item {
  margin-bottom: 10px; // Reduce bottom margin for inline form items
}
.details-pre {
    background-color: #f5f5f5;
    padding: 10px;
    border-radius: 4px;
    white-space: pre-wrap;
    word-break: break-all;
    max-height: 300px;
    overflow-y: auto;
}
.mt-20 {
    margin-top: 20px;
}
</style>
