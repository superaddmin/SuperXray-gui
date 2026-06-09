<template>
  <section class="page-stack api-docs-page">
    <PageHeader
      :eyebrow="translateText('apiDocs.eyebrow')"
      :title="translateText('apiDocs.title')"
      :description="translateText('apiDocs.description')"
    >
      <ASpace wrap>
        <AButton @click="copyDocumentURL">{{ translateText('apiDocs.copyUrl') }}</AButton>
        <AButton :disabled="!document" @click="downloadDocument">
          {{ translateText('apiDocs.downloadJson') }}
        </AButton>
        <AButton type="primary" @click="loadDocument">
          {{ translateText('apiDocs.reload') }}
        </AButton>
      </ASpace>
    </PageHeader>

    <AAlert
      type="info"
      show-icon
      :message="translateText('apiDocs.readOnlyMessage')"
      :description="translateText('apiDocs.readOnlyDescription')"
    />

    <div class="status-grid api-docs-summary">
      <StatusTile
        :label="translateText('apiDocs.basePath')"
        :value="apiBasePath"
        :hint="translateText('apiDocs.basePathHint')"
        tone="info"
      />
      <StatusTile
        :label="translateText('apiDocs.auth')"
        value="Cookie"
        hint="SuperXray"
        tone="success"
      />
      <StatusTile
        :label="translateText('apiDocs.csrf')"
        value="X-CSRF-Token"
        :hint="translateText('apiDocs.csrfHint')"
        tone="warning"
      />
      <StatusTile
        :label="translateText('apiDocs.unauthenticated')"
        value="404"
        :hint="translateText('apiDocs.authHiddenHint')"
        tone="neutral"
      />
    </div>

    <ACard class="work-panel" :bordered="false">
      <div class="panel-header api-docs-toolbar">
        <div>
          <p class="page-eyebrow">{{ translateText('apiDocs.localContract') }}</p>
          <h2>{{ translateText('apiDocs.serverUrl') }} {{ serverURL }}</h2>
          <p class="muted-text">{{ documentURL }}</p>
        </div>
        <ASpace wrap>
          <AInput
            v-model:value="searchText"
            allow-clear
            class="api-docs-search"
            :aria-label="translateText('apiDocs.searchLabel')"
            :placeholder="translateText('apiDocs.searchPlaceholder')"
          />
          <ASelect
            v-model:value="selectedMethod"
            class="api-docs-filter"
            :aria-label="translateText('apiDocs.methodFilter')"
            :options="methodOptions"
          />
          <ASelect
            v-model:value="selectedTag"
            class="api-docs-filter"
            :aria-label="translateText('apiDocs.tagFilter')"
            :options="tagOptions"
          />
        </ASpace>
      </div>
    </ACard>

    <AResult
      v-if="error"
      status="warning"
      :title="translateText('apiDocs.errorTitle')"
      :sub-title="error"
    />
    <div v-else-if="loading" class="api-docs-loading">
      <ASpin />
      <span>{{ translateText('apiDocs.loading') }}</span>
    </div>

    <template v-else>
      <AEmpty
        v-if="groupedOperations.length === 0"
        :description="translateText('apiDocs.noOperations')"
      />
      <template v-else>
        <ACard
          v-for="group in groupedOperations"
          :key="group.tag"
          class="work-panel api-docs-section"
          :bordered="false"
        >
          <div class="panel-header">
            <div>
              <p class="page-eyebrow">{{ translateText('apiDocs.section') }}</p>
              <h2>{{ group.tag }}</h2>
              <p v-if="group.description" class="muted-text">{{ group.description }}</p>
            </div>
            <ATag>{{ group.operations.length }} {{ translateText('apiDocs.operations') }}</ATag>
          </div>

          <ACollapse>
            <ACollapsePanel v-for="operation in group.operations" :key="operation.key">
              <template #header>
                <div class="api-docs-operation-header">
                  <ATag :color="methodColor(operation.method)">
                    {{ operation.method.toUpperCase() }}
                  </ATag>
                  <code>{{ operation.path }}</code>
                  <span>{{ operation.summary || operation.operationId }}</span>
                </div>
              </template>

              <div class="api-docs-operation-body">
                <p v-if="operation.description" class="muted-text">{{ operation.description }}</p>
                <ADescriptions bordered size="small" :column="1">
                  <ADescriptionsItem :label="translateText('apiDocs.operationId')">
                    {{ operation.operationId || '-' }}
                  </ADescriptionsItem>
                  <ADescriptionsItem :label="translateText('apiDocs.security')">
                    <ASpace wrap>
                      <ATag v-for="scheme in operation.security" :key="scheme" color="green">
                        {{ scheme }}
                      </ATag>
                    </ASpace>
                  </ADescriptionsItem>
                  <ADescriptionsItem :label="translateText('apiDocs.responses')">
                    <ASpace wrap>
                      <ATag v-for="code in operation.responseCodes" :key="code">{{ code }}</ATag>
                    </ASpace>
                  </ADescriptionsItem>
                </ADescriptions>

                <div v-if="operation.parameters.length > 0" class="api-docs-block">
                  <h3>{{ translateText('apiDocs.parameters') }}</h3>
                  <ATable
                    :columns="parameterColumns"
                    :data-source="operation.parameters"
                    :pagination="false"
                    row-key="key"
                    size="small"
                  />
                </div>

                <div v-if="operation.requestBody" class="api-docs-block">
                  <h3>{{ translateText('apiDocs.requestBody') }}</h3>
                  <pre
                    class="code-preview compact-preview api-docs-code"
                  ><code>{{ stringify(operation.requestBody) }}</code></pre>
                </div>

                <div class="api-docs-block">
                  <h3>{{ translateText('apiDocs.rawOperation') }}</h3>
                  <pre
                    class="code-preview compact-preview api-docs-code"
                  ><code>{{ stringify(operation.raw) }}</code></pre>
                </div>
              </div>
            </ACollapsePanel>
          </ACollapse>
        </ACard>
      </template>
    </template>
  </section>
