<script setup>
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { ExclamationCircleFilled } from '@ant-design/icons-vue';
import { ObservatorySettings, BurstObservatorySettings } from '@/models/outbound.js';
import SettingListItem from '@/components/SettingListItem.vue';

const { t } = useI18n();

const props = defineProps({
  templateSettings: { type: Object, default: null },
});

const outboundTagOptions = computed(() => {
  const out = [];
  for (const ob of props.templateSettings?.outbounds || []) {
    if (ob.tag && ob.tag !== 'blocked') out.push(ob.tag);
  }
  return out;
});

const observatory = computed({
  get: () => ObservatorySettings.fromJson(props.templateSettings?.observatory),
  set: (next) => {
    if (!props.templateSettings) return;
    const json = next.toJson();
    if (json) {
      props.templateSettings.observatory = json;
    } else {
      delete props.templateSettings.observatory;
    }
  },
});

const burstObservatory = computed({
  get: () => BurstObservatorySettings.fromJson(props.templateSettings?.burstObservatory),
  set: (next) => {
    if (!props.templateSettings) return;
    const json = next.toJson();
    if (json) {
      props.templateSettings.burstObservatory = json;
    } else {
      delete props.templateSettings.burstObservatory;
    }
  },
});

const obsEnabled = computed({
  get: () => observatory.value.enabled,
  set: (val) => {
    const o = observatory.value;
    o.enabled = val;
    observatory.value = o;
  },
});

const burstEnabled = computed({
  get: () => burstObservatory.value.enabled,
  set: (val) => {
    const b = burstObservatory.value;
    b.enabled = val;
    burstObservatory.value = b;
  },
});

const obsInterval = computed({
  get: () => observatory.value.probeInterval,
  set: (val) => {
    const o = observatory.value;
    o.probeInterval = val;
    observatory.value = o;
  },
});

const obsUrl = computed({
  get: () => observatory.value.probeUrl,
  set: (val) => {
    const o = observatory.value;
    o.probeUrl = val;
    observatory.value = o;
  },
});

const obsSubject = computed({
  get: () => observatory.value.subjectSelector,
  set: (val) => {
    const o = observatory.value;
    o.subjectSelector = val;
    observatory.value = o;
  },
});

const obsConcurrency = computed({
  get: () => observatory.value.enableConcurrency,
  set: (val) => {
    const o = observatory.value;
    o.enableConcurrency = val;
    observatory.value = o;
  },
});

const obsMaxQuery = computed({
  get: () => observatory.value.maxQurey,
  set: (val) => {
    const o = observatory.value;
    o.maxQurey = val;
    observatory.value = o;
  },
});

const burstSubject = computed({
  get: () => burstObservatory.value.subjectSelector,
  set: (val) => {
    const b = burstObservatory.value;
    b.subjectSelector = val;
    burstObservatory.value = b;
  },
});

const burstInterval = computed({
  get: () => burstObservatory.value.pingInterval,
  set: (val) => {
    const b = burstObservatory.value;
    b.pingInterval = val;
    burstObservatory.value = b;
  },
});

const burstParallel = computed({
  get: () => burstObservatory.value.pingParallel,
  set: (val) => {
    const b = burstObservatory.value;
    b.pingParallel = val;
    burstObservatory.value = b;
  },
});

const burstDestination = computed({
  get: () => burstObservatory.value.pingDestination,
  set: (val) => {
    const b = burstObservatory.value;
    b.pingDestination = val;
    burstObservatory.value = b;
  },
});

const burstConnectivity = computed({
  get: () => burstObservatory.value.pingConnectivity,
  set: (val) => {
    const b = burstObservatory.value;
    b.pingConnectivity = val;
    burstObservatory.value = b;
  },
});
</script>

