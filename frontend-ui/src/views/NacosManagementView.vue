<template>
  <div class="nacos-management-view container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>Nacos Server Instances</span>
          <el-button type="primary" @click="openAddInstanceDialog" icon="Plus">
            Add Nacos Server
          </el-button>
        </div>
      </template>

      <el-table :data="nacosInstances" v-loading="isLoading" stripe style="width: 100%">
        <el-table-column prop="ID" label="ID" width="80" sortable />
        <el-table-column label="Description / URL" min-width="200" sortable prop="Description">
            <template #default="scope">
                {{ scope.row.Description || scope.row.InstanceURL }}
            </template>
        </el-table-column>
        <el-table-column prop="InstanceURL" label="Instance URL" min-width="200" sortable />
        <el-table-column prop="NamespaceID" label="Default Namespace" width="180" sortable>
            <template #default="scope">
                {{ scope.row.NamespaceID || 'public' }}
            </template>
        </el-table-column>
        <el-table-column prop="Scheme" label="Scheme" width="100" />
        <el-table-column prop="ContextPath" label="Context Path" width="150" />
        <el-table-column label="Actions" width="380" fixed="right">
          <template #default="scope">
            <el-button size="small" @click="openEditInstanceDialog(scope.row)" icon="Edit">Edit</el-button>
            <el-button size="small" type="info" @click="handleTestConnection(scope.row)" icon="Connection" :loading="scope.row._testingConnection">Test</el-button>
            <el-button size="small" type="success" @click="handleManageConfigurations(scope.row)" icon="Files">Configs</el-button>
            <el-button size="small" type="danger" @click="handleDeleteInstance(scope.row)" icon="Delete">Delete</el-button>
          </template>
        </el-table-column>
      </el-table>
       <el-empty v-if="!isLoading && nacosInstances.length === 0" description="No Nacos instances configured yet."></el-empty>
    </el-card>

    <!-- Add/Edit Nacos Instance Dialog -->
    <el-dialog
      v-model="instanceDialogVisible"
      :title="isEditMode ? 'Edit Nacos Instance' : 'Add New Nacos Instance'"
      width="600px"
      @closed="resetInstanceForm"
      draggable
      destroy-on-close
    >
      <el-form :model="instanceForm" :rules="instanceFormRules" ref="instanceFormRef" label-width="140px" label-position="right">
        <el-form-item label="Instance URL" prop="instanceUrl">
          <el-input v-model="instanceForm.instanceUrl" placeholder="e.g., http://localhost:8848 or server1:8848,server2:8848" />
        </el-form-item>
        <el-form-item label="Description" prop="description">
          <el-input v-model="instanceForm.description" placeholder="e.g., Production Nacos Cluster" />
        </el-form-item>
        <el-form-item label="Default Namespace ID" prop="namespaceId">
          <el-input v-model="instanceForm.namespaceId" placeholder="Leave blank for public namespace" />
        </el-form-item>
        <el-form-item label="Scheme" prop="scheme">
           <el-select v-model="instanceForm.scheme" placeholder="Select scheme (default: http)">
            <el-option label="http" value="http"></el-option>
            <el-option label="https" value="https"></el-option>
          </el-select>
        </el-form-item>
        <el-form-item label="Context Path" prop="contextPath">
          <el-input v-model="instanceForm.contextPath" placeholder="/nacos" />
        </el-form-item>
        <el-form-item label="Username" prop="username">
          <el-input v-model="instanceForm.username" placeholder="Optional Nacos username" />
        </el-form-item>
        <el-form-item label="Password" prop="password">
          <el-input type="password" v-model="instanceForm.password" placeholder="Optional Nacos password" show-password />
           <span v-if="isEditMode && instanceForm.id && !instanceForm.password" class="form-text-info">Leave blank to keep existing password.</span>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="instanceDialogVisible = false">Cancel</el-button>
        <el-button type="primary" @click="saveInstance" :loading="isSaving">Save</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted, reactive } from 'vue';
import { useRouter } from 'vue-router';
import { ElCard, ElTable, ElTableColumn, ElButton, ElIcon, ElDialog, ElForm, ElFormItem, ElInput, ElSelect, ElOption, ElMessage, ElMessageBox, ElEmpty } from 'element-plus';
import { Plus, Edit, Delete, Connection, Files } from '@element-plus/icons-vue';
import * as api from '@/services/api';
import type { NacosInstance, NacosInstancePayload } from '@/types';
import { useNacosStore } from '@/store';

const router = useRouter();
const nacosStore = useNacosStore();

const nacosInstances = ref<(NacosInstance & { _testingConnection?: boolean })[]>([]);
const isLoading = ref(false);
const isSaving = ref(false);

const instanceDialogVisible = ref(false);
const isEditMode = ref(false);
const instanceFormRef = ref<InstanceType<typeof ElForm>>();
const initialInstanceFormState: NacosInstancePayload & { id?: number } = {
  id: undefined,
  instanceUrl: '',
  namespaceId: '',
  description: '',
  username: '',
  password: '',
  scheme: 'http',
  contextPath: '/nacos',
};
const instanceForm = reactive({ ...initialInstanceFormState });

const instanceFormRules = {
  instanceUrl: [{ required: true, message: 'Instance URL is required', trigger: 'blur' }],
  scheme: [{ required: true, message: 'Scheme is required', trigger: 'change' }],
  // contextPath: [{ required: true, message: 'Context Path is required', trigger: 'blur' }], // Not strictly required, Nacos default is /nacos
};

