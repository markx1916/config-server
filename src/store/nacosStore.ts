import { defineStore } from 'pinia';
import { ElMessage } from 'element-plus';
import {
  NacosInstance,
  getNacosInstances as apiGetNacosInstances,
  createNacosInstance as apiCreateNacosInstance,
  updateNacosInstance as apiUpdateNacosInstance,
  deleteNacosInstance as apiDeleteNacosInstance,
  CreateNacosInstancePayload,
  UpdateNacosInstancePayload,
} from '../services/api';

interface NacosState {
  instances: NacosInstance[];
  selectedInstanceId: number | null;
  loading: boolean;
  error: string | null;
}

export const useNacosStore = defineStore('nacos', {
  state: (): NacosState => ({
    instances: [],
    selectedInstanceId: null,
    loading: false,
    error: null,
  }),
  getters: {
    selectedInstance: (state): NacosInstance | null => {
      if (state.selectedInstanceId === null) return null;
      return state.instances.find(inst => inst.id === state.selectedInstanceId) || null;
    },
    instanceOptions: (state) => {
        return state.instances.map(instance => ({
            value: instance.id,
            label: `${instance.name} (${instance.server_addr})`
        }));
    }
  },
  actions: {
    async fetchInstances() {
      this.loading = true;
      this.error = null;
      try {
        const response = await apiGetNacosInstances();
        this.instances = response.data;
        if (this.instances.length > 0 && !this.selectedInstanceId) {
          // Automatically select the first instance if none is selected
          // this.selectedInstanceId = this.instances[0].id;
        }
      } catch (err: any) {
        this.error = err.message || 'Failed to fetch Nacos instances';
        ElMessage.error(this.error);
      } finally {
        this.loading = false;
      }
    },
    async createInstance(payload: CreateNacosInstancePayload) {
      this.loading = true;
      this.error = null;
      try {
        const response = await apiCreateNacosInstance(payload);
        this.instances.push(response.data);
        ElMessage.success('Nacos instance created successfully');
        return response.data;
      } catch (err: any) {
        this.error = err.message || 'Failed to create Nacos instance';
        ElMessage.error(this.error);
        throw err; // Re-throw to be caught in component
      } finally {
        this.loading = false;
      }
    },
    async updateInstance(id: number, payload: UpdateNacosInstancePayload) {
      this.loading = true;
      this.error = null;
      try {
        const response = await apiUpdateNacosInstance(id, payload);
        const index = this.instances.findIndex(inst => inst.id === id);
        if (index !== -1) {
          this.instances[index] = response.data;
        }
        ElMessage.success('Nacos instance updated successfully');
        return response.data;
      } catch (err: any) {
        this.error = err.message || 'Failed to update Nacos instance';
        ElMessage.error(this.error);
        throw err;
      } finally {
        this.loading = false;
      }
    },
    async deleteInstance(id: number) {
      this.loading = true;
      this.error = null;
      try {
        await apiDeleteNacosInstance(id);
        this.instances = this.instances.filter(inst => inst.id !== id);
        if (this.selectedInstanceId === id) {
          this.selectedInstanceId = this.instances.length > 0 ? this.instances[0].id : null;
        }
        ElMessage.success('Nacos instance deleted successfully');
      } catch (err: any) {
        this.error = err.message || 'Failed to delete Nacos instance';
        ElMessage.error(this.error);
      } finally {
        this.loading = false;
      }
    },
    selectInstance(id: number | null) {
      this.selectedInstanceId = id;
      // If an instance is selected, ensure it exists in the list.
      // If not, it might be stale, so clear it.
      if (id !== null && !this.instances.find(inst => inst.id === id)) {
          this.selectedInstanceId = null;
      }
    },
  },
});