<template>
  <a-collapse default-active-key="1">
    <a-collapse-panel key="1" header="Observatory (Health Check)">
      <a-alert type="info" class="mb-12 hint-alert">
        <template #icon>
          <ExclamationCircleFilled style="color: #1890ff;" />
        </template>
        {{ t('pages.xray.observatoryDesc') || 'Observatory periodically probes outbound connections and detects failures. Combined with outboundFallbackTag in routing rules, traffic automatically switches to a healthy backup outbound when the primary is blocked.' }}
      </a-alert>

      <SettingListItem paddings="small">
        <template #title>Enable Observatory</template>
        <template #description>Periodically check outbound health for automatic failover</template>
        <template #control>
          <a-switch v-model:checked="obsEnabled" />
        </template>
      </SettingListItem>

      <template v-if="obsEnabled">
        <SettingListItem paddings="small">
          <template #title>Probe Interval</template>
          <template #description>How often to probe (e.g. 10s, 30s, 1m)</template>
          <template #control>
            <a-input v-model:value="obsInterval" placeholder="10s" />
          </template>
        </SettingListItem>

        <SettingListItem paddings="small">
          <template #title>Probe URL</template>
          <template #description>URL used for health checks</template>
          <template #control>
            <a-input v-model:value="obsUrl" placeholder="https://www.gstatic.com/generate_204" />
          </template>
        </SettingListItem>

        <SettingListItem paddings="small">
          <template #title>Subject Outbounds</template>
          <template #description>Outbounds to monitor — mark unhealthy outbounds are skipped by fallback routing</template>
          <template #control>
            <a-select v-model:value="obsSubject" mode="multiple" :style="{ width: '100%' }">
              <a-select-option v-for="tag in outboundTagOptions" :key="tag" :value="tag">{{ tag }}</a-select-option>
            </a-select>
          </template>
        </SettingListItem>

        <SettingListItem paddings="small">
          <template #title>Concurrency</template>
          <template #description>Probe all subjects simultaneously</template>
          <template #control>
            <a-switch v-model:checked="obsConcurrency" />
          </template>
        </SettingListItem>

        <SettingListItem paddings="small">
          <template #title>Max Query</template>
          <template #description>Maximum tracked queries per subject</template>
          <template #control>
            <a-input-number v-model:value="obsMaxQuery" :min="1" :max="100" />
          </template>
        </SettingListItem>
      </template>
    </a-collapse-panel>

    <a-collapse-panel key="2" header="Burst Observatory">
      <a-alert type="info" class="mb-12 hint-alert">
        <template #icon>
          <ExclamationCircleFilled style="color: #1890ff;" />
        </template>
        Aggressive health check variant — bursts ping all subjects immediately when any subject is detected as unhealthy, enabling faster failover.
      </a-alert>

      <SettingListItem paddings="small">
        <template #title>Enable Burst Observatory</template>
        <template #control>
          <a-switch v-model:checked="burstEnabled" />
        </template>
      </SettingListItem>

      <template v-if="burstEnabled">
        <SettingListItem paddings="small">
          <template #title>Subject Outbounds</template>
          <template #control>
            <a-select v-model:value="burstSubject" mode="multiple" :style="{ width: '100%' }">
              <a-select-option v-for="tag in outboundTagOptions" :key="tag" :value="tag">{{ tag }}</a-select-option>
            </a-select>
          </template>
        </SettingListItem>

        <SettingListItem paddings="small">
          <template #title>Ping Interval</template>
          <template #control>
            <a-input v-model:value="burstInterval" placeholder="10s" />
          </template>
        </SettingListItem>

        <SettingListItem paddings="small">
          <template #title>Parallel Ping</template>
          <template #description>Probe all destinations in parallel on each interval</template>
          <template #control>
            <a-switch v-model:checked="burstParallel" />
          </template>
        </SettingListItem>

        <SettingListItem paddings="small">
          <template #title>Destination</template>
          <template #control>
            <a-input v-model:value="burstDestination" placeholder="https://www.gstatic.com/generate_204" />
          </template>
        </SettingListItem>

        <SettingListItem paddings="small">
          <template #title>Connectivity</template>
          <template #description>Optional connectivity check script/URL</template>
          <template #control>
            <a-input v-model:value="burstConnectivity" placeholder="optional" />
          </template>
        </SettingListItem>
      </template>
    </a-collapse-panel>

    <a-collapse-panel key="3" header="Quick Start Guide">
      <div style="padding: 0 20px; font-size: 13px; line-height: 1.8;">
        <p><strong>How to set up automatic protocol blocking bypass:</strong></p>
        <ol>
          <li>Create a <strong>primary outbound</strong> with your preferred protocol/transport</li>
          <li>Create a <strong>backup outbound</strong> with the same server but <strong>xHTTP (SplitHTTP)</strong> transport — xHTTP mimics standard HTTP and is harder to block</li>
          <li>In <strong>Routing Rules</strong>, set the primary outbound's tag as the target, and the backup's tag as <strong>Fallback Outbound Tag</strong></li>
          <li>Enable <strong>Observatory</strong> above and add the primary outbound to the subject list</li>
          <li>When xray detects the primary outbound is blocked, traffic automatically switches to the xHTTP backup</li>
        </ol>
        <p><strong>Dynamic Port rotation:</strong> In the outbound form, use the "Ports" field to specify multiple ports (e.g. "443,8443,2053"). Multiple outbounds will be created automatically, one per port. Add them to Observatory subjects and route traffic through them.</p>
      </div>
    </a-collapse-panel>
  </a-collapse>
</template>

<style scoped>
.mb-12 {
  margin-bottom: 12px;
}
.hint-alert {
  text-align: center;
}
</style>