const fetchNacosInstances = async () => {
  isLoading.value = true;
  try {
    const response = await api.getNacosInstances();
    nacosInstances.value = response.data.map(inst => ({ ...inst, _testingConnection: false }));
  } catch (error) {
    console.error('Failed to fetch Nacos instances:', error);
    // Error notification is handled by global interceptor
  } finally {
    isLoading.value = false;
  }
};

const resetInstanceForm = () => {
  Object.assign(instanceForm, initialInstanceFormState); // Reset to initial state
  isEditMode.value = false;
  if (instanceFormRef.value) {
    instanceFormRef.value.clearValidate();
    instanceFormRef.value.resetFields(); // This might also work if fields are initially empty in definition
  }
};

const openAddInstanceDialog = () => {
  resetInstanceForm(); // Ensure form is clean
  instanceDialogVisible.value = true;
};

const openEditInstanceDialog = (instance: NacosInstance) => {
  resetInstanceForm();
  isEditMode.value = true;
  instanceForm.id = instance.ID;
  instanceForm.instanceUrl = instance.InstanceURL;
  instanceForm.namespaceId = instance.NamespaceID;
  instanceForm.description = instance.Description;
  instanceForm.username = instance.Username || '';
  instanceForm.password = ''; // Password is not fetched; leave blank to keep existing
  instanceForm.scheme = instance.Scheme || 'http';
  instanceForm.contextPath = instance.ContextPath || '/nacos';
  instanceDialogVisible.value = true;
};

const saveInstance = async () => {
  if (!instanceFormRef.value) return;
  await instanceFormRef.value.validate(async (valid) => {
    if (valid) {
      isSaving.value = true;
      const payload: NacosInstancePayload = {
        instanceUrl: instanceForm.instanceUrl,
        namespaceId: instanceForm.namespaceId,
        description: instanceForm.description,
        username: instanceForm.username,
        password: instanceForm.password, // Send password only if user entered something
        scheme: instanceForm.scheme,
        contextPath: instanceForm.contextPath,
      };
      // If password is empty in edit mode, do not send it in payload to keep existing
      if (isEditMode.value && !payload.password) {
        delete payload.password;
      }


      try {
        if (isEditMode.value && instanceForm.id) {
          await api.updateNacosInstance(instanceForm.id, payload);
          ElMessage.success('Nacos instance updated successfully!');
        } else {
          await api.createNacosInstance(payload);
          ElMessage.success('Nacos instance added successfully!');
        }
        instanceDialogVisible.value = false;
        fetchNacosInstances(); // Refresh list
      } catch (error) {
        console.error('Failed to save Nacos instance:', error);
        // Error notification is handled by global interceptor
      } finally {
        isSaving.value = false;
      }
    }
  });
};

const handleDeleteInstance = async (instance: NacosInstance) => {
  try {
    await ElMessageBox.confirm(
      `Are you sure you want to delete instance "${instance.Description || instance.InstanceURL}"? This action cannot be undone. Associated configurations in this tool will also be affected.`,
      'Confirm Deletion',
      {
        confirmButtonText: 'Delete',
        cancelButtonText: 'Cancel',
        type: 'warning',
      }
    );
    await api.deleteNacosInstance(instance.ID);
    ElMessage.success('Nacos instance deleted successfully!');
    fetchNacosInstances(); // Refresh list
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Failed to delete Nacos instance:', error);
      // Error notification is handled by global interceptor, but specific messages for FK issues might be good.
      // The backend should return a 409 Conflict with a clear message if FK constraints prevent deletion.
    }
  }
};

const handleTestConnection = async (instance: NacosInstance & { _testingConnection?: boolean }) => {
  instance._testingConnection = true;
  try {
    const response = await api.testNacosInstanceConnection(instance.ID);
    if (response.data.status === 'successful') {
      ElMessage.success(`Connection to ${instance.InstanceURL} successful!`);
    } else {
      ElMessage.error(`Connection test failed: ${response.data.message || 'Unknown error'}`);
    }
  } catch (error) {
    console.error('Failed to test Nacos connection:', error);
    // Error notification handled by global interceptor
  } finally {
    instance._testingConnection = false;
  }
};

const handleManageConfigurations = (instance: NacosInstance) => {
  nacosStore.selectInstance({ id: instance.ID, instanceUrl: instance.InstanceURL });
  // Fetch namespaces for this instance directly after selecting it
  // This ensures the ConfigurationsView has the namespaces when it loads or when instance changes
  api.getNacosInstanceNamespaces(instance.ID).then(response => {
    const namespaces = response.data.map((ns: any) => ({
        namespaceId: ns.namespace,
        namespaceShowName: ns.namespaceShowName || ns.namespace,
    }));
    nacosStore.setNamespaces(namespaces);
    // Default to first namespace or public after fetching
    if (namespaces.length > 0) {
        nacosStore.setCurrentNamespace(namespaces[0].namespaceId);
    } else {
        nacosStore.setCurrentNamespace(''); // Public
    }
    router.push('/configurations');
  }).catch(error => {
    console.error('Failed to fetch namespaces on instance selection:', error);
    nacosStore.setNamespaces([]); // Clear namespaces if fetch fails
    nacosStore.setCurrentNamespace(''); // Default to public
    router.push('/configurations'); // Navigate anyway, view will handle empty namespaces
  });
};


onMounted(() => {
  fetchNacosInstances();
});
</script>

<style lang="scss" scoped>
.nacos-management-view {
  // Styles for this view
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.form-text-info {
  font-size: 0.85em;
  color: #909399;
  margin-left: 10px;
}
</style>
