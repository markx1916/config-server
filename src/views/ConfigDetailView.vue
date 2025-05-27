<template>
  <div class="config-detail-view">
    <el-page-header @back="goBack" :content="pageTitle" style="margin-bottom: 20px;"></el-page-header>

    <el-tabs v-model="activeTab" type="border-card" v-if="!configStore.loading || (configStore.loading && configStore.currentConfiguration)">
      <!-- Configuration Editor Tab -->
      <el-tab-pane label="Configuration" name="editor">
        <el-form
          v-if="configForm"
          :model="configForm"
          :rules="rules"
          ref="configFormRef"
          label-width="120px"
          style="max-width: 800px; margin-top: 20px;"
          :disabled="configStore.loading || configStore.publishing"
        >
          <el-form-item label="Data ID" prop="data_id">
            <el-input v-model="configForm.data_id" placeholder="e.g., com.example.app.feature" :disabled="!isNewConfig"></el-input>
          </el-form-item>
          <el-form-item label="Group" prop="group_name">
            <el-input v-model="configForm.group_name" placeholder="e.g., DEFAULT_GROUP" :disabled="!isNewConfig"></el-input>
          </el-form-item>
          <el-form-item label="Nacos Instance" v-if="isNewConfig && newConfigNacosInstanceInfo">
             <el-input :value="`${newConfigNacosInstanceInfo.name} (${newConfigNacosInstanceInfo.server_addr})`" disabled></el-input>
          </el-form-item>
          <el-form-item label="Format" prop="format">
            <el-select v-model="configForm.format" placeholder="Select format">
              <el-option label="Text" value="text"></el-option>
              <el-option label="JSON" value="json"></el-option>
              <el-option label="XML" value="xml"></el-option>
              <el-option label="YAML" value="yaml"></el-option>
              <el-option label="Properties" value="properties"></el-option>
              <el-option label="HTML" value="html"></el-option>
            </el-select>
          </el-form-item>
          <el-form-item label="Description" prop="description">
            <el-input v-model="configForm.description" type="textarea" :rows="2" placeholder="Optional description"></el-input>
          </el-form-item>
          <el-form-item label="Content" prop="content">
            <el-input
              v-model="configForm.content"
              type="textarea"
              :autosize="{ minRows: 10, maxRows: 20 }"
              placeholder="Enter configuration content"
              @input="onContentChange"
            ></el-input>
          </el-form-item>

          <el-form-item v-if="!isNewConfig && showDiffView" label="Changes (Diff)">
            <div class="diff-view-container">
               <pre v-html="diffOutputHtml" class="diff-output"></pre>
            </div>
             <el-button @click="fetchAndShowDiff(configStore.currentConfiguration!.version -1)" size="small" v-if="configStore.currentConfiguration && configStore.currentConfiguration.version > 1">
                Compare with Previous Version (v{{ configStore.currentConfiguration.version -1 }})
            </el-button>
          </el-form-item>

          <el-form-item>
            <el-button type="primary" @click="handleSaveConfig" :loading="configStore.loading || configStore.publishing">
              {{ isNewConfig ? 'Create Configuration' : 'Save Changes' }}
            </el-button>
            <el-button @click="goBack" :disabled="configStore.publishing">Cancel</el-button>
          </el-form-item>

          <el-divider v-if="!isNewConfig && configStore.currentConfiguration">Publish to Nacos</el-divider>
           <el-form-item v-if="!isNewConfig && configStore.currentConfiguration" label="Publish">
             <el-input v-model="betaIps" placeholder="Beta IPs (comma-separated, for Gray publish)" style="width: calc(100% - 210px); margin-right:10px;"></el-input>
             <el-button type="warning" @click="handlePublish('gray')" :loading="configStore.publishing" :disabled="!configStore.currentConfiguration">
                <el-icon><Promotion /></el-icon> Gray Publish
            </el-button>
            <el-button type="success" @click="handlePublish('full')" :loading="configStore.publishing" :disabled="!configStore.currentConfiguration">
                <el-icon><Promotion /></el-icon> Full Publish
            </el-button>
          </el-form-item>
        </el-form>>
        <el-skeleton :rows="10" animated v-if="configStore.loading && !configStore.currentConfiguration && !isNewConfig" />
        <el-empty description="Configuration data not loaded or not found." v-if="!configStore.loading && !configStore.currentConfiguration && !isNewConfig"></el-empty>
      </el-tab-pane>

      <!-- Save History Tab -->
      <el-tab-pane label="Save History" name="saveHistory" :disabled="isNewConfig">
        <ConfigSaveHistoryTab :config-id="configIdNumber" v-if="activeTab === 'saveHistory' && !isNewConfig"/>
      </el-tab-pane>

      <!-- Deployment History Tab -->
      <el-tab-pane label="Deployment History" name="deploymentHistory" :disabled="isNewConfig">
        <ConfigDeploymentHistoryTab :config-id-prop="configIdNumber" v-if="activeTab === 'deploymentHistory' && !isNewConfig"/>
      </el-tab-pane>
    </el-tabs>
     <el-empty description="Loading configuration details..." v-if="configStore.loading && !configStore.currentConfiguration && !isNewConfig && activeTab !== 'editor'"></el-empty>


  </div>
