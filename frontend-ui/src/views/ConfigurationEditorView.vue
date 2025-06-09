<template>
  <div class="configuration-editor-view container">
    <el-page-header @back="goBack" :content="pageTitle" class="mb-20 page-header-fixed">
       <template #extra>
        <div class="page-header-actions">
          <el-button @click="fetchFromNacos" :loading="isFetching" icon="Refresh">Fetch from Nacos</el-button>
          <el-button @click="compareWithNacos" :loading="isComparing" icon="Files">Compare with Nacos</el-button>
          <el-button type="warning" @click="handlePublish" :disabled="!configData.ID" icon="Promotion">Publish to Nacos</el-button>
        </div>
      </template>
    </el-page-header>

    <el-card v-if="!isLoading">
      <el-form :model="configData" :rules="formRules" ref="configFormRef" label-width="120px" label-position="top">
        <el-row :gutter="20">
          <el-col :span="8">
            <el-form-item label="Data ID" prop="DataID">
              <el-input v-model="configData.DataID" :disabled="isEditMode" placeholder="e.g., com.example.service" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="Group" prop="Group">
              <el-input v-model="configData.Group" :disabled="isEditMode" placeholder="e.g., DEFAULT_GROUP" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="Nacos Instance" prop="NacosInstanceID">
               <el-select
                v-model="configData.NacosInstanceID"
                placeholder="Select Nacos Instance"
                class="full-width"
                :disabled="isEditMode"
                filterable
              >
                <el-option
                  v-for="instance in nacosInstances"
                  :key="instance.ID"
                  :label="`${instance.Description || instance.InstanceURL} (${instance.InstanceURL})`"
                  :value="instance.ID"
                />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="20">
           <el-col :span="8">
            <el-form-item label="Nacos Namespace ID" prop="NamespaceID">
              <el-input v-model="configData.NamespaceID" :disabled="isEditMode" placeholder="Leave blank for public" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="Content Type" prop="ContentType">
              <el-select v-model="configData.ContentType" placeholder="Select content type" class="full-width">
                <el-option label="Text" value="text"></el-option>
                <el-option label="JSON" value="json"></el-option>
                <el-option label="XML" value="xml"></el-option>
                <el-option label="YAML" value="yaml"></el-option>
                <el-option label="HTML" value="html"></el-option>
                <el-option label="Properties" value="properties"></el-option>
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>

        <el-form-item label="Description" prop="Description">
          <el-input type="textarea" v-model="configData.Description" placeholder="Optional: Describe this configuration" />
        </el-form-item>

        <el-form-item label="Configuration Content" prop="Content">
          <div class="editor-container">
            <codemirror
              v-model="configData.Content"
              placeholder="Enter configuration content here..."
              :style="{ height: '400px', width: '100%' }"
              :autofocus="true"
              :indent-with-tab="true"
              :tab-size="2"
              :extensions="codemirrorExtensions"
              @ready="handleEditorReady"
            />
          </div>
        </el-form-item>

        <el-form-item>
          <el-button type="primary" @click="saveToDB" :loading="isSaving">
            <el-icon><DocumentChecked /></el-icon> {{ isEditMode ? 'Save Changes to DB' : 'Create Configuration in DB' }}
          </el-button>
          <el-button @click="goBack">Cancel</el-button>
        </el-form-item>
      </el-form>
    </el-card>
     <el-skeleton :rows="10" animated v-else />

    <!-- Diff Dialog -->
    <el-dialog v-model="diffDialogVisible" title="Compare with Nacos" width="70%" top="5vh">
      <div v-if="isComparing" style="text-align: center;">Loading diff...</div>
      <div v-else-if="diffHtml" v-html="diffHtml" class="diff-output"></div>
      <div v-else>No differences found or unable to compare.</div>
      <template #footer>
        <el-button @click="diffDialogVisible = false">Close</el-button>
      </template>
    </el-dialog>

    <!-- Publish Dialog -->
    <el-dialog v-model="publishDialogVisible" title="Publish Configuration to Nacos" width="500px">
        <el-form :model="publishForm" label-position="top">
            <el-form-item label="Publish Type">
                <el-radio-group v-model="publishForm.publishType">
                    <el-radio label="FULL">Full Publish</el-radio>
                    <el-radio label="GRAYSCALE" disabled>Grayscale (Not Implemented)</el-radio>
                </el-radio-group>
            </el-form-item>
            <el-form-item label="Operator (Optional)">
                <el-input v-model="publishForm.operator" placeholder="Your name or system ID" />
            </el-form-item>
        </el-form>
        <template #footer>
            <el-button @click="publishDialogVisible = false">Cancel</el-button>
            <el-button type="primary" @click="confirmPublish" :loading="isPublishing">Confirm Publish</el-button>
        </template>
    </el-dialog>

  </div>
