import { defineStore } from 'pinia';
import { ElMessage } from 'element-plus';
import {
  Configuration,
  ConfigurationHistory,
  DeploymentHistory,
  DiffResponse,
  listConfigurations as apiListConfigurations,
  getConfigurationById as apiGetConfigurationById,
  createConfiguration as apiCreateConfiguration,
  updateConfiguration as apiUpdateConfiguration,
  getConfigurationDiff as apiGetConfigurationDiff,
  publishConfiguration as apiPublishConfiguration,
  getConfigurationSaveHistory as apiGetConfigurationSaveHistory,
  rollbackConfiguration as apiRollbackConfiguration,
  getConfigurationDeploymentHistory as apiGetConfigurationDeploymentHistory,
  CreateConfigurationPayload,
  UpdateConfigurationPayload,
} from '../services/api';
import { useNacosStore } from './nacosStore'; // To get selected Nacos instance

interface ConfigState {
  configurations: Configuration[];
  currentConfiguration: Configuration | null;
  currentConfigSaveHistory: ConfigurationHistory[];
  currentConfigDeploymentHistory: DeploymentHistory[];
  currentConfigDiff: DiffResponse | null;
  loading: boolean;
  error: string | null;
  publishing: boolean;
}

export const useConfigStore = defineStore('config', {
  state: (): ConfigState => ({
    configurations: [],
    currentConfiguration: null,
    currentConfigSaveHistory: [],
    currentConfigDeploymentHistory: [],
    currentConfigDiff: null,
    loading: false,
    error: null,
    publishing: false,
  }),
  actions: {
    async fetchConfigurations(nacosInstanceId: number, dataId?: string, group?: string) {
      this.loading = true;
      this.error = null;
      try {
        const response = await apiListConfigurations(nacosInstanceId, dataId, group);
        this.configurations = response.data;
      } catch (err: any) {
        this.error = err.message || 'Failed to fetch configurations';
        ElMessage.error(this.error);
        this.configurations = []; // Clear on error
      } finally {
        this.loading = false;
      }
    },
    async fetchConfigurationById(id: number) {
      this.loading = true;
      this.error = null;
      try {
        const response = await apiGetConfigurationById(id);
        this.currentConfiguration = response.data;
        return response.data;
      } catch (err: any) {
        this.error = err.message || `Failed to fetch configuration ${id}`;
        ElMessage.error(this.error);
        this.currentConfiguration = null;
        throw err;
      } finally {
        this.loading = false;
      }
    },
    async createConfiguration(payload: CreateConfigurationPayload) {
      this.loading = true;
      this.error = null;
      try {
        const response = await apiCreateConfiguration(payload);
        // No need to push to this.configurations here, as list view will refetch or be on different page.
        ElMessage.success('Configuration created successfully');
        return response.data; // Return for navigation or further use
      } catch (err: any) {
        this.error = err.message || 'Failed to create configuration';
        // Error already shown by api.ts interceptor, but can add more context
        // ElMessage.error(this.error);
        throw err;
      } finally {
        this.loading = false;
      }
    },
    async updateConfiguration(id: number, payload: UpdateConfigurationPayload) {
      this.loading = true;
      this.error = null;
      try {
        const response = await apiUpdateConfiguration(id, payload);
        this.currentConfiguration = response.data; // Update current if it's the one being edited
        ElMessage.success('Configuration updated successfully');
        return response.data;
      } catch (err: any) {
        this.error = err.message || 'Failed to update configuration';
        // ElMessage.error(this.error);
        throw err;
      } finally {
        this.loading = false;
      }
    },
    async fetchDiff(id: number, compareToVersion?: number) {
      this.loading = true;
      this.error = null;
      try {
        const response = await apiGetConfigurationDiff(id, compareToVersion);
        this.currentConfigDiff = response.data;
        return response.data;
      } catch (err: any) {
        this.error = err.message || 'Failed to fetch diff';
        ElMessage.error(this.error);
        this.currentConfigDiff = null;
        throw err;
      } finally {
        this.loading = false;
      }
    },
    async publishConfiguration(id: number, type: 'gray' | 'full', betaIps?: string) {
      this.publishing = true;
      this.error = null;
      try {
        const response = await apiPublishConfiguration(id, type, betaIps);
        ElMessage.success(response.data.message || 'Publish command sent successfully.');
        // Potentially update deployment history or config status here
      } catch (err: any) {
        this.error = err.message || `Failed to publish configuration (type: ${type})`;
        // Error already shown by api.ts interceptor
        // ElMessage.error(this.error);
        throw err;
      } finally {
        this.publishing = false;
      }
    },
    async fetchSaveHistory(id: number) {
      this.loading = true;
      this.error = null;
      try {
        const response = await apiGetConfigurationSaveHistory(id);
        this.currentConfigSaveHistory = response.data;
      } catch (err: any) {
        this.error = err.message || 'Failed to fetch save history';
        ElMessage.error(this.error);
        this.currentConfigSaveHistory = [];
      } finally {
        this.loading = false;
      }
    },
    async rollbackConfiguration(id: number, historyId: number) {
      this.loading = true;
      this.error = null;
      try {
        const response = await apiRollbackConfiguration(id, historyId);
        this.currentConfiguration = response.data; // Update current view if it's the one rolled back
        ElMessage.success('Configuration rolled back successfully. New version created.');
        // Fetch new save history to reflect the rollback version
        await this.fetchSaveHistory(id);
        return response.data;
      } catch (err: any) {
        this.error = err.message || 'Failed to rollback configuration';
        // ElMessage.error(this.error);
        throw err;
      } finally {
        this.loading = false;
      }
    },
    async fetchDeploymentHistory(id: number) {
      this.loading = true;
      this.error = null;
      try {
        const response = await apiGetConfigurationDeploymentHistory(id);
        this.currentConfigDeploymentHistory = response.data;
      } catch (err: any) {
        this.error = err.message || 'Failed to fetch deployment history';
        ElMessage.error(this.error);
        this.currentConfigDeploymentHistory = [];
      } finally {
        this.loading = false;
      }
    },
    clearCurrentConfiguration() {
      this.currentConfiguration = null;
      this.currentConfigSaveHistory = [];
      this.currentConfigDeploymentHistory = [];
      this.currentConfigDiff = null;
    },
  },
});