</template>

<script setup lang="ts">
import {
  Alert as AAlert,
  Button as AButton,
  Card as ACard,
  Collapse as ACollapse,
  CollapsePanel as ACollapsePanel,
  Descriptions as ADescriptions,
  DescriptionsItem as ADescriptionsItem,
  Empty as AEmpty,
  Input as AInput,
  Result as AResult,
  Select as ASelect,
  Space as ASpace,
  Spin as ASpin,
  Table as ATable,
  Tag as ATag,
  message,
} from 'ant-design-vue';
import type { SelectProps, TableColumnsType } from 'ant-design-vue';
import { computed, onMounted, ref } from 'vue';

import {
  OpenAPIDocumentLoadError,
  fetchOpenAPIDocument,
  getOpenAPIDocumentURL,
} from '@/api/openapi';
import PageHeader from '@/components/PageHeader.vue';
import StatusTile from '@/components/StatusTile.vue';
import { translate } from '@/i18n/messages';
import { useAppStore } from '@/stores/app';
import type {
  OpenAPIDocument,
  OpenAPIMethod,
  OpenAPIOperation,
  OpenAPIParameter,
  OpenAPIParameterOrRef,
} from '@/types/openapi';
import { copyText, downloadText } from '@/utils/textExport';

interface ParameterRow {
  description: string;
  in: string;
  key: string;
  name: string;
  required: string;
}

interface OperationRow {
  description: string;
  key: string;
  method: OpenAPIMethod;
  operationId: string;
  parameters: ParameterRow[];
  path: string;
  raw: OpenAPIOperation;
  requestBody?: unknown;
  responseCodes: string[];
  security: string[];
  summary: string;
  tag: string;
}

interface OperationGroup {
  description: string;
  operations: OperationRow[];
  tag: string;
}

const methods: OpenAPIMethod[] = [
  'get',
  'post',
  'put',
  'delete',
  'patch',
  'head',
  'options',
  'trace',
];
const appStore = useAppStore();
const document = ref<OpenAPIDocument>();
const error = ref('');
const loading = ref(true);
const searchText = ref('');
const selectedMethod = ref<OpenAPIMethod | 'all'>('all');
const selectedTag = ref('all');

const documentURL = computed(() => getOpenAPIDocumentURL());
const apiBasePath = computed(() => documentURL.value.replace(/\/openapi\.json$/, ''));
const serverURL = computed(() => document.value?.servers?.[0]?.url || '/');
const methodOptions = computed<SelectProps['options']>(() => [
  { label: translateText('apiDocs.allMethods'), value: 'all' },
  ...methods.map((method) => ({ label: method.toUpperCase(), value: method })),
]);
const tagDescriptions = computed(() => {
  const descriptions = new Map<string, string>();
  document.value?.tags?.forEach((tag) => descriptions.set(tag.name, tag.description || ''));
  operations.value.forEach((operation) => {
    if (!descriptions.has(operation.tag)) {
      descriptions.set(operation.tag, '');
    }
  });
  return descriptions;
});
const tagOptions = computed<SelectProps['options']>(() => [
  { label: translateText('apiDocs.allTags'), value: 'all' },
  ...Array.from(tagDescriptions.value.keys()).map((tag) => ({ label: tag, value: tag })),
]);
const operations = computed<OperationRow[]>(() => {
  if (!document.value) {
    return [];
  }

  return Object.entries(document.value.paths).flatMap(([path, pathItem]) => {
    const pathParameters = pathItem.parameters || [];
    return methods.flatMap((method) => {
      const operation = pathItem[method];
      if (!operation) {
        return [];
      }
      const tag = operation.tags?.[0] || 'Panel API';
      const parameters = [...pathParameters, ...(operation.parameters || [])].map(
        (parameter, index) => toParameterRow(parameter, `${method}-${path}-${index}`),
      );

      return [
        {
          description: operation.description || '',
          key: `${method}-${path}`,
          method,
          operationId: operation.operationId || '',
          parameters,
          path,
          raw: operation,
          requestBody: operation.requestBody,
          responseCodes: Object.keys(operation.responses || {}),
          security: operationSecurity(operation),
          summary: operation.summary || '',
          tag,
        },
      ];
    });
  });
});
const filteredOperations = computed(() => {
  const search = searchText.value.trim().toLocaleLowerCase('en-US');
  return operations.value.filter((operation) => {
    if (selectedMethod.value !== 'all' && operation.method !== selectedMethod.value) {
      return false;
    }
    if (selectedTag.value !== 'all' && operation.tag !== selectedTag.value) {
      return false;
    }
    if (!search) {
      return true;
    }
    return [
      operation.path,
      operation.summary,
      operation.description,
      operation.operationId,
      operation.tag,
    ]
      .join(' ')
      .toLocaleLowerCase('en-US')
      .includes(search);
  });
});
const groupedOperations = computed<OperationGroup[]>(() => {
  const groups = new Map<string, OperationRow[]>();
  filteredOperations.value.forEach((operation) => {
    const group = groups.get(operation.tag) || [];
    group.push(operation);
    groups.set(operation.tag, group);
  });
  return Array.from(groups.entries()).map(([tag, groupOperations]) => ({
    description: tagDescriptions.value.get(tag) || '',
    operations: groupOperations,
    tag,
  }));
});

