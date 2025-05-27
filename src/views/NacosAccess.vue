<template>
  <div class="nacos-access-view">
    <h1>Nacos Instance Management</h1>

    <el-button type="primary" @click="handleOpenCreateDialog" style="margin-bottom: 20px;">
      <el-icon><Plus /></el-icon> Add Nacos Instance
    </el-button>

    <el-table :data="nacosStore.instances" v-loading="nacosStore.loading" border style="width: 100%">
      <el-table-column prop="id" label="ID" width="80"></el-table-column>
      <el-table-column prop="name" label="Name" sortable></el-table-column>
      <el-table-column prop="server_addr" label="Server Address" sortable></el-table-column>
      <el-table-column prop="namespace_id" label="Namespace ID"></el-table-column>
      <el-table-column prop="username" label="Username"></el-table-column>
      <el-table-column label="Actions" width="200">
        <template #default="scope">
          <el-button size="small" @click="handleOpenEditDialog(scope.row)">
            <el-icon><Edit /></el-icon> Edit
          </el-button>
          <el-popconfirm
            title="Are you sure you want to delete this instance?"
            confirm-button-text="Yes"
            cancel-button-text="No"
            @confirm="handleDeleteInstance(scope.row.id)"
          >
            <template #reference>
              <el-button size="small" type="danger">
                <el-icon><Delete /></el-icon> Delete
              </el-button>
            </template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog
      :title="isEditMode ? 'Edit Nacos Instance' : 'Create Nacos Instance'"
      v-model="dialogVisible"
      width="500px"
      @closed="resetForm"
    >
      <el-form :model="currentInstanceForm" :rules="rules" ref="instanceFormRef" label-width="120px">
        <el-form-item label="Name" prop="name">
          <el-input v-model="currentInstanceForm.name" placeholder="e.g., Dev Nacos"></el-input>
        </el-form-item>
        <el-form-item label="Server Address" prop="server_addr">
          <el-input v-model="currentInstanceForm.server_addr" placeholder="e.g., host:port or host1:port1,host2:port2"></el-input>
        </el-form-item>
        <el-form-item label="Namespace ID" prop="namespace_id">
          <el-input v-model="currentInstanceForm.namespace_id" placeholder="Optional"></el-input>
        </el-form-item>
        <el-form-item label="Username" prop="username">
          <el-input v-model="currentInstanceForm.username" placeholder="Optional"></el-input>
        </el-form-item>
        <el-form-item label="Password" prop="password">
          <el-input
            v-model="currentInstanceForm.password"
            type="password"
            :placeholder="isEditMode ? 'Leave blank to keep unchanged' : 'Optional'"
            show-password
          ></el-input>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">Cancel</el-button>
        <el-button type="primary" @click="handleSubmitForm">
          {{ isEditMode ? 'Save Changes' : 'Create' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted, reactive, computed } from 'vue';
import { useNacosStore } from '../store/nacosStore';
import type { NacosInstance, CreateNacosInstancePayload, UpdateNacosInstancePayload } from '../services/api';
import { ElMessage, FormInstance, FormRules } from 'element-plus';
import { Plus, Edit, Delete } from '@element-plus/icons-vue';

const nacosStore = useNacosStore();

const dialogVisible = ref(false);
const isEditMode = ref(false);
const instanceFormRef = ref<FormInstance>();

const initialFormState: CreateNacosInstancePayload & { id?: number; password?: string } = {
  name: '',
  server_addr: '',
  namespace_id: '',
  username: '',
  password: '',
};
const currentInstanceForm = reactive({ ...initialFormState });

const rules = reactive<FormRules>({
  name: [{ required: true, message: 'Please input instance name', trigger: 'blur' }],
  server_addr: [{ required: true, message: 'Please input server address', trigger: 'blur' }],
});

onMounted(() => {
  nacosStore.fetchInstances();
});

const handleOpenCreateDialog = () => {
  isEditMode.value = false;
  Object.assign(currentInstanceForm, initialFormState); // Reset to initial state
  currentInstanceForm.id = undefined; // Ensure no ID for create mode
  dialogVisible.value = true;
};

const handleOpenEditDialog = (instance: NacosInstance) => {
  isEditMode.value = true;
  // Create a mutable copy for the form
  currentInstanceForm.id = instance.id;
  currentInstanceForm.name = instance.name;
  currentInstanceForm.server_addr = instance.server_addr;
  currentInstanceForm.namespace_id = instance.namespace_id || '';
  currentInstanceForm.username = instance.username || '';
  currentInstanceForm.password = ''; // Clear password field for edit mode
  dialogVisible.value = true;
};

const resetForm = () => {
  instanceFormRef.value?.resetFields();
  Object.assign(currentInstanceForm, initialFormState);
  currentInstanceForm.id = undefined;
};

const handleSubmitForm = async () => {
  if (!instanceFormRef.value) return;
  await instanceFormRef.value.validate(async (valid) => {
    if (valid) {
      try {
        if (isEditMode.value && currentInstanceForm.id) {
          const payload: UpdateNacosInstancePayload = {
            name: currentInstanceForm.name,
            server_addr: currentInstanceForm.server_addr,
            namespace_id: currentInstanceForm.namespace_id,
            username: currentInstanceForm.username,
          };
          // Only include password if it's been changed
          if (currentInstanceForm.password) {
            payload.password = currentInstanceForm.password;
          }
          await nacosStore.updateInstance(currentInstanceForm.id, payload);
        } else {
          const payload: CreateNacosInstancePayload = {
            name: currentInstanceForm.name,
            server_addr: currentInstanceForm.server_addr,
            namespace_id: currentInstanceForm.namespace_id,
            username: currentInstanceForm.username,
            password: currentInstanceForm.password,
          };
          await nacosStore.createInstance(payload);
        }
        dialogVisible.value = false;
      } catch (error) {
        // Error is already handled by the store and api.ts interceptor
        console.error("Submit form error:", error)
      }
    } else {
      ElMessage.error('Please correct the errors in the form.');
    }
  });
};

const handleDeleteInstance = async (id: number) => {
  await nacosStore.deleteInstance(id);
};

</script>

<style scoped>
.nacos-access-view {
  padding: 20px;
}
</style>