</template>

<script lang="ts" setup>
import { ref, reactive, onMounted, computed, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useConfigStore } from '../store/configStore';
import { useNacosStore } from '../store/nacosStore';
import type { Configuration, CreateConfigurationPayload, UpdateConfigurationPayload, NacosInstance as NacosInstanceType } from '../services/api';
import { ElMessage, FormInstance, FormRules } from 'element-plus';
import { Promotion } from '@element-plus/icons-vue';
import ConfigSaveHistoryTab from '../components/ConfigSaveHistoryTab.vue';
import ConfigDeploymentHistoryTab from '../components/ConfigDeploymentHistoryTab.vue'; // Adjusted path
import { DiffMatchPatch, DIFF_DELETE, DIFF_INSERT, DIFF_EQUAL } from 'diff-match-patch';

const route = useRoute();
const router = useRouter();
const configStore = useConfigStore();
const nacosStore = useNacosStore();

const activeTab = ref('editor');
const configFormRef = ref<FormInstance>();

const initialFormState = {
  data_id: '',
  group_name: 'DEFAULT_GROUP',
  format: 'text' as Configuration['format'],
  content: '',
  description: '',
  nacos_instance_id: null as number | null,
};

const configForm = reactive({ ...initialFormState });
const originalContentForDiff = ref('');
const diffOutputHtml = ref('');
const showDiffView = ref(false);
const betaIps = ref('');


const isNewConfig = computed(() => route.params.id === 'new');
const configIdNumber = computed(() => isNewConfig.value ? null : Number(route.params.id));

const newConfigNacosInstanceInfo = ref<NacosInstanceType | null>(null);


const pageTitle = computed(() => {
  if (isNewConfig.value) return 'Create New Configuration';
  if (configStore.currentConfiguration) {
    return `View/Edit: ${configStore.currentConfiguration.data_id} (Group: ${configStore.currentConfiguration.group_name})`;
  }
  return 'Configuration Detail';
});

const rules = reactive<FormRules>({
  data_id: [{ required: true, message: 'Please input Data ID', trigger: 'blur' }],
  group_name: [{ required: true, message: 'Please input Group', trigger: 'blur' }],
  format: [{ required: true, message: 'Please select format', trigger: 'change' }],
  content: [{ required: true, message: 'Please input configuration content', trigger: 'blur' }],
});

const loadNacosInstanceForNewConfig = async (instanceId: number) => {
    if (nacosStore.instances.length === 0) {
        await nacosStore.fetchInstances();
    }
    newConfigNacosInstanceInfo.value = nacosStore.instances.find(inst => inst.id === instanceId) || null;
    if (!newConfigNacosInstanceInfo.value) {
        ElMessage.error(`Nacos Instance with ID ${instanceId} not found. Cannot create configuration.`);
        router.push({ name: 'config-list' });
    }
};


onMounted(async () => {
  configStore.clearCurrentConfiguration(); // Clear any previous state
  if (isNewConfig.value) {
    Object.assign(configForm, initialFormState);
    const nacosInstanceIdFromQuery = Number(route.query.nacosInstanceId);
    if (!nacosInstanceIdFromQuery) {
        ElMessage.error('Nacos Instance ID is required to create a new configuration.');
        router.push({ name: 'config-list' }); // Redirect if no instance ID
        return;
    }
    configForm.nacos_instance_id = nacosInstanceIdFromQuery;
    await loadNacosInstanceForNewConfig(nacosInstanceIdFromQuery);

  } else if (configIdNumber.value) {
    try {
      const loadedConfig = await configStore.fetchConfigurationById(configIdNumber.value);
      if (loadedConfig) {
        configForm.data_id = loadedConfig.data_id;
        configForm.group_name = loadedConfig.group_name;
        configForm.format = loadedConfig.format;
        configForm.content = loadedConfig.content;
        configForm.description = loadedConfig.description || '';
        originalContentForDiff.value = loadedConfig.content;
      }
    } catch (error) {
      // Error already handled by store
      router.push({ name: 'config-list' }); // Redirect if config not found or error
    }
  }
});

watch(() => configStore.currentConfiguration, (newConfig) => {
  if (newConfig && !isNewConfig.value) {
    configForm.data_id = newConfig.data_id;
    configForm.group_name = newConfig.group_name;
    configForm.format = newConfig.format;
    configForm.content = newConfig.content;
    configForm.description = newConfig.description || '';
    originalContentForDiff.value = newConfig.content;
    showDiffView.value = false; // Reset diff view when config changes (e.g., after rollback)
    diffOutputHtml.value = '';
  }
}, { immediate: true });


const onContentChange = () => {
    if (isNewConfig.value || !configStore.currentConfiguration) {
        showDiffView.value = false;
        return;
    }
    if (configForm.content !== originalContentForDiff.value) {
        generateDiff(originalContentForDiff.value, configForm.content);
        showDiffView.value = true;
    } else {
        showDiffView.value = false;
        diffOutputHtml.value = '';
    }
};