</template>

<script lang="ts" setup>
import { ref, computed, onMounted, watch, shallowRef } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { ElCard, ElForm, ElFormItem, ElInput, ElButton, ElIcon, ElSelect, ElOption, ElPageHeader, ElRow, ElCol, ElDialog, ElRadioGroup, ElRadio, ElMessage, ElSkeleton } from 'element-plus';
import { Plus, DocumentChecked, Refresh, Files, Promotion } from '@element-plus/icons-vue';
import { Codemirror } from 'vue-codemirror';
import { javascript } from '@codemirror/lang-javascript';
import { yaml } from '@codemirror/lang-yaml';
import { json } from '@codemirror/lang-json';
import { html } from '@codemirror/lang-html';
// import { properties } from '@codemirror/lang-properties'; // If it existed
import { oneDark } from '@codemirror/theme-one-dark'; // Example theme
import * as Diff2Html from 'diff2html'; // Correct import for ES modules
import 'diff2html/bundles/css/diff2html.min.css';
import * as Diff from 'diff'; // For generating diffs

import * as api from '@/services/api';
import type { Configuration, ConfigurationPayload, ConfigurationUpdatePayload, NacosInstance, PublishPayload } from '@/types';

const props = defineProps<{
  configId?: string | number; // From route params for edit mode
  mode: 'create' | 'edit';
}>();

const route = useRoute();
const router = useRouter();

const isLoading = ref(false);
const isSaving = ref(false);
const isFetching = ref(false);
const isComparing = ref(false);
const isPublishing = ref(false);

const configFormRef = ref<InstanceType<typeof ElForm>>();
const configData = ref<Partial<Configuration>>({
  DataID: '',
  Group: 'DEFAULT_GROUP',
  NamespaceID: route.query.namespaceId as string || '', // Pre-fill from query for create mode
  NacosInstanceID: route.query.nacosInstanceId ? parseInt(route.query.nacosInstanceId as string) : undefined,
  ContentType: 'text',
  Content: '',
  Description: '',
});

const nacosInstances = ref<NacosInstance[]>([]);

const formRules = {
  DataID: [{ required: true, message: 'Data ID is required', trigger: 'blur' }],
  Group: [{ required: true, message: 'Group is required', trigger: 'blur' }],
  NacosInstanceID: [{ required: true, message: 'Nacos Instance is required', trigger: 'change' }],
  ContentType: [{ required: true, message: 'Content Type is required', trigger: 'change' }],
  Content: [{ required: true, message: 'Content is required', trigger: 'blur' }],
};

const isEditMode = computed(() => props.mode === 'edit');
const pageTitle = computed(() => (isEditMode.value ? `Edit Configuration: ${configData.value.DataID || ''}` : 'Create New Configuration'));

// Codemirror setup
const editorView = shallowRef();
const handleEditorReady = (payload: any) => {
  editorView.value = payload.view;
};

const codemirrorExtensions = computed(() => {
  const langExtMap: { [key: string]: () => any } = {
    json: json,
    yaml: yaml,
    javascript: javascript,
    html: html,
    // properties: () => import('@codemirror/legacy-modes/mode/properties').then(m => m.properties), // Example for legacy/other modes
    text: () => [], // No specific language for plain text
  };
  const selectedLangExtension = langExtMap[configData.value.ContentType || 'text'] || (() => []);
  return [selectedLangExtension(), oneDark]; // Add theme
});


