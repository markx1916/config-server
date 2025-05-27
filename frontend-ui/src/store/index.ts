import { defineStore } from 'pinia';

// Example: Store for managing the currently selected Nacos instance and its namespaces
export const useNacosStore = defineStore('nacos', {
  state: () => ({
    selectedInstanceId: null as number | null,
    selectedInstanceUrl: '', // For display purposes
    availableNamespaces: [] as { namespaceId: string; namespaceShowName: string }[],
    currentNamespaceId: '' as string | null, // Selected namespace for configuration listing
  }),
  getters: {
    hasSelectedInstance: (state) => state.selectedInstanceId !== null,
  },
  actions: {
    selectInstance(instance: { id: number; instanceUrl: string }) {
      this.selectedInstanceId = instance.id;
      this.selectedInstanceUrl = instance.instanceUrl;
      // Reset namespaces and current namespace when instance changes
      this.availableNamespaces = [];
      this.currentNamespaceId = null; 
      // TODO: Fetch namespaces for this instance from backend
    },
    setNamespaces(namespaces: { namespaceId: string; namespaceShowName: string }[]) {
      this.availableNamespaces = namespaces;
      // Optionally select the first namespace or a default one
      if (namespaces.length > 0) {
        this.currentNamespaceId = namespaces[0].namespaceId; // Default to the first one
      } else {
        this.currentNamespaceId = ''; // Default to public if no custom namespaces
      }
    },
    setCurrentNamespace(namespaceId: string) {
      this.currentNamespaceId = namespaceId;
    },
    clearSelection() {
      this.selectedInstanceId = null;
      this.selectedInstanceUrl = '';
      this.availableNamespaces = [];
      this.currentNamespaceId = null;
    },
  },
});

// Example: Store for managing the configuration editor state
export const useConfigurationStore = defineStore('configuration', {
    state: () => ({
        editingConfig: null as any | null, // Replace 'any' with a proper Configuration type
        isLoading: false,
        isDiffVisible: false,
        diffContent: { local: '', remote: '' },
    }),
    actions: {
        setEditingConfig(config: any) {
            this.editingConfig = config;
        },
        clearEditingConfig() {
            this.editingConfig = null;
        },
        setLoading(loading: boolean) {
            this.isLoading = loading;
        },
        showDiff(localContent: string, remoteContent: string) {
            this.diffContent = { local: localContent, remote: remoteContent };
            this.isDiffVisible = true;
        },
        hideDiff() {
            this.isDiffVisible = false;
        }
    }
});

// You can define more stores here as needed.
