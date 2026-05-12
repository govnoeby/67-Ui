<script setup>
import { onMounted, ref } from 'vue';
import { HttpUtil } from '@/utils';

const logs = ref([]);
const total = ref(0);
const loading = ref(false);
const page = ref(1);
const pageSize = ref(50);

async function fetchLogs() {
  loading.value = true;
  try {
    const msg = await HttpUtil.get('/panel/api/audit/logs', { page: page.value, pageSize: pageSize.value });
    if (msg?.success) {
      logs.value = msg.obj.logs || [];
      total.value = msg.obj.total || 0;
    }
  } finally {
    loading.value = false;
  }
}

function onPageChange(p) {
  page.value = p;
  fetchLogs();
}

onMounted(fetchLogs);
</script>

<template>
  <div class="audit-log-tab">
    <a-table :data-source="logs" :loading="loading" row-key="id"
      :pagination="{ current: page, pageSize, total, onChange: onPageChange }" size="small">
      <a-table-column title="Time" data-index="createdAt" :width="160">
        <template #default="{ text }">{{ text ? new Date(text).toLocaleString() : '-' }}</template>
      </a-table-column>
      <a-table-column title="User" data-index="username" :width="120" />
      <a-table-column title="Action" data-index="action" :width="250" />
      <a-table-column title="Method" data-index="method" :width="70" />
      <a-table-column title="Status" data-index="status" :width="70">
        <template #default="{ text }">
          <a-tag :color="text < 400 ? 'green' : 'red'">{{ text }}</a-tag>
        </template>
      </a-table-column>
      <a-table-column title="IP" data-index="ip" :width="140" />
      <a-table-column title="Detail" data-index="detail" ellipsis />
    </a-table>
  </div>
</template>