const parameterColumns = computed<TableColumnsType<ParameterRow>>(() => [
  { title: translateText('apiDocs.paramName'), dataIndex: 'name', key: 'name' },
  { title: translateText('apiDocs.paramIn'), dataIndex: 'in', key: 'in', width: 90 },
  {
    title: translateText('apiDocs.paramRequired'),
    dataIndex: 'required',
    key: 'required',
    width: 110,
  },
  { title: translateText('apiDocs.paramDescription'), dataIndex: 'description', key: 'description' },
]);

onMounted(() => {
  void loadDocument();
});

async function loadDocument() {
  loading.value = true;
  error.value = '';
  try {
    document.value = await fetchOpenAPIDocument();
  } catch (caught) {
    document.value = undefined;
    error.value = formatLoadError(caught);
  } finally {
    loading.value = false;
  }
}

function translateText(key: Parameters<typeof translate>[0]): string {
  return translate(key, appStore.locale);
}

function formatLoadError(caught: unknown): string {
  if (caught instanceof OpenAPIDocumentLoadError) {
    if (caught.sessionExpired) {
      return translateText('apiDocs.sessionExpired');
    }
    return `${translateText('apiDocs.loadFailed')} (${caught.status})`;
  }
  return caught instanceof Error ? caught.message : String(caught);
}

function methodColor(method: OpenAPIMethod): string {
  return method === 'get' ? 'blue' : method === 'post' ? 'orange' : 'default';
}

function toParameterRow(parameterOrRef: OpenAPIParameterOrRef, key: string): ParameterRow {
  const parameter = resolveParameter(parameterOrRef);
  return {
    description: parameter?.description || stringify(parameter?.schema || parameterOrRef),
    in: parameter?.in || '-',
    key,
    name: parameter?.name || referenceName(parameterOrRef),
    required: parameter?.required ? translateText('apiDocs.yes') : translateText('apiDocs.no'),
  };
}

function resolveParameter(parameter: OpenAPIParameterOrRef): OpenAPIParameter | undefined {
  if (!('$ref' in parameter)) {
    return parameter;
  }
  const prefix = '#/components/parameters/';
  if (!parameter.$ref.startsWith(prefix)) {
    return undefined;
  }
  const key = parameter.$ref.slice(prefix.length);
  return document.value?.components?.parameters?.[key];
}

function referenceName(parameter: OpenAPIParameterOrRef): string {
  if (!('$ref' in parameter)) {
    return parameter.name || '-';
  }
  return parameter.$ref.split('/').pop() || parameter.$ref;
}

function operationSecurity(operation: OpenAPIOperation): string[] {
  const schemes = new Set<string>();
  operation.security?.forEach((requirement) => {
    Object.keys(requirement).forEach((scheme) => schemes.add(scheme));
  });
  if (schemes.size === 0) {
    schemes.add('cookieAuth');
  }
  return Array.from(schemes);
}

function stringify(value: unknown): string {
  return JSON.stringify(value ?? {}, null, 2);
}

async function copyDocumentURL() {
  await copyText(documentURL.value);
  void message.success(translateText('apiDocs.copied'));
}

function downloadDocument() {
  if (!document.value) {
    return;
  }
  downloadText(
    'superxray-panel-openapi.json',
    `${JSON.stringify(document.value, null, 2)}\n`,
    'application/json;charset=utf-8',
  );
}
</script>
