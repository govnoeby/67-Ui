<script setup>
import { useI18n } from 'vue-i18n';
import EndpointRow from './EndpointRow.vue';

const { t } = useI18n();

const props = defineProps({
  section: { type: Object, required: true },
});

function sectionTitle(section) {
  const key = `pages.apiDocs.sections.${section.id}.title`;
  const translated = t(key);
  return translated !== key ? translated : section.title;
}

function sectionDesc(section) {
  if (!section.desc) return '';
  const key = `pages.apiDocs.sections.${section.id}.desc`;
  const translated = t(key);
  return translated !== key ? translated : section.desc;
}
</script>

<template>
  <section :id="section.id" class="api-section">
    <h2 class="section-title">{{ sectionTitle(section) }}</h2>
    <p v-if="section.desc" class="section-description">{{ sectionDesc(section) }}</p>
    <div class="endpoints">
      <EndpointRow v-for="(endpoint, idx) in section.endpoints" :key="idx" :endpoint="endpoint" :section-id="section.id" :endpoint-idx="idx" />
    </div>
  </section>
</template>

<style scoped>
.api-section {
  background: #fff;
  border: 1px solid rgba(128, 128, 128, 0.15);
  border-radius: 8px;
  padding: 20px 24px;
  margin-bottom: 20px;
  scroll-margin-top: 16px;
}

.section-title {
  font-size: 20px;
  font-weight: 600;
  margin: 0;
  color: rgba(0, 0, 0, 0.88);
}

.section-description {
  margin: 6px 0 14px;
  color: rgba(0, 0, 0, 0.65);
  line-height: 1.55;
}

.endpoints > :first-child {
  padding-top: 0;
}
</style>

<style>
body.dark .api-section {
  background: #252526;
  border-color: rgba(255, 255, 255, 0.1);
}

html[data-theme='ultra-dark'] .api-section {
  background: #0a0a0a;
  border-color: rgba(255, 255, 255, 0.08);
}

body.dark .section-title {
  color: rgba(255, 255, 255, 0.92);
}

body.dark .section-description {
  color: rgba(255, 255, 255, 0.7);
}
</style>
