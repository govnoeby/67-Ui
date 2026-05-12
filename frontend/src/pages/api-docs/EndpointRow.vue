<script setup>
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { methodColors } from './endpoints.js';

const { t } = useI18n();

const props = defineProps({
  endpoint: { type: Object, required: true },
  sectionId: { type: String, default: '' },
  endpointIdx: { type: Number, default: -1 },
});

const tagColor = computed(() => methodColors[props.endpoint.method] || 'default');
const hasParams = computed(() => Array.isArray(props.endpoint.params) && props.endpoint.params.length > 0);

const translatedParams = computed(() => {
  if (!props.endpoint.params) return [];
  return props.endpoint.params.map((p, i) => ({
    ...p,
    desc: paramDesc(p, i),
  }));
});

function endpointSummary(endpoint) {
  const key = `pages.apiDocs.sections.${props.sectionId}.endpoints.${props.endpointIdx}.summary`;
  const translated = t(key);
  return translated !== key ? translated : endpoint.summary;
}

function paramDesc(param, idx) {
  const key = `pages.apiDocs.sections.${props.sectionId}.endpoints.${props.endpointIdx}.params.${idx}.desc`;
  const translated = t(key);
  return translated !== key ? translated : param.desc;
}

const paramColumns = computed(() => [
  { title: t('pages.apiDocs.paramName'), dataIndex: 'name', key: 'name', width: 180 },
  { title: t('pages.apiDocs.paramIn'), dataIndex: 'in', key: 'in', width: 100 },
  { title: t('pages.apiDocs.paramType'), dataIndex: 'type', key: 'type', width: 120 },
  { title: t('pages.apiDocs.paramDesc'), dataIndex: 'desc', key: 'desc' },
]);
</script>

<template>
  <div class="endpoint-row">
    <div class="endpoint-header">
      <a-tag :color="tagColor" class="method-tag">{{ endpoint.method }}</a-tag>
      <code class="endpoint-path">{{ endpoint.path }}</code>
    </div>

    <p v-if="endpoint.summary" class="endpoint-summary">{{ endpointSummary(endpoint) }}</p>

    <div v-if="hasParams" class="endpoint-block">
      <div class="block-label">{{ t('pages.apiDocs.parameters') }}</div>
      <a-table :columns="paramColumns" :data-source="translatedParams" :pagination="false" size="small" row-key="name" />
    </div>

    <div v-if="endpoint.body" class="endpoint-block">
      <div class="block-label">{{ t('pages.apiDocs.requestBody') }}</div>
      <a-typography-paragraph :copyable="{ text: endpoint.body }">
        <pre class="code-block">{{ endpoint.body }}</pre>
      </a-typography-paragraph>
    </div>

    <div v-if="endpoint.response" class="endpoint-block">
      <div class="block-label">{{ t('pages.apiDocs.response') }}</div>
      <a-typography-paragraph :copyable="{ text: endpoint.response }">
        <pre class="code-block">{{ endpoint.response }}</pre>
      </a-typography-paragraph>
    </div>
  </div>
</template>

<style scoped>
.endpoint-row {
  padding: 12px 0;
}

.endpoint-row + .endpoint-row {
  border-top: 1px solid rgba(128, 128, 128, 0.15);
}

.endpoint-header {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.method-tag {
  font-weight: 600;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  letter-spacing: 0.5px;
  min-width: 60px;
  text-align: center;
}

.endpoint-path {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 13px;
  word-break: break-all;
}

.endpoint-summary {
  margin: 8px 0 0;
  color: rgba(0, 0, 0, 0.65);
  line-height: 1.55;
}

.endpoint-block {
  margin-top: 12px;
}

.block-label {
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: rgba(0, 0, 0, 0.5);
  margin-bottom: 6px;
}

.code-block {
  background: rgba(128, 128, 128, 0.08);
  border: 1px solid rgba(128, 128, 128, 0.15);
  border-radius: 6px;
  padding: 10px 12px;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 12.5px;
  line-height: 1.55;
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  overflow-x: auto;
}
</style>

<style>
body.dark .endpoint-summary {
  color: rgba(255, 255, 255, 0.7);
}

body.dark .block-label {
  color: rgba(255, 255, 255, 0.55);
}

body.dark .code-block {
  background: rgba(255, 255, 255, 0.04);
  border-color: rgba(255, 255, 255, 0.1);
  color: rgba(255, 255, 255, 0.88);
}
</style>