// Diff Dialog
const diffDialogVisible = ref(false);
const diffHtml = ref('');


// Publish Dialog
const publishDialogVisible = ref(false);
const publishForm = ref<PublishPayload & { operator?: string }>({
    publishType: 'FULL',
    operator: 'frontend-user', // Default operator
});


const fetchNacosInstancesForSelect = async () => {
  try {
    const response = await api.getNacosInstances();
    nacosInstances.value = response.data;
  } catch (error) {
    ElMessage.error('Failed to load Nacos instances for selection.');
  }
};


const loadConfiguration = async () => {
  if (!isEditMode.value || !props.configId) return;
  isLoading.value = true;
  try {
    const response = await api.getConfigurationById(Number(props.configId));
    configData.value = response.data;
  } catch (error) {
    console.error('Failed to load configuration:', error);
    ElMessage.error('Failed to load configuration details.');
    router.push('/configurations'); // Redirect if load fails
  } finally {
    isLoading.value = false;
  }
};

const saveToDB = async () => {
  if (!configFormRef.value) return;
  await configFormRef.value.validate(async (valid) => {
    if (valid) {
      isSaving.value = true;
      try {
        let response;
        const payload = { ...configData.value };
        // Ensure NacosInstanceID is a number
        if (payload.NacosInstanceID && typeof payload.NacosInstanceID === 'string') {
            payload.NacosInstanceID = parseInt(payload.NacosInstanceID);
        }

        if (isEditMode.value && configData.value.ID) {
          const updatePayload: ConfigurationUpdatePayload = {
            content: payload.Content,
            contentType: payload.ContentType,
            description: payload.Description,
            // operator: 'frontend-user' // Add operator if needed
          };
          response = await api.updateConfiguration(configData.value.ID, updatePayload);
          ElMessage.success('Configuration updated successfully in DB!');
        } else {
          const createPayload: ConfigurationPayload = {
            nacosInstanceId: payload.NacosInstanceID!,
            dataId: payload.DataID!,
            groupName: payload.Group!,
            namespaceId: payload.NamespaceID,
            content: payload.Content!,
            contentType: payload.ContentType,
            description: payload.Description,
          };
          response = await api.createConfiguration(createPayload);
          ElMessage.success('Configuration created successfully in DB!');
          // If creating, redirect to edit mode for the new config
          router.replace({ name: 'ConfigurationEdit', params: { id: response.data.ID.toString() } });
        }
        configData.value = response.data; // Update local data with response (e.g. new ID, timestamps)
      } catch (error) {
        console.error('Failed to save configuration:', error);
        // Error already handled by global interceptor
      } finally {
        isSaving.value = false;
      }
    } else {
      ElMessage.error('Please correct the form errors.');
      return false;
    }
  });
};

const fetchFromNacos = async () => {
  if (!configData.value.ID) {
    ElMessage.warning('Configuration must be saved locally first.');
    return;
  }
  isFetching.value = true;
  try {
    const response = await api.fetchConfigurationFromNacos(configData.value.ID);
    configData.value.Content = response.data.Content; // Update content
    configData.value.MD5 = response.data.MD5;
    configData.value.LastSyncTime = response.data.LastSyncTime;
    ElMessage.success('Configuration fetched from Nacos and updated locally.');
  } catch (error) {
    console.error('Failed to fetch from Nacos:', error);
  } finally {
    isFetching.value = false;
  }
};