const generateDiff = (text1: string, text2: string) => {
    const dmp = new DiffMatchPatch();
    const diffs = dmp.diff_main(text1, text2);
    dmp.diff_cleanupSemantic(diffs);
    diffOutputHtml.value = dmp.diff_prettyHtml(diffs);
};

const fetchAndShowDiff = async (compareToVersion: number) => {
    if (!configIdNumber.value) return;
    try {
        const diffData = await configStore.fetchDiff(configIdNumber.value, compareToVersion);
        if (diffData) {
            // Use a more structured diff display if possible, or just the raw output
            // For this example, we'll use the diff_prettyHtml from diff-match-patch
            const dmp = new DiffMatchPatch();
            const diffs = dmp.diff_main(diffData.compared_version_content, diffData.current_version_content);
            dmp.diff_cleanupSemantic(diffs);
            diffOutputHtml.value = dmp.diff_prettyHtml(diffs);
            showDiffView.value = true;
            ElMessage.success(`Diff loaded against version ${compareToVersion}.`);
        }
    } catch (error) {
        ElMessage.error('Failed to load diff.');
    }
};


const handleSaveConfig = async () => {
  if (!configFormRef.value) return;
  await configFormRef.value.validate(async (valid) => {
    if (valid) {
      try {
        let savedConfig: Configuration | null = null;
        if (isNewConfig.value) {
          if (!configForm.nacos_instance_id) {
            ElMessage.error("Nacos instance ID is missing. Cannot create configuration.");
            return;
          }
          const payload: CreateConfigurationPayload = {
            nacos_instance_id: configForm.nacos_instance_id,
            data_id: configForm.data_id,
            group_name: configForm.group_name,
            content: configForm.content,
            format: configForm.format,
            description: configForm.description,
          };
          savedConfig = await configStore.createConfiguration(payload);
          if (savedConfig) {
            // After successful creation, navigate to the edit/view mode for this new config
            router.replace({ name: 'config-detail', params: { id: savedConfig.id.toString() } });
          }
        } else if (configIdNumber.value) {
          const payload: UpdateConfigurationPayload = {
            content: configForm.content,
            format: configForm.format,
            description: configForm.description,
          };
          savedConfig = await configStore.updateConfiguration(configIdNumber.value, payload);
          if (savedConfig) {
            originalContentForDiff.value = savedConfig.content; // Update original content after save
            showDiffView.value = false; // Hide diff view after saving
            diffOutputHtml.value = '';
            // Trigger Feishu notification placeholder
            console.log("Feishu notification would be sent here for config update:", savedConfig.id);
            ElMessage.info("Placeholder: Feishu notification sent!");

          }
        }
      } catch (error) {
        // Error already handled by store
      }
    } else {
      ElMessage.error('Please correct the errors in the form.');
    }
  });
};

const handlePublish = async (type: 'gray' | 'full') => {
  if (!configIdNumber.value) return;
  
  let confirmMessage = `Are you sure you want to perform a ${type.toUpperCase()} publish for this configuration?`;
  if (type === 'gray' && !betaIps.value) {
    ElMessage.warning('For Gray Publish, please specify Beta IPs.');
    return;
  }
   if (type === 'gray' && betaIps.value) {
    confirmMessage += `\nTarget Beta IPs: ${betaIps.value}`;
  }


  ElMessageBox.confirm(confirmMessage, 'Confirm Publish', {
    confirmButtonText: 'Publish',
    cancelButtonText: 'Cancel',
    type: 'info',
  }).then(async () => {
    try {
      await configStore.publishConfiguration(configIdNumber.value!, type, type === 'gray' ? betaIps.value : undefined);
      // The store action handles success/error messages.
      // Optionally, refresh deployment history or other relevant data.
      if (activeTab.value === 'deploymentHistory') {
        // If deployment history tab is active, refresh it
        // This requires the component to expose a refresh method or watch props
      }
    } catch (error) {
      // Error already handled by store
    }
  }).catch(() => {
    ElMessage.info('Publish canceled.');
  });
};


const goBack = () => {
  router.push({ name: 'config-list' });
};

// When tab changes, if it's history, fetch data if not already loaded
watch(activeTab, (newTab) => {
    if (isNewConfig.value) return; // No history for new configs
    if (newTab === 'saveHistory' && configIdNumber.value) {
        configStore.fetchSaveHistory(configIdNumber.value);
    } else if (newTab === 'deploymentHistory' && configIdNumber.value) {
        configStore.fetchDeploymentHistory(configIdNumber.value);
    }
});

</script>

<style scoped>
.config-detail-view {
  padding: 20px;
}
.diff-view-container {
  border: 1px solid #dcdfe6;
  padding: 10px;
  border-radius: 4px;
  background-color: #f9f9f9;
  max-height: 300px;
  overflow-y: auto;
  width: 100%;
}
.diff-output {
  white-space: pre-wrap;
  font-family: 'Courier New', Courier, monospace;
  font-size: 0.9em;
}
.diff-output ins {
  background-color: #e6ffed;
  text-decoration: none;
}
.diff-output del {
  background-color: #ffebe9;
  text-decoration: none;
}
</style>
