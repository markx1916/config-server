import axios from 'axios';
import { ElNotification } from 'element-plus';
import type {
  NacosInstance,
  NacosInstancePayload,
  NacosNamespace,
  Configuration,
  ConfigurationPayload,
  ConfigurationUpdatePayload,
  ConfigurationHistory,
  PublishRecord,
  PublishPayload,
  RollbackPayload,
  // ApiError, // Already declared, not directly used in function signatures here
} from '@/types';

// Base URL for the backend API.
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api';

const apiClient = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
  timeout: 30000, // 30 seconds timeout
});

// Request Interceptor
apiClient.interceptors.request.use(
  (config) => {
    // Example: Add auth token if available
    // const token = localStorage.getItem('authToken');
    // if (token) {
    //   config.headers.Authorization = `Bearer ${token}`;
    // }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Response Interceptor for global error handling
apiClient.interceptors.response.use(
  (response) => response, // Simply return the response if it's successful
  (error) => {
    let errorMessage = 'An unexpected error occurred';
    if (error.response) {
      console.error('API Error Response:', error.response);
      if (error.response.data && error.response.data.error) {
        errorMessage = error.response.data.error;
      } else if (error.response.statusText && error.response.status) {
        errorMessage = `Error ${error.response.status}: ${error.response.statusText}`;
      } else if (error.response.status) {
         errorMessage = `Request failed with status code ${error.response.status}`;
      }
    } else if (error.request) {
      console.error('API No Response:', error.request);
      errorMessage = 'No response from server. Please check your network connection or if the backend is running.';
    } else {
      console.error('API Request Setup Error:', error.message);
      errorMessage = error.message;
    }

    ElNotification({
      title: 'API Error',
      message: errorMessage,
      type: 'error',
      duration: 5000,
    });

    return Promise.reject(error); // Important to reject the promise so individual calls can also catch it
  }
);

// --- Nacos Instance Management ---
export const getNacosInstances = () => apiClient.get<NacosInstance[]>('/nacos-instances');
export const getNacosInstanceById = (id: number) => apiClient.get<NacosInstance>(`/nacos-instances/${id}`);
export const createNacosInstance = (data: NacosInstancePayload) => apiClient.post<NacosInstance>('/nacos-instances', data);
export const updateNacosInstance = (id: number, data: Partial<NacosInstancePayload>) => apiClient.put<NacosInstance>(`/nacos-instances/${id}`, data);
export const deleteNacosInstance = (id: number) => apiClient.delete(`/nacos-instances/${id}`);
export const testNacosInstanceConnection = (id: number) => apiClient.post<{ status: string; message: string }>(`/nacos-instances/${id}/test-connection`);
export const getNacosInstanceNamespaces = (id: number) => apiClient.get<NacosNamespace[]>(`/nacos-instances/${id}/namespaces`);


// --- Configuration Management ---
export interface ListConfigurationsParams {
  nacosInstanceId?: number;
  namespaceId?: string;
  dataId?: string;
  group?: string;
}
export const getConfigurations = (params?: ListConfigurationsParams) => apiClient.get<Configuration[]>('/configurations', { params });
export const getConfigurationById = (id: number) => apiClient.get<Configuration>(`/configurations/${id}`);
export const createConfiguration = (data: ConfigurationPayload) => apiClient.post<Configuration>('/configurations', data);
// Corrected updateConfiguration to use PUT as per REST standards and backend implementation.
export const updateConfiguration = (id: number, data: ConfigurationUpdatePayload) => apiClient.put<Configuration>(`/configurations/${id}`, data);


export const getConfigurationDiffWithNacos = (id: number) => apiClient.get<{ diff: string; localContent: string; nacosContent: string; isDifferent: boolean; nacosError?: string }>(`/configurations/${id}/diff-nacos`);
export const fetchConfigurationFromNacos = (id: number, operator?: string) => apiClient.post<Configuration>(`/configurations/${id}/fetch-from-nacos`, null, { params: { operator } });


// --- Configuration History & Rollback ---
export const getConfigurationHistory = (configId: number) => apiClient.get<ConfigurationHistory[]>(`/configurations/${configId}/history`);
export const getConfigurationHistoryEntry = (historyId: number) => apiClient.get<ConfigurationHistory>(`/configurations/history/${historyId}`);
export const rollbackConfiguration = (configId: number, historyId: number, payload: RollbackPayload) => apiClient.post<Configuration>(`/configurations/${configId}/rollback/${historyId}`, payload);


// --- Publishing ---
export const publishConfiguration = (configId: number, data: PublishPayload) => apiClient.post<{ message: string; publishRecord: PublishRecord }>(`/configurations/${configId}/publish`, data);


// --- Publish Records ---
export interface ListPublishRecordsParams {
  configurationId?: number;
  nacosInstanceId?: number;
  status?: string;
}
export const getPublishRecords = (params?: ListPublishRecordsParams) => apiClient.get<PublishRecord[]>('/publish-records', { params });
export const getPublishRecordById = (id: number) => apiClient.get<PublishRecord>(`/publish-records/${id}`);


export default apiClient; // Export default for convenience if preferred in some places