const compareWithNacos = async () => {
  if (!configData.value.ID) {
    ElMessage.warning('Configuration must be saved locally first to compare.');
    return;
  }
  isComparing.value = true;
  diffDialogVisible.value = true;
  diffHtml.value = ''; // Clear previous diff

  try {
    const response = await api.getConfigurationDiffWithNacos(configData.value.ID);
    const diffData = response.data;

    if (!diffData.isDifferent) {
      diffHtml.value = '<p>Contents are identical.</p>';
    } else if (diffData.nacosError) {
        diffHtml.value = `<p>Error from Nacos: ${diffData.nacosError}</p>
                          <p>Local Content:</p><pre>${diffData.localContent || ''}</pre>`;
    } else {
      // Use Diff.createPatch or Diff.structuredPatch for more structured diff data
      const patch = Diff.createPatch(
        `Config: ${configData.value.DataID} Group: ${configData.value.Group}`, // oldFileName
        `Config: ${configData.value.DataID} Group: ${configData.value.Group}`, // newFileName
        diffData.localContent || '',  // oldStr
        diffData.nacosContent || '', // newStr
        'Local DB', // oldHeader
        'Nacos Server'  // newHeader
      );
      diffHtml.value = Diff2Html.html(patch, {
        drawFileList: false,
        outputFormat: 'side-by-side', // or 'line-by-line'
        matching: 'lines',
      });
    }
  } catch (error) {
    console.error('Failed to compare with Nacos:', error);
    diffHtml.value = '<p>Error occurred during comparison.</p>';
  } finally {
    isComparing.value = false;
  }
};


const handlePublish = () => {
    if (!configData.value.ID) {
        ElMessage.error('Configuration must be saved before publishing.');
        return;
    }
    publishDialogVisible.value = true;
    // Reset publish form if needed
    publishForm.value.operator = 'frontend-user';
};

const confirmPublish = async () => {
    if (!configData.value.ID) return;
    isPublishing.value = true;
    try {
        await api.publishConfiguration(configData.value.ID, {
            publishType: publishForm.value.publishType,
            operator: publishForm.value.operator,
        });
        ElMessage.success('Configuration published successfully!');
        publishDialogVisible.value = false;
        loadConfiguration(); // Refresh config data (might have new sync status)
    } catch (error) {
        console.error('Failed to publish configuration:', error);
        // Error already handled by global interceptor
    } finally {
        isPublishing.value = false;
    }
};


const goBack = () => {
  router.back(); // Or router.push('/configurations');
};

onMounted(() => {
  fetchNacosInstancesForSelect();
  if (isEditMode.value) {
    loadConfiguration();
  } else {
    // Pre-fill from query parameters if any for create mode (already done in ref definition)
    if (!configData.value.NacosInstanceID && nacosInstances.value.length > 0) {
        // configData.value.NacosInstanceID = nacosInstances.value[0].ID; // Default if needed
    }
  }
});

// Watch for content type change to update CodeMirror language (if needed, though extensions prop is computed)
watch(() => configData.value.ContentType, (newType, oldType) => {
  // console.log(`Content type changed from ${oldType} to ${newType}`);
  // Extensions are reactive, so CodeMirror should update automatically.
});

</script>

<style lang="scss" scoped>
.configuration-editor-view {
  .page-header-fixed {
    // If you want a sticky header, you might need more CSS here
    // position: sticky;
    // top: 0; // Adjust based on your main layout header height if any
    // background-color: #fff; // Or your page background
    // z-index: 100;
    // padding-bottom: 10px;
    // border-bottom: 1px solid var(--el-border-color-light);
  }
  .editor-container {
    border: 1px solid var(--el-border-color);
    border-radius: 4px;
    overflow: hidden; // Ensures border radius is applied to CodeMirror
    width: 100%;
  }
}
.diff-output {
  max-height: 60vh;
  overflow-y: auto;
  pre {
    white-space: pre-wrap;
    word-wrap: break-word;
  }
}
.page-header-actions {
    display: flex;
    gap: 10px; // Adds space between buttons in the header
}

</style>
<style>
/* Styles for Diff2Html - not scoped */
.d2h-file-header {
  display: none; /* Hide the default file header from diff2html if not needed */
}
.d2h-wrapper {
  border: 1px solid #ddd;
  border-radius: 3px;
  margin-top: 10px;
}
</style>
