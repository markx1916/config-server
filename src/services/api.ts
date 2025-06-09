import axios from 'axios';
import { ElMessage } from 'element-plus';

// Base URL for the API.
// In development, Vite's proxy will handle this.
// In production, this would be the actual API endpoint.
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api';

const apiClient = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Interceptors for request
apiClient.interceptors.request.use(
  (config) => {
    // TODO: Add token or other auth headers if needed
    // const token = localStorage.getItem('token');
    // if (token) {
    //   config.headers.Authorization = `Bearer ${token}`;
    // }
    return config;
  },
  (error) => {
    ElMessage({
      message: `Request Error: ${error.message}`,
      type: 'error',
      duration: 5 * 1000,
    });
    return Promise.reject(error);
  }
);

// Interceptors for response
apiClient.interceptors.response.use(
  (response) => {
    // Any status code that lie within the range of 2xx cause this function to trigger
    return response;
  },
  (error) => {
    // Any status codes that falls outside the range of 2xx cause this function to trigger
    let errorMessage = 'An unknown error occurred';
    if (error.response) {
      // The request was made and the server responded with a status code
      // that falls out of the range of 2xx
      console.error('API Error Response:', error.response);
      errorMessage = error.response.data?.message || error.response.data?.error || `Server Error: ${error.response.status}`;
      if (error.response.status === 401) {
        // TODO: Handle unauthorized access, e.g., redirect to login
        console.warn('Unauthorized access detected.');
      }
    } else if (error.request) {
      // The request was made but no response was received
      console.error('API No Response:', error.request);
      errorMessage = 'No response from server. Please check your network connection.';
    } else {
      // Something happened in setting up the request that triggered an Error
      console.error('API Request Setup Error:', error.message);
      errorMessage = `Request Setup Error: ${error.message}`;
    }
    ElMessage({
      message: errorMessage,
      type: 'error',
      duration: 5 * 1000,
    });
    return Promise.reject(error);
  }
);

export default apiClient;

// Define Nacos Instance types (can be moved to a types/interfaces file)
export interface NacosInstance {
  id: number;
  name: string;
  server_addr: string;
  namespace_id?: string;
  username?: string;
  // Password is not typically sent from backend for list/get
  created_at: string;
  updated_at: string;
}

export interface CreateNacosInstancePayload {
  name: string;
  server_addr: string;
  namespace_id?: string;
  username?: string;
  password?: string;
}

export interface UpdateNacosInstancePayload {
  name?: string;
  server_addr?: string;
  namespace_id?: string;
  username?: string;
  password?: string;
}

// Nacos Instance API calls
export const getNacosInstances = () => apiClient.get<NacosInstance[]>('/nacos/instances');
export const createNacosInstance = (data: CreateNacosInstancePayload) => apiClient.post<NacosInstance>('/nacos/instances', data);
export const getNacosInstanceById = (id: number) => apiClient.get<NacosInstance>(`/nacos/instances/${id}`);
export const updateNacosInstance = (id: number, data: UpdateNacosInstancePayload) => apiClient.put<NacosInstance>(`/nacos/instances/${id}`, data);
export const deleteNacosInstance = (id: number) => apiClient.delete(`/nacos/instances/${id}`);


// Configuration types
export interface Configuration {
  id: number;
  nacos_instance_id: number;
  data_id: string;
  group_name: string;
  content: string;
  format: 'text' | 'json' | 'yaml' | 'properties' | 'xml' | 'html';
  description?: string;
  version: number;
  created_at: string;
  updated_at: string;
}

export interface CreateConfigurationPayload {
  nacos_instance_id: number;
  data_id: string;
  group_name: string;
  content: string;
  format: 'text' | 'json' | 'yaml' | 'properties' | 'xml' | 'html';
  description?: string;
}

export interface UpdateConfigurationPayload {
  content?: string;
  format?: 'text' | 'json' | 'yaml' | 'properties' | 'xml' | 'html';
  description?: string;
}

export interface ConfigurationHistory {
  id: number;
  configuration_id: number;
  content: string;
  format: string;
  version: number;
  saved_by?: string;
  saved_at: string;
}

export interface DeploymentHistory {
  id: number;
  configuration_id: number;
  configuration_version: number;
  nacos_instance_id: number;
  deployment_type: string; // "GRAY", "FULL"
  status: string; // "SUCCESS", "FAILURE", "IN_PROGRESS"
  message?: string;
  deployed_by?: string;
  deployed_at: string;
}

export interface DiffResponse {
  current_version_content: string;
  compared_version: number;
  compared_version_content: string;
  diff_output: string;
}

// Configuration API calls
export const listConfigurations = (nacosInstanceId: number, dataId?: string, group?: string) => {
  return apiClient.get<Configuration[]>('/nacos/configs', {
    params: { nacos_instance_id: nacosInstanceId, data_id: dataId, group: group },
  });
};
export const createConfiguration = (data: CreateConfigurationPayload) => apiClient.post<Configuration>('/nacos/configs', data);
export const getConfigurationById = (id: number) => apiClient.get<Configuration>(`/nacos/configs/${id}`);
export const updateConfiguration = (id: number, data: UpdateConfigurationPayload) => apiClient.put<Configuration>(`/nacos/configs/${id}`, data);
export const getConfigurationDiff = (id: number, compareToVersion?: number) => {
  return apiClient.get<DiffResponse>(`/nacos/configs/${id}/diff`, {
    params: { compare_to_version: compareToVersion },
  });
};
export const publishConfiguration = (id: number, type: 'gray' | 'full', betaIps?: string) => {
  const params: { beta_ips?: string } = {};
  if (type === 'gray' && betaIps) {
    params.beta_ips = betaIps;
  }
  return apiClient.post<{status: string; message: string }>(`/nacos/configs/${id}/publish/${type}`, null, { params });
};
export const getConfigurationSaveHistory = (id: number) => apiClient.get<ConfigurationHistory[]>(`/nacos/configs/${id}/history`);
export const rollbackConfiguration = (id: number, historyId: number) => apiClient.post<Configuration>(`/nacos/configs/${id}/rollback/${historyId}`);
export const getConfigurationDeploymentHistory = (id: number) => apiClient.get<DeploymentHistory[]>(`/nacos/configs/${id}/deployments`);
