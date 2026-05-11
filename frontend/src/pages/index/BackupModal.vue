<script setup>
import { ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { message as antMessage } from 'ant-design-vue';
import { DownloadOutlined, UploadOutlined, ArrowLeftOutlined } from '@ant-design/icons-vue';
import { HttpUtil, PromiseUtil } from '@/utils';
import axios from 'axios';

const { t } = useI18n();
const password = ref('');
const showPasswordPrompt = ref(false);

defineProps({
  open: { type: Boolean, default: false },
  basePath: { type: String, default: '' },
});

const emit = defineEmits(['update:open', 'busy']);

function close() {
  showPasswordPrompt.value = false;
  password.value = '';
  emit('update:open', false);
}

function promptExport() {
  showPasswordPrompt.value = true;
}

async function confirmExport() {
  if (!password.value) return;

  emit('busy', { busy: true, tip: t('pages.index.exportDatabase') + '…' });

  try {
    const resp = await axios.post(
      window.X_UI_BASE_PATH + 'panel/api/server/getDb',
      { password: password.value },
      {
        responseType: 'blob',
        headers: { 'Content-Type': 'application/json' },
        transformRequest: [],
      }
    );

    if (resp.data instanceof Blob && resp.data.type !== 'application/json') {
      const url = URL.createObjectURL(resp.data);
      const link = document.createElement('a');
      link.href = url;
      link.download = 'x-ui.db';
      link.click();
      URL.revokeObjectURL(url);
      link.remove();
      close();
    } else {
      const text = await resp.data.text();
      try {
        const json = JSON.parse(text);
        if (!json.success) {
          const msg = json.msg || t('pages.index.importDatabaseError');
          antMessage.error(msg);
        }
      } catch {
        /* not JSON, ignore */
      }
    }
  } catch (err) {
    console.error('Export failed:', err);
    antMessage.error(err.response?.data?.msg || t('pages.index.importDatabaseError'));
  } finally {
    emit('busy', { busy: false });
    showPasswordPrompt.value = false;
    password.value = '';
  }
}

function cancelExport() {
  showPasswordPrompt.value = false;
  password.value = '';
}

function importDb() {
  const fileInput = document.createElement('input');
  fileInput.type = 'file';
  fileInput.accept = '.db';
  fileInput.addEventListener('change', async (e) => {
    const dbFile = e.target.files?.[0];
    if (!dbFile) return;

    const formData = new FormData();
    formData.append('db', dbFile);

    close();
    emit('busy', { busy: true, tip: t('pages.index.importDatabase') + '…' });

    const upload = await HttpUtil.post('/panel/api/server/importDB', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
    if (!upload?.success) {
      emit('busy', { busy: false });
      return;
    }

    emit('busy', { busy: true, tip: t('pages.settings.restartPanel') + '…' });
    const restart = await HttpUtil.post('/panel/setting/restartPanel');
    if (restart?.success) {
      await PromiseUtil.sleep(5000);
      window.location.reload();
    } else {
      emit('busy', { busy: false });
    }
  });
  fileInput.click();
}
</script>

<template>
  <a-modal :open="open" :title="t('pages.index.backupTitle')" :closable="true" :footer="null" @cancel="close">
    <template v-if="!showPasswordPrompt">
      <a-list bordered class="backup-list">
        <a-list-item class="backup-item">
          <a-list-item-meta>
            <template #title>{{ t('pages.index.exportDatabase') }}</template>
            <template #description>{{ t('pages.index.exportDatabaseDesc') }}</template>
          </a-list-item-meta>
          <a-button type="primary" @click="promptExport">
            <template #icon>
              <DownloadOutlined />
            </template>
          </a-button>
        </a-list-item>

        <a-list-item class="backup-item">
          <a-list-item-meta>
            <template #title>{{ t('pages.index.importDatabase') }}</template>
            <template #description>{{ t('pages.index.importDatabaseDesc') }}</template>
          </a-list-item-meta>
          <a-button type="primary" @click="importDb">
            <template #icon>
              <UploadOutlined />
            </template>
          </a-button>
        </a-list-item>
      </a-list>
    </template>

    <template v-else>
      <div class="password-prompt">
        <a-button type="text" class="back-btn" @click="cancelExport">
          <ArrowLeftOutlined />
        </a-button>
        <p class="prompt-text">{{ t('pages.index.exportDatabase') }}</p>
        <a-input-password
          v-model:value="password"
          :placeholder="t('password')"
          @press-enter="confirmExport"
          autofocus
        />
        <div class="prompt-actions">
          <a-button @click="cancelExport">{{ t('cancel') }}</a-button>
          <a-button type="primary" :disabled="!password" @click="confirmExport">
            {{ t('confirm') }}
          </a-button>
        </div>
      </div>
    </template>
  </a-modal>
</template>

<style scoped>
.backup-list {
  width: 100%;
}

.backup-item {
  display: flex;
  align-items: center;
  gap: 16px;
}

.password-prompt {
  display: flex;
  flex-direction: column;
  gap: 16px;
  position: relative;
}

.back-btn {
  position: absolute;
  top: -8px;
  left: -8px;
}

.prompt-text {
  margin: 0;
  font-size: 14px;
  font-weight: 500;
  text-align: center;
}

.prompt-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
