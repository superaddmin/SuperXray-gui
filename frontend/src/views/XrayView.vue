<template>
  <section class="page-stack xray-page">
    <PageHeader
      eyebrow="Runtime"
      title="Xray"
      description="Control the legacy Xray runtime, template, versions, outbounds, and provider tools."
    >
      <ASpace wrap>
        <AButton :loading="refreshing" @click="refreshRuntime">
          <template #icon><ReloadOutlined /></template>
          Refresh
        </AButton>
        <AButton :disabled="!configText" @click="copyConfig">
          <template #icon><CopyOutlined /></template>
          Copy
        </AButton>
        <AButton :disabled="!configText" @click="downloadConfig">
          <template #icon><DownloadOutlined /></template>
          Download
        </AButton>
      </ASpace>
    </PageHeader>

    <AAlert v-if="error" banner type="warning" :message="error" />

    <div class="status-grid">
      <StatusTile
        label="Xray State"
        :value="xrayStateLabel"
        :hint="xrayErrorHint"
        :tone="xrayStateTone"
      />
      <StatusTile
        label="Current Version"
        :value="currentVersion"
        hint="Existing Xray process"
        tone="info"
      />
      <StatusTile
        label="Template"
        :value="templateState"
        :hint="configChangedHint"
        :tone="configChanged ? 'warning' : 'success'"
      />
      <StatusTile
        label="Outbound Test"
        :value="outboundTestUrl || '-'"
        hint="Saved with legacy template"
        tone="neutral"
      />
    </div>

    <ACard class="work-panel" :bordered="false">
      <div class="panel-header">
        <div>
          <p class="page-eyebrow">Lifecycle</p>
          <h2>Xray Runtime Control</h2>
        </div>
        <ASpace wrap>
          <AButton :loading="lifecycleBusy" type="primary" @click="confirmStartOrRestart">
            <template #icon><PoweroffOutlined /></template>
            {{ startRestartLabel }}
          </AButton>
          <AButton :loading="lifecycleBusy" danger @click="confirmStop">
            <template #icon><PauseCircleOutlined /></template>
            Stop
          </AButton>
        </ASpace>
      </div>
      <pre class="code-preview compact-preview">{{ runtimeResult }}</pre>
    </ACard>

    <ACard class="work-panel" :bordered="false">
      <div class="panel-header">
        <div>
          <p class="page-eyebrow">Version</p>
          <h2>Xray Version Management</h2>
        </div>
        <ASpace wrap>
          <AButton :loading="loadingVersions" @click="loadVersions">
            <template #icon><ReloadOutlined /></template>
            Versions
          </AButton>
          <AButton
            :disabled="!selectedVersion"
            :loading="installingVersion"
            danger
            @click="confirmInstallVersion"
          >
            Install
          </AButton>
        </ASpace>
      </div>
      <ASelect
        v-model:value="selectedVersion"
        aria-label="Select Xray version"
        class="version-select"
        :loading="loadingVersions"
        :options="versionOptions"
        placeholder="Select Xray version"
        show-search
      />
    </ACard>

    <ACard class="work-panel" :bordered="false">
      <div class="panel-header">
        <div>
          <p class="page-eyebrow">Configuration</p>
          <h2>Xray Template Editor</h2>
        </div>
        <ASpace wrap>
          <AButton :disabled="!configText" @click="formatConfig">
            <template #icon><AlignLeftOutlined /></template>
            Format
          </AButton>
          <AButton
            :disabled="!configChanged"
            :loading="savingConfig"
            type="primary"
            @click="confirmSaveConfig"
          >
            Save
          </AButton>
        </ASpace>
      </div>

      <AInput
        v-model:value="outboundTestUrl"
        aria-label="Outbound test URL"
        class="mb-12"
        placeholder="Outbound test URL"
      />
      <textarea
        v-model="configText"
        aria-label="Xray JSON template"
        class="json-editor"
        spellcheck="false"
      />
    </ACard>

    <ACard class="work-panel gateway-egress-mvp-panel" :bordered="false">
      <FormSection
        eyebrow="Gateway"
        title="Gateway Egress MVP"
        description="Generate Xray-compatible Gateway-facing SOCKS5 inbounds and a manifest without adding backend egress models."
      >
        <template #actions>
          <ASpace wrap>
            <AButton
              :disabled="Boolean(gatewayEgressNetworkError)"
              type="primary"
              @click="applyGatewayEgressMvp"
            >
              Generate Xray Config
            </AButton>
            <AButton
              :disabled="Boolean(gatewayEgressNetworkError)"
              @click="copyGatewayEgressManifest"
            >
              <template #icon><CopyOutlined /></template>
              Copy Manifest
            </AButton>
            <AButton
              :disabled="Boolean(gatewayEgressNetworkError)"
              @click="downloadGatewayEgressManifest"
            >
              <template #icon><DownloadOutlined /></template>
              Download Manifest
            </AButton>
          </ASpace>
        </template>

        <AAlert
          v-if="gatewayEgressNetworkError"
          class="mb-12"
          show-icon
          type="warning"
          :message="gatewayEgressNetworkError"
        />

        <AForm class="gateway-egress-network-form" layout="vertical">
          <div class="form-grid">
            <AFormItem label="Listen Host">
              <AInput v-model:value="gatewayEgressNetwork.listenHost" />
            </AFormItem>
            <AFormItem label="Manifest Host">
              <AInput v-model:value="gatewayEgressNetwork.manifestHost" />
            </AFormItem>
            <AFormItem label="Strategy Label">
              <AInput v-model:value="gatewayEgressNetwork.strategyLabel" />
            </AFormItem>
          </div>
        </AForm>

        <div class="gateway-egress-mvp-grid">
          <div class="client-link-card">
            <strong>{{ gatewayEgressMvpPreview.profileCount }}</strong>
            <p class="muted-text">profiles</p>
          </div>
          <div class="client-link-card">
            <strong>{{ gatewayEgressMvpPreview.ports.join(', ') }}</strong>
            <p class="muted-text">ports</p>
          </div>
          <div class="client-link-card">
            <strong>{{ gatewayEgressMvpPreview.listenHost }}</strong>
            <p class="muted-text">Xray listen host</p>
          </div>
          <div class="client-link-card">
            <strong>{{ gatewayEgressMvpPreview.manifestHost }}</strong>
            <p class="muted-text">Gateway manifest host</p>
          </div>
        </div>

        <pre class="code-preview compact-preview mt-16">{{
          gatewayEgressManifestCsv || 'Select a valid network strategy before exporting manifest.'
        }}</pre>
      </FormSection>
    </ACard>

    <ACard class="work-panel" :bordered="false">
      <div class="panel-header">
        <div>
          <p class="page-eyebrow">Outbound</p>
          <h2>Outbound Tools</h2>
        </div>
        <ASpace wrap>
          <AButton :loading="loadingOutboundsTraffic" @click="loadOutboundsTraffic">
            <template #icon><ReloadOutlined /></template>
            Refresh Traffic
          </AButton>
          <AButton
            :disabled="outboundsFromConfig.length === 0"
            :loading="testingOutbound"
            type="primary"
            @click="runFirstOutboundTest"
          >
            Test First Outbound
          </AButton>
          <AButton :loading="resettingOutboundTraffic" danger @click="confirmResetAllTraffic">
            Reset All Traffic
          </AButton>
        </ASpace>
      </div>

      <ATable
        :columns="outboundTrafficColumns"
        :data-source="outboundTrafficRows"
        :loading="loadingOutboundsTraffic"
        :pagination="false"
        row-key="tag"
        size="middle"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'action'">
            <AButton
              size="small"
              type="link"
              :loading="resettingOutboundTag === record.tag"
              @click="confirmResetOutboundTraffic(record.tag)"
            >
              Reset
            </AButton>
          </template>
        </template>
      </ATable>

      <pre v-if="outboundToolResult" class="code-preview compact-preview mt-16">{{
        outboundToolResult
      }}</pre>

      <div class="panel-header mt-16">
        <div>
          <p class="page-eyebrow">Providers</p>
          <h2>Warp / Nord</h2>
        </div>
        <ASpace wrap>
          <AButton :loading="providerAction === 'warp-data'" @click="runProviderData('warp')">
            Warp Data
          </AButton>
          <AButton
            :loading="providerAction === 'warp-config'"
            @click="runProviderData('warp-config')"
          >
            Warp Config
          </AButton>
          <AButton :loading="providerAction === 'nord-data'" @click="runProviderData('nord')">
            Nord Data
          </AButton>
          <AButton
            :loading="providerAction === 'nord-countries'"
            @click="runProviderData('nord-countries')"
          >
            Nord Countries
          </AButton>
        </ASpace>
      </div>
    </ACard>

    <ACard class="work-panel" :bordered="false">
      <FormSection eyebrow="Structured" title="Residential IP Pool">
        <template #actions>
          <ASpace wrap>
            <AButton type="primary" @click="openResidentialIpModal()">
              <template #icon><PlusOutlined /></template>
              Add SOCKS5 IP
            </AButton>
            <AButton
              :disabled="residentialIpRows.length === 0"
              @click="applyAiResidentialRoutingChanges"
            >
              <template #icon><ClusterOutlined /></template>
              Apply AI Routing
            </AButton>
          </ASpace>
        </template>
        <ATable
          :columns="residentialIpColumns"
          :data-source="residentialIpRows"
          :pagination="false"
          row-key="key"
          size="middle"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'routed'">
              {{ record.routed ? 'AI' : '-' }}
            </template>
            <template v-if="column.key === 'action'">
              <ASpace wrap>
                <AButton
                  size="small"
                  :loading="testingResidentialIpKey === record.key"
                  @click="testResidentialIpOutbound(record.key)"
                >
                  Test
                </AButton>
                <AButton size="small" @click="openResidentialIpModal(record.key)">
                  <template #icon><EditOutlined /></template>
                  Edit
                </AButton>
                <AButton size="small" danger @click="confirmDeleteOutbound(record.key)">
                  <template #icon><DeleteOutlined /></template>
                  Delete
                </AButton>
              </ASpace>
            </template>
          </template>
        </ATable>
      </FormSection>
    </ACard>

    <ACard class="work-panel" :bordered="false">
      <FormSection eyebrow="Structured" title="Outbounds">
        <template #actions>
          <AButton type="primary" @click="openOutboundModal()">
            <template #icon><PlusOutlined /></template>
            Add Outbound
          </AButton>
        </template>
        <ATable :columns="outboundColumns" :data-source="outboundRows" :pagination="false" row-key="key" size="middle">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'action'">
              <ASpace wrap>
                <AButton size="small" :disabled="record.key === 0" @click="setFirstOutbound(record.key)">
                  <template #icon><VerticalAlignTopOutlined /></template>
                  First
                </AButton>
                <AButton size="small" @click="openOutboundModal(record.key)">
                  <template #icon><EditOutlined /></template>
                  Edit
                </AButton>
                <AButton size="small" danger @click="confirmDeleteOutbound(record.key)">
                  <template #icon><DeleteOutlined /></template>
                  Delete
                </AButton>
              </ASpace>
            </template>
          </template>
        </ATable>
      </FormSection>
    </ACard>

    <ACard class="work-panel" :bordered="false">
      <FormSection eyebrow="Structured" title="Routing Rules">
        <template #actions>
          <AButton type="primary" @click="openRoutingRuleModal()">
            <template #icon><PlusOutlined /></template>
            Add Rule
          </AButton>
        </template>
        <ATable :columns="routingColumns" :data-source="routingRuleRows" :pagination="false" row-key="key" size="middle">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'action'">
              <ASpace wrap>
                <AButton size="small" :disabled="record.key === 0" @click="moveRoutingRule(record.key, record.key - 1)">
                  Up
                </AButton>
                <AButton
                  size="small"
                  :disabled="record.key === routingRuleRows.length - 1"
                  @click="moveRoutingRule(record.key, record.key + 1)"
                >
                  Down
                </AButton>
                <AButton size="small" @click="openRoutingRuleModal(record.key)">
                  <template #icon><EditOutlined /></template>
                  Edit
                </AButton>
                <AButton size="small" danger @click="confirmDeleteRoutingRule(record.key)">
                  <template #icon><DeleteOutlined /></template>
                  Delete
                </AButton>
              </ASpace>
            </template>
          </template>
        </ATable>
      </FormSection>
    </ACard>

    <ACard class="work-panel" :bordered="false">
      <FormSection eyebrow="Structured" title="DNS Servers">
        <template #actions>
          <AButton type="primary" @click="openDnsServerModal()">
            <template #icon><PlusOutlined /></template>
            Add DNS Server
          </AButton>
        </template>
        <ATable :columns="dnsColumns" :data-source="dnsServerRows" :pagination="false" row-key="key" size="middle">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'action'">
              <ASpace wrap>
                <AButton size="small" @click="openDnsServerModal(record.key)">
                  <template #icon><EditOutlined /></template>
                  Edit
                </AButton>
                <AButton size="small" danger @click="confirmDeleteDnsServer(record.key)">
                  <template #icon><DeleteOutlined /></template>
                  Delete
                </AButton>
              </ASpace>
            </template>
          </template>
        </ATable>
      </FormSection>
    </ACard>

    <ACard class="work-panel" :bordered="false">
      <FormSection eyebrow="Structured" title="FakeDNS Pools">
        <template #actions>
          <AButton type="primary" @click="openFakeDnsModal()">
            <template #icon><PlusOutlined /></template>
            Add FakeDNS
          </AButton>
        </template>
        <ATable :columns="fakeDnsColumns" :data-source="fakeDnsRows" :pagination="false" row-key="key" size="middle">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'action'">
              <ASpace wrap>
                <AButton size="small" @click="openFakeDnsModal(record.key)">
                  <template #icon><EditOutlined /></template>
                  Edit
                </AButton>
                <AButton size="small" danger @click="confirmDeleteFakeDns(record.key)">
                  <template #icon><DeleteOutlined /></template>
                  Delete
                </AButton>
              </ASpace>
            </template>
          </template>
        </ATable>
      </FormSection>
    </ACard>

    <ACard class="work-panel" :bordered="false">
      <FormSection eyebrow="Structured" title="Balancers">
        <template #actions>
          <AButton type="primary" @click="openBalancerModal()">
            <template #icon><ClusterOutlined /></template>
            Add Balancer
          </AButton>
        </template>
        <ATable :columns="balancerColumns" :data-source="balancerRows" :pagination="false" row-key="key" size="middle">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'action'">
              <ASpace wrap>
                <AButton size="small" @click="openBalancerModal(record.key)">
                  <template #icon><EditOutlined /></template>
                  Edit
                </AButton>
                <AButton size="small" danger @click="confirmDeleteBalancer(record.key)">
                  <template #icon><DeleteOutlined /></template>
                  Delete
                </AButton>
              </ASpace>
            </template>
          </template>
        </ATable>
      </FormSection>
    </ACard>

    <ACard class="work-panel" :bordered="false">
      <FormSection eyebrow="Structured" title="Reverse">
        <template #actions>
          <AButton type="primary" @click="openReverseModal()">
            <template #icon><ApartmentOutlined /></template>
            Add Reverse
          </AButton>
        </template>
        <ATable :columns="reverseColumns" :data-source="reverseRows" :pagination="false" row-key="key" size="middle">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'action'">
              <ASpace wrap>
                <AButton size="small" @click="openReverseModal(record.key)">
                  <template #icon><EditOutlined /></template>
                  Edit
                </AButton>
                <AButton size="small" danger @click="confirmDeleteReverse(record.key)">
                  <template #icon><DeleteOutlined /></template>
                  Delete
                </AButton>
              </ASpace>
            </template>
          </template>
        </ATable>
      </FormSection>
    </ACard>

    <ACard class="work-panel" :bordered="false">
      <FormSection eyebrow="Structured" title="Protocol Tools">
        <template #actions>
          <AButton @click="generateProtocolToolOutput">
            <template #icon><PlusOutlined /></template>
            Generate
          </AButton>
          <AButton :disabled="!protocolToolGenerated" @click="copyProtocolToolOutput">
            <template #icon><CopyOutlined /></template>
            Copy
          </AButton>
          <AButton :disabled="!canApplyProtocolToolOutbound" type="primary" @click="applyProtocolToolOutbound">
            Add Outbound
          </AButton>
        </template>
        <AForm layout="vertical">
          <AFormItem label="Mode">
            <ASelect v-model:value="protocolToolMode" :options="[{ label: 'Combo', value: 'combo' }, { label: 'Argo', value: 'argo' }]" />
          </AFormItem>
          <template v-if="protocolToolMode === 'combo'">
            <div class="form-grid">
              <AFormItem label="Combo">
                <ASelect v-model:value="protocolTool.combo" :options="protocolToolRows.map((row) => ({ label: row.label, value: row.value }))" />
              </AFormItem>
              <AFormItem label="Tag"><AInput v-model:value="protocolTool.tag" /></AFormItem>
              <AFormItem label="Server"><AInput v-model:value="protocolTool.server" /></AFormItem>
              <AFormItem label="Port"><AInputNumber v-model:value="protocolTool.port" class="full-width" :min="1" :max="65535" /></AFormItem>
              <AFormItem label="UUID"><AInput v-model:value="protocolTool.uuid" /></AFormItem>
              <AFormItem label="Password"><AInput v-model:value="protocolTool.password" /></AFormItem>
              <AFormItem label="SNI"><AInput v-model:value="protocolTool.sni" /></AFormItem>
              <AFormItem label="Public Key"><AInput v-model:value="protocolTool.publicKey" /></AFormItem>
              <AFormItem label="Short ID"><AInput v-model:value="protocolTool.shortId" /></AFormItem>
              <AFormItem label="Path"><AInput v-model:value="protocolTool.path" /></AFormItem>
            </div>
          </template>
          <template v-else>
            <div class="form-grid">
              <AFormItem label="Tunnel Mode">
                <ASelect v-model:value="protocolTool.combo" :options="[{ label: 'Quick', value: 'quick' }, { label: 'Fixed', value: 'fixed' }]" />
              </AFormItem>
              <AFormItem label="Origin URL"><AInput v-model:value="protocolTool.originUrl" /></AFormItem>
              <AFormItem v-if="protocolTool.combo === 'fixed'" label="Tunnel Name"><AInput v-model:value="protocolTool.tunnelName" /></AFormItem>
              <AFormItem v-if="protocolTool.combo === 'fixed'" label="Tunnel Token"><AInput v-model:value="protocolTool.token" /></AFormItem>
            </div>
          </template>
        </AForm>
        <ATable :columns="[{ title: 'Combo', dataIndex: 'label', key: 'label' }, { title: 'Runtime', dataIndex: 'runtime', key: 'runtime' }, { title: 'Scope', dataIndex: 'saveToXray', key: 'saveToXray' }]" :data-source="protocolToolRows" :pagination="false" row-key="value" size="small" />
        <textarea v-model="protocolToolGenerated" class="json-editor compact-json-editor mt-16" readonly />
      </FormSection>
    </ACard>

    <ACard class="work-panel" :bordered="false">
      <FormSection eyebrow="Structured" title="WARP Matrix">
        <template #actions>
          <AButton :loading="loadingWarpMatrix" @click="loadWarpMatrixConfig">
            <template #icon><ReloadOutlined /></template>
            Load WARP
          </AButton>
          <AButton :loading="loadingWarpMatrix" type="primary" @click="applyWarpMatrix">
            Apply Matrix
          </AButton>
        </template>
        <AAlert
          :message="
            warpDataRaw && warpConfigRaw
              ? 'WARP data and config loaded. Matrix can now be applied.'
              : 'Load WARP data and config before applying the matrix.'
          "
          show-icon
          type="info"
        />
        <AFormItem class="mt-16" label="Matrix Options">
          <ASelect
            v-model:value="warpMatrixSelected"
            mode="multiple"
            :options="WARP_MATRIX_OPTIONS.map((item) => ({ label: item.label, value: item.tag }))"
          />
        </AFormItem>
      </FormSection>
    </ACard>

    <ACard class="work-panel" :bordered="false">
      <FormSection eyebrow="Structured" title="DNS Presets">
        <template #actions>
          <ASpace wrap>
            <AButton
              v-for="preset in DNS_PRESET_OPTIONS"
              :key="preset.name"
              size="small"
              @click="applyDnsPresetOption(preset.data)"
            >
              {{ preset.name }}
            </AButton>
          </ASpace>
        </template>
        <AAlert
          message="Install a DNS preset into dns.servers while preserving the current DNS policy."
          show-icon
          type="info"
        />
        <div class="client-link-grid mt-16">
          <div v-for="preset in DNS_PRESET_OPTIONS" :key="`${preset.name}-card`" class="client-link-card">
            <div class="client-link-title-row">
              <div>
                <strong>{{ preset.name }}</strong>
                <p class="muted-text">{{ preset.family ? 'Family profile' : 'Standard profile' }}</p>
              </div>
              <span class="client-link-format">{{ preset.data.length }} servers</span>
            </div>
            <pre class="code-preview compact-preview">{{ preset.data.join('\n') }}</pre>
          </div>
        </div>
      </FormSection>
    </ACard>

    <ACard class="work-panel" :bordered="false">
      <FormSection eyebrow="Structured" title="DNS Policy">
        <template #actions>
          <AButton type="primary" @click="applyDnsPolicyChanges">Apply DNS Policy</AButton>
        </template>
        <div class="form-grid">
          <AFormItem label="Enable DNS"><ASwitch v-model:checked="dnsPolicyForm.enableDNS" /></AFormItem>
          <AFormItem label="Tag"><AInput v-model:value="dnsPolicyForm.dnsTag" /></AFormItem>
          <AFormItem label="Client IP"><AInput v-model:value="dnsPolicyForm.dnsClientIp" /></AFormItem>
          <AFormItem label="Strategy">
            <ASelect v-model:value="dnsPolicyForm.dnsStrategy" :options="['UseSystem', 'UseIP', 'UseIPv4', 'UseIPv6'].map((value) => ({ label: value, value }))" />
          </AFormItem>
          <AFormItem label="Disable Cache"><ASwitch v-model:checked="dnsPolicyForm.dnsDisableCache" /></AFormItem>
          <AFormItem label="Disable Fallback"><ASwitch v-model:checked="dnsPolicyForm.dnsDisableFallback" /></AFormItem>
          <AFormItem label="Disable Fallback If Match"><ASwitch v-model:checked="dnsPolicyForm.dnsDisableFallbackIfMatch" /></AFormItem>
          <AFormItem label="Enable Parallel Query"><ASwitch v-model:checked="dnsPolicyForm.dnsEnableParallelQuery" /></AFormItem>
          <AFormItem label="Use System Hosts"><ASwitch v-model:checked="dnsPolicyForm.dnsUseSystemHosts" /></AFormItem>
        </div>
      </FormSection>
    </ACard>

    <ACard class="work-panel" :bordered="false">
      <FormSection eyebrow="Structured" title="Runtime Policy">
        <template #actions>
          <AButton type="primary" @click="applyRuntimePolicyChanges">Apply Runtime Policy</AButton>
        </template>
        <div class="form-grid">
          <AFormItem label="Freedom Strategy">
            <ASelect v-model:value="runtimePolicyForm.freedomStrategy" :options="['AsIs', 'UseIP', 'UseIPv4', 'UseIPv6'].map((value) => ({ label: value, value }))" />
          </AFormItem>
          <AFormItem label="Routing Strategy">
            <ASelect v-model:value="runtimePolicyForm.routingStrategy" :options="['AsIs', 'IPIfNonMatch', 'IPOnDemand'].map((value) => ({ label: value, value }))" />
          </AFormItem>
          <AFormItem label="Log Level">
            <ASelect v-model:value="runtimePolicyForm.logLevel" :options="['none', 'debug', 'info', 'warning', 'error'].map((value) => ({ label: value, value }))" />
          </AFormItem>
          <AFormItem label="Mask Address">
            <ASelect v-model:value="runtimePolicyForm.maskAddressLog" :options="['', 'quarter', 'half', 'full'].map((value) => ({ label: value || 'Empty', value }))" />
          </AFormItem>
          <AFormItem label="Access Log">
            <ASelect v-model:value="runtimePolicyForm.accessLog" :options="['', 'none', './access.log'].map((value) => ({ label: value || 'Empty', value }))" />
          </AFormItem>
          <AFormItem label="Error Log">
            <ASelect v-model:value="runtimePolicyForm.errorLog" :options="['', 'none', './error.log'].map((value) => ({ label: value || 'Empty', value }))" />
          </AFormItem>
          <AFormItem label="DNS Log"><ASwitch v-model:checked="runtimePolicyForm.dnsLog" /></AFormItem>
          <AFormItem label="Stats Inbound Uplink"><ASwitch v-model:checked="runtimePolicyForm.statsInboundUplink" /></AFormItem>
          <AFormItem label="Stats Inbound Downlink"><ASwitch v-model:checked="runtimePolicyForm.statsInboundDownlink" /></AFormItem>
          <AFormItem label="Stats Outbound Uplink"><ASwitch v-model:checked="runtimePolicyForm.statsOutboundUplink" /></AFormItem>
          <AFormItem label="Stats Outbound Downlink"><ASwitch v-model:checked="runtimePolicyForm.statsOutboundDownlink" /></AFormItem>
        </div>
      </FormSection>
    </ACard>

    <ACard class="work-panel" :bordered="false">
      <FormSection eyebrow="Structured" title="Observatory">
        <template #actions>
          <AButton type="primary" @click="applyObservatoryChanges">Apply Observatory</AButton>
        </template>
        <div class="form-grid">
          <AFormItem label="Enable Observatory">
            <ASwitch v-model:checked="observatoryForm.observatoryEnable" />
          </AFormItem>
          <AFormItem label="Enable Burst Observatory">
            <ASwitch v-model:checked="observatoryForm.burstObservatoryEnable" />
          </AFormItem>
        </div>
        <AFormItem label="Observatory JSON">
          <textarea v-model="observatoryForm.observatoryJson" class="json-editor compact-json-editor" />
        </AFormItem>
        <AFormItem label="Burst Observatory JSON">
          <textarea v-model="observatoryForm.burstObservatoryJson" class="json-editor compact-json-editor" />
        </AFormItem>
      </FormSection>
    </ACard>

    <ACard class="work-panel" :bordered="false">
      <ATable
        :columns="sectionColumns"
        :data-source="advancedRows"
        :pagination="false"
        row-key="key"
        size="middle"
      />
    </ACard>

    <Modal v-model:open="outboundModalOpen" title="Outbound Editor" @ok="submitOutboundModal">
      <AForm layout="vertical">
        <AFormItem label="Tag"><AInput v-model:value="outboundEditor.tag" /></AFormItem>
        <AFormItem label="Protocol"><AInput v-model:value="outboundEditor.protocol" /></AFormItem>
        <AFormItem label="Send Through"><AInput v-model:value="outboundEditor.sendThrough" /></AFormItem>
        <AFormItem label="Settings JSON"><textarea v-model="outboundEditor.settingsJson" class="json-editor compact-json-editor" /></AFormItem>
        <AFormItem label="Stream Settings JSON"><textarea v-model="outboundEditor.streamSettingsJson" class="json-editor compact-json-editor" /></AFormItem>
        <AFormItem label="Proxy Settings JSON"><textarea v-model="outboundEditor.proxySettingsJson" class="json-editor compact-json-editor" /></AFormItem>
        <AFormItem label="Mux JSON"><textarea v-model="outboundEditor.muxJson" class="json-editor compact-json-editor" /></AFormItem>
      </AForm>
    </Modal>

    <Modal
      v-model:open="residentialIpModalOpen"
      title="Residential SOCKS5 Editor"
      @ok="submitResidentialIpModal"
    >
      <AForm layout="vertical">
        <div class="form-grid">
          <AFormItem label="Tag"><AInput v-model:value="residentialIpEditor.tag" /></AFormItem>
          <AFormItem label="Protocol">
            <ASelect
              v-model:value="residentialIpEditor.protocol"
              :options="[{ label: 'SOCKS5', value: 'socks' }]"
            />
          </AFormItem>
          <AFormItem label="Server"><AInput v-model:value="residentialIpEditor.server" /></AFormItem>
          <AFormItem label="Port">
            <AInputNumber
              v-model:value="residentialIpEditor.port"
              class="full-width"
              :min="1"
              :max="65535"
            />
          </AFormItem>
          <AFormItem label="Username">
            <AInput v-model:value="residentialIpEditor.username" />
          </AFormItem>
          <AFormItem label="Password">
            <AInput v-model:value="residentialIpEditor.password" type="password" />
          </AFormItem>
        </div>
      </AForm>
    </Modal>

    <Modal v-model:open="routingRuleModalOpen" title="Routing Rule Editor" @ok="submitRoutingRuleModal">
      <AForm layout="vertical">
        <div class="form-grid">
          <AFormItem label="Type"><AInput v-model:value="routingRuleEditor.type" /></AFormItem>
          <AFormItem label="Outbound Tag"><AInput v-model:value="routingRuleEditor.outboundTag" /></AFormItem>
          <AFormItem label="Balancer Tag"><AInput v-model:value="routingRuleEditor.balancerTag" /></AFormItem>
          <AFormItem label="Network"><AInput v-model:value="routingRuleEditor.networkText" placeholder="tcp,udp" /></AFormItem>
          <AFormItem label="Port"><AInput v-model:value="routingRuleEditor.portText" /></AFormItem>
          <AFormItem label="Source Port"><AInput v-model:value="routingRuleEditor.sourcePortText" /></AFormItem>
        </div>
        <AFormItem label="Domain"><textarea v-model="routingRuleEditor.domainText" class="json-editor compact-json-editor" /></AFormItem>
        <AFormItem label="IP"><textarea v-model="routingRuleEditor.ipText" class="json-editor compact-json-editor" /></AFormItem>
        <AFormItem label="Source"><textarea v-model="routingRuleEditor.sourceText" class="json-editor compact-json-editor" /></AFormItem>
        <AFormItem label="Inbound Tags"><textarea v-model="routingRuleEditor.inboundTagText" class="json-editor compact-json-editor" /></AFormItem>
        <AFormItem label="Users"><textarea v-model="routingRuleEditor.userText" class="json-editor compact-json-editor" /></AFormItem>
        <AFormItem label="Protocols"><textarea v-model="routingRuleEditor.protocolText" class="json-editor compact-json-editor" /></AFormItem>
        <AFormItem label="Attrs JSON"><textarea v-model="routingRuleEditor.attrsJson" class="json-editor compact-json-editor" /></AFormItem>
      </AForm>
    </Modal>

    <Modal v-model:open="dnsServerModalOpen" title="DNS Server Editor" @ok="submitDnsServerModal">
      <AForm layout="vertical">
        <AFormItem label="Address"><AInput v-model:value="dnsServerEditor.address" /></AFormItem>
        <AFormItem label="Domains"><textarea v-model="dnsServerEditor.domainsText" class="json-editor compact-json-editor" /></AFormItem>
        <AFormItem label="Expect IPs"><textarea v-model="dnsServerEditor.expectIPsText" class="json-editor compact-json-editor" /></AFormItem>
        <div class="form-grid">
          <AFormItem label="Skip Fallback"><ASwitch v-model:checked="dnsServerEditor.skipFallback" /></AFormItem>
          <AFormItem label="Client IP"><AInput v-model:value="dnsServerEditor.clientIP" /></AFormItem>
          <AFormItem label="Query Strategy"><AInput v-model:value="dnsServerEditor.queryStrategy" /></AFormItem>
        </div>
      </AForm>
    </Modal>

    <Modal v-model:open="fakeDnsModalOpen" title="FakeDNS Editor" @ok="submitFakeDnsModal">
      <AForm layout="vertical">
        <AFormItem label="IP Pool"><AInput v-model:value="fakeDnsEditor.ipPool" /></AFormItem>
        <AFormItem label="Pool Size"><AInputNumber v-model:value="fakeDnsEditor.poolSize" class="full-width" :min="0" /></AFormItem>
      </AForm>
    </Modal>

    <Modal v-model:open="balancerModalOpen" title="Balancer Editor" @ok="submitBalancerModal">
      <AForm layout="vertical">
        <AFormItem label="Tag"><AInput v-model:value="balancerEditor.tag" /></AFormItem>
        <AFormItem label="Strategy"><AInput v-model:value="balancerEditor.strategy" /></AFormItem>
        <AFormItem label="Selectors"><textarea v-model="balancerEditor.selectorText" class="json-editor compact-json-editor" /></AFormItem>
        <AFormItem label="Fallback Tag"><AInput v-model:value="balancerEditor.fallbackTag" /></AFormItem>
      </AForm>
    </Modal>

    <Modal v-model:open="reverseModalOpen" title="Reverse Editor" @ok="submitReverseModal">
      <AForm layout="vertical">
        <div class="form-grid">
          <AFormItem label="Type"><ASelect v-model:value="reverseEditor.type" :options="[{ label: 'bridge', value: 'bridge' }, { label: 'portal', value: 'portal' }]" /></AFormItem>
          <AFormItem label="Tag"><AInput v-model:value="reverseEditor.tag" /></AFormItem>
          <AFormItem label="Domain"><AInput v-model:value="reverseEditor.domain" /></AFormItem>
          <AFormItem v-if="reverseEditor.type === 'bridge'" label="Bridge Outbound">
            <AInput v-model:value="reverseEditor.bridgeOutboundTag" />
          </AFormItem>
          <AFormItem v-if="reverseEditor.type === 'bridge'" label="Bridge Reply Outbound">
            <AInput v-model:value="reverseEditor.bridgeReplyOutboundTag" />
          </AFormItem>
        </div>
        <AFormItem v-if="reverseEditor.type === 'portal'" label="Portal Inbound Tags">
          <textarea v-model="reverseEditor.portalInboundTagsText" class="json-editor compact-json-editor" />
        </AFormItem>
      </AForm>
    </Modal>
  </section>
</template>

<script setup lang="ts">
import {
  AlignLeftOutlined,
  CopyOutlined,
  DownloadOutlined,
  EditOutlined,
  DeleteOutlined,
  PlusOutlined,
  VerticalAlignTopOutlined,
  ClusterOutlined,
  ApartmentOutlined,
  PauseCircleOutlined,
  PoweroffOutlined,
  ReloadOutlined,
} from '@ant-design/icons-vue';
import {
  Alert as AAlert,
  Button as AButton,
  Card as ACard,
  Form as AForm,
  FormItem as AFormItem,
  Input as AInput,
  InputNumber as AInputNumber,
  Modal,
  Select as ASelect,
  Space as ASpace,
  Switch as ASwitch,
  Table as ATable,
  message,
} from 'ant-design-vue';
import { computed, onMounted, reactive, ref } from 'vue';

import {
  getXrayVersions,
  installXrayVersion,
  startXrayService,
  stopXrayService,
} from '@/api/server';
import {
  getOutboundsTraffic,
  getXrayResult,
  getXraySetting,
  resetOutboundsTraffic,
  runNordAction,
  runWarpAction,
  testOutbound,
  updateXraySetting,
} from '@/api/xray';
import FormSection from '@/components/FormSection.vue';
import PageHeader from '@/components/PageHeader.vue';
import StatusTile from '@/components/StatusTile.vue';
import { useServerStore } from '@/stores/server';
import type { JsonObject, JsonValue } from '@/types/api';
import { hasInjectedRuntimeConfig } from '@/types/runtime';
import type { OutboundTraffic } from '@/types/xray';
import { formatBytes, formatCount } from '@/utils/format';
import type {
  BalancerEditorForm,
  DnsPolicyForm,
  DnsServerEditorForm,
  FakeDnsEditorForm,
  ObservatoryForm,
  OutboundEditorForm,
  ResidentialIpEditorForm,
  ReverseEditorForm,
  RuntimePolicyForm,
  RoutingRuleEditorForm,
} from '@/utils/xrayCompat';
import {
  DNS_PRESET_OPTIONS,
  applyAiResidentialRouting,
  applyDnsPolicyForm,
  applyDnsPreset,
  applyObservatoryForm,
  applyRuntimePolicyForm,
  deleteBalancerAt,
  deleteDnsServerAt,
  deleteFakeDnsAt,
  deleteOutboundAt,
  deleteReverseAt,
  deleteRoutingRuleAt,
  getBalancerRows,
  getDnsPolicyForm,
  getDnsServerRows,
  getFakeDnsRows,
  getObservatoryForm,
  getOutboundRows,
  getResidentialIpRows,
  getReverseRows,
  getRoutingRuleRows,
  getRuntimePolicyForm,
  moveArrayItem,
  upsertResidentialIpOutbound,
  upsertBalancer,
  upsertDnsServer,
  upsertFakeDns,
  upsertOutbound,
  upsertReverse,
  upsertRoutingRule,
} from '@/utils/xrayCompat';
import {
  PROTOCOL_TOOL_PRESETS,
  WARP_MATRIX_OPTIONS,
  applyWarpMatrixToTemplate,
  buildWarpMatrixBaseSettings,
  generateProtocolToolArgo,
  generateProtocolToolCombo,
} from '@/utils/xrayProtocolTools';
import { copyText, downloadText } from '@/utils/textExport';
import {
  DEFAULT_GATEWAY_EGRESS_MVP_NETWORK_STRATEGY,
  buildGatewayEgressManifestCsv,
  buildGatewayEgressMvpPreview,
  mergeGatewayEgressMvpConfig,
  normalizeGatewayEgressMvpNetworkStrategy,
} from '@/utils/gatewayEgressMvp';

const serverStore = useServerStore();
const error = ref('');
const configText = ref('');
const originalConfigText = ref('');
const outboundTestUrl = ref('');
const lifecycleBusy = ref(false);
const loadingConfig = ref(false);
const savingConfig = ref(false);
const loadingVersions = ref(false);
const installingVersion = ref(false);
const availableVersions = ref<string[]>([]);
const selectedVersion = ref<string>();
const xrayResult = ref('');
const outboundsTraffic = ref<OutboundTraffic[]>([]);
const loadingOutboundsTraffic = ref(false);
const testingOutbound = ref(false);
const resettingOutboundTraffic = ref(false);
const resettingOutboundTag = ref('');
const outboundToolResult = ref('');
const providerAction = ref('');
const outboundModalOpen = ref(false);
const residentialIpModalOpen = ref(false);
const routingRuleModalOpen = ref(false);
const dnsServerModalOpen = ref(false);
const fakeDnsModalOpen = ref(false);
const balancerModalOpen = ref(false);
const reverseModalOpen = ref(false);
const testingResidentialIpKey = ref<number | null>(null);
const editingOutboundIndex = ref<number | null>(null);
const editingResidentialIpOutboundIndex = ref<number | null>(null);
const editingRoutingRuleIndex = ref<number | null>(null);
const editingDnsServerIndex = ref<number | null>(null);
const editingFakeDnsIndex = ref<number | null>(null);
const editingBalancerIndex = ref<number | null>(null);
const editingReverseIndex = ref<number | null>(null);
const outboundEditor = reactive<OutboundEditorForm>(createOutboundEditor());
const residentialIpEditor = reactive<ResidentialIpEditorForm>(createResidentialIpEditor());
const routingRuleEditor = reactive<RoutingRuleEditorForm>(createRoutingRuleEditor());
const dnsServerEditor = reactive<DnsServerEditorForm>(createDnsServerEditor());
const fakeDnsEditor = reactive<FakeDnsEditorForm>(createFakeDnsEditor());
const balancerEditor = reactive<BalancerEditorForm>(createBalancerEditor());
const reverseEditor = reactive<ReverseEditorForm>(createReverseEditor());
const protocolToolMode = ref<'combo' | 'argo'>('combo');
const protocolToolGenerated = ref('');
const protocolToolLastResult = ref<ReturnType<typeof generateProtocolToolCombo> | ReturnType<typeof generateProtocolToolArgo> | null>(null);
const protocolTool = reactive({
  combo: 'vless-reality-vision',
  server: 'example.com',
  port: 443,
  uuid: '11111111-1111-4111-8111-111111111111',
  password: 'change-me',
  sni: 'www.microsoft.com',
  publicKey: '',
  shortId: '',
  path: '/xhttp',
  tag: 'proxy',
  originUrl: 'http://localhost:2053',
  tunnelName: 'superxray',
  token: '',
});
const warpMatrixSelected = ref<string[]>(['warp']);
const loadingWarpMatrix = ref(false);
const warpDataRaw = ref<Record<string, unknown> | null>(null);
const warpConfigRaw = ref<Record<string, unknown> | null>(null);
const dnsPolicyForm = reactive<DnsPolicyForm>(createDnsPolicyForm());
const runtimePolicyForm = reactive<RuntimePolicyForm>(createRuntimePolicyForm());
const observatoryForm = reactive<ObservatoryForm>(createObservatoryForm());
const gatewayEgressNetwork = reactive({ ...DEFAULT_GATEWAY_EGRESS_MVP_NETWORK_STRATEGY });

const status = computed(() => serverStore.status);
const refreshing = computed(
  () => serverStore.loadingStatus || loadingConfig.value || loadingOutboundsTraffic.value,
);
const currentVersion = computed(() => status.value?.xray.version || '-');
const xrayStateLabel = computed(() => {
  const state = status.value?.xray.state;
  return state ? state.charAt(0).toUpperCase() + state.slice(1) : '-';
});
const xrayStateTone = computed<'danger' | 'neutral' | 'success' | 'warning'>(() => {
  switch (status.value?.xray.state) {
    case 'running':
      return 'success';
    case 'error':
      return 'danger';
    case 'stop':
      return 'warning';
    default:
      return 'neutral';
  }
});
const xrayErrorHint = computed(() => status.value?.xray.errorMsg || 'Existing Xray service');
const startRestartLabel = computed(() =>
  status.value?.xray.state === 'stop' ? 'Start' : 'Restart',
);
const configChanged = computed(
  () =>
    configText.value !== originalConfigText.value ||
    outboundTestUrl.value !== loadedOutboundTestUrl.value,
);
const configChangedHint = computed(() =>
  configChanged.value ? 'Unsaved changes' : 'Legacy-compatible',
);
const templateState = computed(() => (parsedConfig.value ? 'Valid JSON' : 'Invalid JSON'));
const runtimeResult = computed(() => {
  const lines = [
    `State: ${status.value?.xray.state || '-'}`,
    `Version: ${status.value?.xray.version || '-'}`,
    status.value?.xray.errorMsg ? `Status error: ${status.value.xray.errorMsg}` : '',
    xrayResult.value ? `Result: ${xrayResult.value}` : '',
  ].filter(Boolean);
  return lines.join('\n') || 'No runtime result.';
});
const versionOptions = computed(() =>
  availableVersions.value.map((version) => ({
    label: version,
    value: version,
  })),
);
const parsedConfig = computed<JsonValue | null>(() => parseJson(configText.value));
const outboundsFromConfig = computed<JsonObject[]>(() => {
  const config = parsedConfig.value;
  if (!isJsonObject(config) || !Array.isArray(config.outbounds)) {
    return [];
  }
  return config.outbounds.filter(isJsonObject);
});
const advancedRows = computed(() => [
  {
    key: 'outbounds',
    section: 'Outbounds',
    value: sectionArrayCount(parsedConfig.value, 'outbounds'),
    detail: 'Legacy outbound array in Xray template',
  },
  {
    key: 'routing',
    section: 'Routing',
    value: nestedArrayCount(parsedConfig.value, 'routing', 'rules'),
    detail: `${nestedArrayCount(parsedConfig.value, 'routing', 'balancers')} balancers`,
  },
  {
    key: 'dns',
    section: 'DNS',
    value: objectState(parsedConfig.value, 'dns'),
    detail: 'Raw DNS object remains editable in JSON',
  },
  {
    key: 'reverse',
    section: 'Reverse',
    value: objectState(parsedConfig.value, 'reverse'),
    detail: 'Reverse bridge and portal settings',
  },
  {
    key: 'fakedns',
    section: 'FakeDNS',
    value: sectionArrayCount(parsedConfig.value, 'fakedns'),
    detail: 'Top-level fakedns array when present',
  },
]);
const sectionColumns = [
  { title: 'Section', dataIndex: 'section', key: 'section' },
  { title: 'State', dataIndex: 'value', key: 'value' },
  { title: 'Detail', dataIndex: 'detail', key: 'detail' },
];
const outboundTrafficColumns = [
  { title: 'Tag', dataIndex: 'tag', key: 'tag' },
  { title: 'Up', dataIndex: 'upText', key: 'upText' },
  { title: 'Down', dataIndex: 'downText', key: 'downText' },
  { title: 'Total', dataIndex: 'totalText', key: 'totalText' },
  { title: 'Actions', key: 'action', width: 120 },
];
const outboundColumns = [
  { title: 'Tag', dataIndex: 'tag', key: 'tag' },
  { title: 'Protocol', dataIndex: 'protocol', key: 'protocol', width: 120 },
  { title: 'Address', dataIndex: 'address', key: 'address' },
  { title: 'Send Through', dataIndex: 'sendThrough', key: 'sendThrough', width: 140 },
  { title: 'Actions', key: 'action', width: 260 },
];
const residentialIpColumns = [
  { title: 'Tag', dataIndex: 'tag', key: 'tag' },
  { title: 'Protocol', dataIndex: 'protocol', key: 'protocol', width: 120 },
  { title: 'Address', dataIndex: 'address', key: 'address' },
  { title: 'Routing', key: 'routed', width: 100 },
  { title: 'Actions', key: 'action', width: 260 },
];
const routingColumns = [
  { title: 'Type', dataIndex: 'type', key: 'type', width: 90 },
  { title: 'Outbound', dataIndex: 'outboundTag', key: 'outboundTag', width: 140 },
  { title: 'Balancer', dataIndex: 'balancerTag', key: 'balancerTag', width: 140 },
  { title: 'Domain', dataIndex: 'domainText', key: 'domainText' },
  { title: 'Actions', key: 'action', width: 260 },
];
const dnsColumns = [
  { title: 'Address', dataIndex: 'address', key: 'address' },
  { title: 'Domains', dataIndex: 'domainsText', key: 'domainsText' },
  { title: 'Expect IPs', dataIndex: 'expectIPsText', key: 'expectIPsText' },
  { title: 'Actions', key: 'action', width: 220 },
];
const fakeDnsColumns = [
  { title: 'IP Pool', dataIndex: 'ipPool', key: 'ipPool' },
  { title: 'Pool Size', dataIndex: 'poolSize', key: 'poolSize', width: 140 },
  { title: 'Actions', key: 'action', width: 220 },
];
const balancerColumns = [
  { title: 'Tag', dataIndex: 'tag', key: 'tag' },
  { title: 'Strategy', dataIndex: 'strategy', key: 'strategy', width: 140 },
  { title: 'Selectors', dataIndex: 'selectorText', key: 'selectorText' },
  { title: 'Fallback', dataIndex: 'fallbackTag', key: 'fallbackTag', width: 140 },
  { title: 'Actions', key: 'action', width: 220 },
];
const reverseColumns = [
  { title: 'Type', dataIndex: 'type', key: 'type', width: 120 },
  { title: 'Tag', dataIndex: 'tag', key: 'tag', width: 140 },
  { title: 'Domain', dataIndex: 'domain', key: 'domain' },
  { title: 'Actions', key: 'action', width: 220 },
];
const outboundTrafficRows = computed(() => {
  const trafficRows = outboundsTraffic.value.map((traffic) => ({
    ...traffic,
    downText: formatBytes(traffic.down),
    totalText: formatBytes(traffic.total),
    upText: formatBytes(traffic.up),
  }));
  if (trafficRows.length > 0) {
    return trafficRows;
  }
  return outboundsFromConfig.value.map((outbound, index) => {
    const tag = typeof outbound.tag === 'string' ? outbound.tag : `outbound-${index + 1}`;
    return {
      down: 0,
      downText: '-',
      id: index,
      tag,
      total: 0,
      totalText: '-',
      up: 0,
      upText: '-',
    };
  });
});
const outboundRows = computed(() => getOutboundRows(parsedConfig.value));
const residentialIpRows = computed(() => getResidentialIpRows(parsedConfig.value));
const routingRuleRows = computed(() => getRoutingRuleRows(parsedConfig.value));
const dnsServerRows = computed(() => getDnsServerRows(parsedConfig.value));
const fakeDnsRows = computed(() => getFakeDnsRows(parsedConfig.value));
const balancerRows = computed(() => getBalancerRows(parsedConfig.value));
const reverseRows = computed(() => getReverseRows(parsedConfig.value));
const gatewayEgressNetworkError = computed(() => {
  try {
    normalizeGatewayEgressMvpNetworkStrategy(gatewayEgressNetwork);
    return '';
  } catch (caught) {
    return caught instanceof Error ? caught.message : 'Invalid Gateway egress network strategy';
  }
});
const gatewayEgressMvpPreview = computed(() => {
  try {
    return buildGatewayEgressMvpPreview(gatewayEgressNetwork);
  } catch {
    return buildGatewayEgressMvpPreview(DEFAULT_GATEWAY_EGRESS_MVP_NETWORK_STRATEGY);
  }
});
const gatewayEgressManifestCsv = computed(() =>
  gatewayEgressNetworkError.value ? '' : buildGatewayEgressManifestCsv(gatewayEgressNetwork),
);
const protocolToolRows = computed(() => PROTOCOL_TOOL_PRESETS);
const canApplyProtocolToolOutbound = computed(() =>
  Boolean(protocolToolLastResult.value?.saveToXray && protocolToolLastResult.value?.clientOutbound),
);

const loadedOutboundTestUrl = ref('');

function createOutboundEditor(): OutboundEditorForm {
  return {
    tag: '',
    protocol: 'freedom',
    sendThrough: '',
    settingsJson: '{}',
    streamSettingsJson: '{}',
    proxySettingsJson: '{}',
    muxJson: '{}',
  };
}

function createResidentialIpEditor(): ResidentialIpEditorForm {
  return {
    tag: 'residential-us-1',
    protocol: 'socks',
    server: '',
    port: 1080,
    username: '',
    password: '',
  };
}

function createRoutingRuleEditor(): RoutingRuleEditorForm {
  return {
    type: 'field',
    outboundTag: '',
    balancerTag: '',
    domainText: '',
    ipText: '',
    sourceText: '',
    userText: '',
    inboundTagText: '',
    protocolText: '',
    attrsJson: '{}',
    networkText: '',
    portText: '',
    sourcePortText: '',
  };
}

function createDnsServerEditor(): DnsServerEditorForm {
  return {
    address: '',
    domainsText: '',
    expectIPsText: '',
    skipFallback: false,
    clientIP: '',
    queryStrategy: '',
  };
}

function createFakeDnsEditor(): FakeDnsEditorForm {
  return {
    ipPool: '198.18.0.0/15',
    poolSize: 65535,
  };
}

function createBalancerEditor(): BalancerEditorForm {
  return {
    tag: '',
    strategy: 'random',
    selectorText: '',
    fallbackTag: '',
  };
}

function createReverseEditor(): ReverseEditorForm {
  return {
    type: 'bridge',
    tag: 'reverse-0',
    domain: 'reverse.xui',
    bridgeOutboundTag: '',
    bridgeReplyOutboundTag: '',
    portalInboundTagsText: '',
  };
}

function createDnsPolicyForm(): DnsPolicyForm {
  return {
    enableDNS: false,
    dnsTag: '',
    dnsClientIp: '',
    dnsStrategy: 'UseIP',
    dnsDisableCache: false,
    dnsDisableFallback: false,
    dnsDisableFallbackIfMatch: false,
    dnsEnableParallelQuery: false,
    dnsUseSystemHosts: false,
  };
}

function createRuntimePolicyForm(): RuntimePolicyForm {
  return {
    freedomStrategy: 'AsIs',
    routingStrategy: 'AsIs',
    logLevel: 'warning',
    accessLog: '',
    errorLog: '',
    dnsLog: false,
    maskAddressLog: '',
    statsInboundUplink: false,
    statsInboundDownlink: false,
    statsOutboundUplink: false,
    statsOutboundDownlink: false,
  };
}

function createObservatoryForm(): ObservatoryForm {
  return {
    observatoryEnable: false,
    observatoryJson: '',
    burstObservatoryEnable: false,
    burstObservatoryJson: '',
  };
}

function applyTemplateConfig(nextTemplate: JsonObject) {
  const formatted = JSON.stringify(nextTemplate, null, 2);
  configText.value = formatted;
  syncStructuredFormsFromConfig(nextTemplate);
}

function applyGatewayEgressMvp() {
  try {
    const merged = mergeGatewayEgressMvpConfig(parsedConfig.value, gatewayEgressNetwork);
    applyTemplateConfig(merged);
    void message.success('Gateway egress MVP config generated. Review and save the Xray template.');
  } catch (caught) {
    const errorMessage =
      caught instanceof Error ? caught.message : 'Failed to generate Gateway egress config';
    error.value = errorMessage;
    void message.error(errorMessage);
  }
}

async function copyGatewayEgressManifest() {
  if (gatewayEgressNetworkError.value) {
    void message.error(gatewayEgressNetworkError.value);
    return;
  }
  await copyText(gatewayEgressManifestCsv.value);
  void message.success('Gateway egress manifest copied');
}

function downloadGatewayEgressManifest() {
  if (gatewayEgressNetworkError.value) {
    void message.error(gatewayEgressNetworkError.value);
    return;
  }
  downloadText('gateway-egress-mvp.csv', gatewayEgressManifestCsv.value, 'text/csv;charset=utf-8');
}

function syncStructuredFormsFromConfig(config: JsonValue | null | undefined) {
  Object.assign(dnsPolicyForm, createDnsPolicyForm(), getDnsPolicyForm(config));
  Object.assign(runtimePolicyForm, createRuntimePolicyForm(), getRuntimePolicyForm(config));
  Object.assign(observatoryForm, createObservatoryForm(), getObservatoryForm(config));
}

async function refreshRuntime() {
  await Promise.all([serverStore.refreshStatus(), refreshXrayResult(), loadOutboundsTraffic()]);
}

async function refreshXrayResult() {
  if (!hasInjectedRuntimeConfig()) {
    return;
  }

  try {
    xrayResult.value = await getXrayResult({ notifyOnError: false });
  } catch {
    xrayResult.value = '';
  }
}

async function loadOutboundsTraffic() {
  if (!hasInjectedRuntimeConfig()) {
    return;
  }

  loadingOutboundsTraffic.value = true;
  try {
    outboundsTraffic.value = await getOutboundsTraffic({ notifyOnError: false });
  } catch {
    outboundsTraffic.value = [];
  } finally {
    loadingOutboundsTraffic.value = false;
  }
}

async function refreshConfig() {
  if (!hasInjectedRuntimeConfig()) {
    return;
  }

  loadingConfig.value = true;
  error.value = '';
  try {
    const payload = await getXraySetting({ notifyOnError: false });
    const formatted = JSON.stringify(payload.xraySetting, null, 2);
    configText.value = formatted;
    originalConfigText.value = formatted;
    outboundTestUrl.value = payload.outboundTestUrl || '';
    loadedOutboundTestUrl.value = payload.outboundTestUrl || '';
    syncStructuredFormsFromConfig(payload.xraySetting);
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to load Xray settings';
  } finally {
    loadingConfig.value = false;
  }
}

function openOutboundModal(index?: number) {
  const row = typeof index === 'number' ? outboundRows.value[index] : null;
  editingOutboundIndex.value = typeof index === 'number' ? index : null;
  Object.assign(outboundEditor, createOutboundEditor(), {
    tag: row?.tag || '',
    protocol: row?.protocol || 'freedom',
    sendThrough: typeof row?.sendThrough === 'string' ? row.sendThrough : '',
    settingsJson: JSON.stringify(row?.settings || {}, null, 2),
    streamSettingsJson: JSON.stringify(row?.streamSettings || {}, null, 2),
    proxySettingsJson: JSON.stringify(row?.proxySettings || {}, null, 2),
    muxJson: JSON.stringify(row?.mux || {}, null, 2),
  });
  outboundModalOpen.value = true;
}

function openResidentialIpModal(index?: number) {
  const outbound = typeof index === 'number' ? outboundsFromConfig.value[index] : null;
  const settings = isJsonObject(outbound?.settings) ? outbound.settings : {};
  const servers = Array.isArray(settings.servers) ? settings.servers : [];
  const server = isJsonObject(servers[0]) ? servers[0] : {};
  const users = Array.isArray(server.users) ? server.users : [];
  const user = isJsonObject(users[0]) ? users[0] : {};
  const defaultEditor = createResidentialIpEditor();

  editingResidentialIpOutboundIndex.value = typeof index === 'number' ? index : null;
  Object.assign(residentialIpEditor, defaultEditor, {
    tag:
      (typeof outbound?.tag === 'string' && outbound.tag) ||
      `residential-us-${residentialIpRows.value.length + 1}`,
    protocol: 'socks',
    server: typeof server.address === 'string' ? server.address : '',
    port: typeof server.port === 'number' ? server.port : Number(server.port) || defaultEditor.port,
    username: typeof user.user === 'string' ? user.user : '',
    password: typeof user.pass === 'string' ? user.pass : '',
  });
  residentialIpModalOpen.value = true;
}

function openRoutingRuleModal(index?: number) {
  const row = typeof index === 'number' ? routingRuleRows.value[index] : null;
  editingRoutingRuleIndex.value = typeof index === 'number' ? index : null;
  Object.assign(routingRuleEditor, createRoutingRuleEditor(), {
    type: row?.type || 'field',
    outboundTag: row?.outboundTag || '',
    balancerTag: row?.balancerTag || '',
    domainText: row?.domainText || '',
    ipText: row?.ipText || '',
    sourceText: row?.sourceText || '',
    userText: row?.userText || '',
    inboundTagText: row?.inboundTagText || '',
    protocolText: row?.protocolText || '',
    attrsJson: JSON.stringify(row?.attrs || {}, null, 2),
    networkText: row?.networkText || '',
    portText: row?.portText || '',
    sourcePortText: row?.sourcePortText || '',
  });
  routingRuleModalOpen.value = true;
}

function openDnsServerModal(index?: number) {
  const row = typeof index === 'number' ? dnsServerRows.value[index] : null;
  editingDnsServerIndex.value = typeof index === 'number' ? index : null;
  Object.assign(dnsServerEditor, createDnsServerEditor(), {
    address: row?.address || '',
    domainsText: row?.domainsText || '',
    expectIPsText: row?.expectIPsText || '',
    skipFallback: row?.skipFallback || false,
    clientIP: row?.clientIP || '',
    queryStrategy: row?.queryStrategy || '',
  });
  dnsServerModalOpen.value = true;
}

function openFakeDnsModal(index?: number) {
  const row = typeof index === 'number' ? fakeDnsRows.value[index] : null;
  editingFakeDnsIndex.value = typeof index === 'number' ? index : null;
  Object.assign(fakeDnsEditor, createFakeDnsEditor(), {
    ipPool: row?.ipPool || '198.18.0.0/15',
    poolSize: typeof row?.poolSize === 'number' ? row.poolSize : 65535,
  });
  fakeDnsModalOpen.value = true;
}

function openBalancerModal(index?: number) {
  const row = typeof index === 'number' ? balancerRows.value[index] : null;
  editingBalancerIndex.value = typeof index === 'number' ? index : null;
  Object.assign(balancerEditor, createBalancerEditor(), {
    tag: row?.tag || '',
    strategy: row?.strategy || 'random',
    selectorText: row?.selectorText || '',
    fallbackTag: row?.fallbackTag || '',
  });
  balancerModalOpen.value = true;
}

function openReverseModal(index?: number) {
  const row = typeof index === 'number' ? reverseRows.value[index] : null;
  editingReverseIndex.value = typeof index === 'number' ? index : null;
  Object.assign(reverseEditor, createReverseEditor(), {
    type: row?.type || 'bridge',
    tag: row?.tag || 'reverse-0',
    domain: row?.domain || 'reverse.xui',
    bridgeOutboundTag: row?.bridgeOutboundTag || '',
    bridgeReplyOutboundTag: row?.bridgeReplyOutboundTag || '',
    portalInboundTagsText: row?.portalInboundTagsText || '',
  });
  reverseModalOpen.value = true;
}

function submitOutboundModal() {
  const current = parsedConfig.value;
  const next = upsertOutbound(current, editingOutboundIndex.value, outboundEditor);
  applyTemplateConfig(next);
  outboundModalOpen.value = false;
}

function submitResidentialIpModal() {
  if (!residentialIpEditor.tag.trim() || !residentialIpEditor.server.trim()) {
    error.value = 'Residential IP tag and server are required';
    return;
  }
  const next = upsertResidentialIpOutbound(
    parsedConfig.value,
    editingResidentialIpOutboundIndex.value,
    residentialIpEditor,
  );
  applyTemplateConfig(next);
  residentialIpModalOpen.value = false;
  void message.success('Residential IP saved to template');
}

function applyAiResidentialRoutingChanges() {
  const next = applyAiResidentialRouting(parsedConfig.value);
  applyTemplateConfig(next);
  void message.success('AI residential routing updated');
}

async function testResidentialIpOutbound(index: number) {
  const outbound = outboundsFromConfig.value[index];
  if (!outbound) {
    return;
  }

  testingResidentialIpKey.value = index;
  outboundToolResult.value = '';
  try {
    const result = await testOutbound(
      JSON.stringify(outbound),
      JSON.stringify(outboundsFromConfig.value),
      { notifyOnError: false },
    );
    outboundToolResult.value = result.success
      ? `Residential IP test succeeded in ${result.delay} ms with status ${result.statusCode || '-'}`
      : `Residential IP test failed: ${result.error || 'unknown error'}`;
  } catch (caught) {
    outboundToolResult.value = caught instanceof Error ? caught.message : 'Residential IP test failed';
  } finally {
    testingResidentialIpKey.value = null;
  }
}

function submitRoutingRuleModal() {
  const current = parsedConfig.value;
  const next = upsertRoutingRule(current, editingRoutingRuleIndex.value, routingRuleEditor);
  applyTemplateConfig(next);
  routingRuleModalOpen.value = false;
}

function submitDnsServerModal() {
  const current = parsedConfig.value;
  const next = upsertDnsServer(current, editingDnsServerIndex.value, dnsServerEditor);
  applyTemplateConfig(next);
  dnsServerModalOpen.value = false;
}

function submitFakeDnsModal() {
  const current = parsedConfig.value;
  const next = upsertFakeDns(current, editingFakeDnsIndex.value, fakeDnsEditor);
  applyTemplateConfig(next);
  fakeDnsModalOpen.value = false;
}

function submitBalancerModal() {
  const current = parsedConfig.value;
  const next = upsertBalancer(current, editingBalancerIndex.value, balancerEditor);
  applyTemplateConfig(next);
  balancerModalOpen.value = false;
}

function submitReverseModal() {
  const current = parsedConfig.value;
  const next = upsertReverse(current, editingReverseIndex.value, reverseEditor);
  applyTemplateConfig(next);
  reverseModalOpen.value = false;
}

function setFirstOutbound(index: number) {
  const config = parsedConfig.value;
  if (!isJsonObject(config) || !Array.isArray(config.outbounds)) {
    return;
  }
  const next = {
    ...config,
    outbounds: moveArrayItem(config.outbounds, index, 0),
  };
  applyTemplateConfig(next);
}

function moveRoutingRule(from: number, to: number) {
  const config = parsedConfig.value;
  if (!isJsonObject(config)) {
    return;
  }
  const routing = isJsonObject(config.routing) ? { ...config.routing } : {};
  const rules = Array.isArray(routing.rules) ? routing.rules : [];
  routing.rules = moveArrayItem(rules, from, to);
  applyTemplateConfig({
    ...config,
    routing,
  });
}

function confirmDeleteOutbound(index: number) {
  Modal.confirm({
    title: 'Delete outbound?',
    okButtonProps: { danger: true },
    onOk: () => {
      applyTemplateConfig(deleteOutboundAt(parsedConfig.value, index));
    },
  });
}

function confirmDeleteRoutingRule(index: number) {
  Modal.confirm({
    title: 'Delete routing rule?',
    okButtonProps: { danger: true },
    onOk: () => {
      applyTemplateConfig(deleteRoutingRuleAt(parsedConfig.value, index));
    },
  });
}

function confirmDeleteDnsServer(index: number) {
  Modal.confirm({
    title: 'Delete DNS server?',
    okButtonProps: { danger: true },
    onOk: () => {
      applyTemplateConfig(deleteDnsServerAt(parsedConfig.value, index));
    },
  });
}

function confirmDeleteFakeDns(index: number) {
  Modal.confirm({
    title: 'Delete FakeDNS pool?',
    okButtonProps: { danger: true },
    onOk: () => {
      applyTemplateConfig(deleteFakeDnsAt(parsedConfig.value, index));
    },
  });
}

function confirmDeleteBalancer(index: number) {
  Modal.confirm({
    title: 'Delete balancer?',
    okButtonProps: { danger: true },
    onOk: () => {
      applyTemplateConfig(deleteBalancerAt(parsedConfig.value, index));
    },
  });
}

function confirmDeleteReverse(index: number) {
  Modal.confirm({
    title: 'Delete reverse entry?',
    okButtonProps: { danger: true },
    onOk: () => {
      applyTemplateConfig(deleteReverseAt(parsedConfig.value, index));
    },
  });
}

function generateProtocolToolOutput() {
  if (protocolToolMode.value === 'argo') {
    const result = generateProtocolToolArgo({
      mode: protocolTool.combo === 'fixed' ? 'fixed' : 'quick',
      originUrl: protocolTool.originUrl,
      tunnelName: protocolTool.tunnelName,
      token: protocolTool.token,
    });
    protocolToolLastResult.value = result;
    protocolToolGenerated.value = JSON.stringify(result, null, 2);
    return;
  }

  const result = generateProtocolToolCombo({
    combo: protocolTool.combo,
    server: protocolTool.server,
    port: protocolTool.port,
    uuid: protocolTool.uuid,
    password: protocolTool.password,
    sni: protocolTool.sni,
    publicKey: protocolTool.publicKey,
    shortId: protocolTool.shortId,
    path: protocolTool.path,
    tag: protocolTool.tag,
  });
  protocolToolLastResult.value = result;
  protocolToolGenerated.value = JSON.stringify(result, null, 2);
}

async function copyProtocolToolOutput() {
  if (!protocolToolGenerated.value) {
    return;
  }
  await copyText(protocolToolGenerated.value);
  void message.success('Protocol tool output copied');
}

function applyProtocolToolOutbound() {
  const result = protocolToolLastResult.value;
  if (!result?.saveToXray || !result.clientOutbound) {
    return;
  }
  const outbound = parseJson(result.clientOutbound);
  if (!isJsonObject(outbound)) {
    error.value = 'Generated outbound is not valid JSON';
    return;
  }
  const next = upsertOutbound(parsedConfig.value, null, {
    tag: String(outbound.tag || protocolTool.tag || 'proxy'),
    protocol: String(outbound.protocol || 'freedom'),
    sendThrough: String(outbound.sendThrough || ''),
    settingsJson: JSON.stringify(outbound.settings || {}, null, 2),
    streamSettingsJson: JSON.stringify(outbound.streamSettings || {}, null, 2),
    proxySettingsJson: JSON.stringify(outbound.proxySettings || {}, null, 2),
    muxJson: JSON.stringify(outbound.mux || {}, null, 2),
  });
  applyTemplateConfig(next);
  void message.success('Generated outbound added to template');
}

async function loadWarpMatrixConfig() {
  loadingWarpMatrix.value = true;
  error.value = '';
  try {
    const [dataRaw, configRaw] = await Promise.all([
      runWarpAction('data', {}, { notifyOnError: false }),
      runWarpAction('config', {}, { notifyOnError: false }),
    ]);
    warpDataRaw.value = JSON.parse(dataRaw || '{}') as Record<string, unknown>;
    warpConfigRaw.value = JSON.parse(configRaw || '{}') as Record<string, unknown>;
    void message.success('WARP data loaded');
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to load WARP data';
  } finally {
    loadingWarpMatrix.value = false;
  }
}

async function applyWarpMatrix() {
  if (!warpDataRaw.value || !warpConfigRaw.value) {
    await loadWarpMatrixConfig();
  }
  if (!warpDataRaw.value || !warpConfigRaw.value) {
    return;
  }

  try {
    const baseSettings = buildWarpMatrixBaseSettings(
      warpDataRaw.value as { private_key: string; client_id: string },
      warpConfigRaw.value as {
        interface: { addresses: { v4?: string; v6?: string } };
        peers: Array<{ public_key: string; endpoint: { host: string } }>;
      },
    );
    const next = applyWarpMatrixToTemplate(
      parsedConfig.value,
      baseSettings,
      warpMatrixSelected.value.length > 0 ? warpMatrixSelected.value : ['warp'],
    );
    applyTemplateConfig(next);
    void message.success('WARP matrix applied to template');
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to apply WARP matrix';
  }
}

function applyDnsPresetOption(presetData: string[]) {
  const next = applyDnsPreset(parsedConfig.value, presetData);
  applyTemplateConfig(next);
  void message.success('DNS preset installed');
}

function applyDnsPolicyChanges() {
  const next = applyDnsPolicyForm(parsedConfig.value, dnsPolicyForm);
  applyTemplateConfig(next);
  void message.success('DNS policy updated');
}

function applyRuntimePolicyChanges() {
  const next = applyRuntimePolicyForm(parsedConfig.value, runtimePolicyForm);
  applyTemplateConfig(next);
  void message.success('Runtime policy updated');
}

function applyObservatoryChanges() {
  const next = applyObservatoryForm(parsedConfig.value, observatoryForm);
  applyTemplateConfig(next);
  void message.success('Observatory updated');
}

async function copyConfig() {
  await copyText(configText.value);
  void message.success('Copied');
}

function downloadConfig() {
  downloadText('xray-config.json', configText.value, 'application/json;charset=utf-8');
}

function formatConfig() {
  const parsed = parseJson(configText.value);
  if (!parsed) {
    error.value = 'Xray template is not valid JSON';
    return;
  }
  configText.value = JSON.stringify(parsed, null, 2);
  error.value = '';
}

function confirmStartOrRestart() {
  Modal.confirm({
    title: `${startRestartLabel.value} Xray?`,
    content: 'This uses the existing legacy Xray control path.',
    okText: startRestartLabel.value,
    onOk: runStartOrRestart,
  });
}

async function runStartOrRestart() {
  lifecycleBusy.value = true;
  error.value = '';
  try {
    await startXrayService({ notifyOnError: false });
    void message.success(`${startRestartLabel.value} command sent`);
    await refreshRuntime();
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to control Xray';
  } finally {
    lifecycleBusy.value = false;
  }
}

function confirmStop() {
  Modal.confirm({
    title: 'Stop Xray?',
    content: 'Active proxy traffic may be interrupted.',
    okButtonProps: { danger: true },
    okText: 'Stop',
    onOk: runStop,
  });
}

async function runStop() {
  lifecycleBusy.value = true;
  error.value = '';
  try {
    await stopXrayService({ notifyOnError: false });
    void message.success('Stop command sent');
    await refreshRuntime();
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to stop Xray';
  } finally {
    lifecycleBusy.value = false;
  }
}

async function loadVersions() {
  if (!hasInjectedRuntimeConfig()) {
    return;
  }

  loadingVersions.value = true;
  error.value = '';
  try {
    availableVersions.value = await getXrayVersions({ notifyOnError: false });
    selectedVersion.value ||= availableVersions.value[0];
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to load Xray versions';
  } finally {
    loadingVersions.value = false;
  }
}

function confirmInstallVersion() {
  if (!selectedVersion.value) {
    return;
  }

  Modal.confirm({
    title: `Install Xray ${selectedVersion.value}?`,
    content: 'The legacy backend will stop Xray, replace the binary, and restart Xray.',
    okButtonProps: { danger: true },
    okText: 'Install',
    onOk: installSelectedVersion,
  });
}

async function installSelectedVersion() {
  if (!selectedVersion.value) {
    return;
  }

  installingVersion.value = true;
  error.value = '';
  try {
    await installXrayVersion(selectedVersion.value, { notifyOnError: false });
    void message.success('Install command completed');
    await refreshRuntime();
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to install Xray version';
  } finally {
    installingVersion.value = false;
  }
}

function confirmSaveConfig() {
  const parsed = parseJson(configText.value);
  if (!parsed) {
    error.value = 'Xray template is not valid JSON';
    return;
  }

  Modal.confirm({
    title: 'Save Xray configuration?',
    content:
      'The template will be saved through the existing legacy endpoint so the old UI can still read it.',
    okText: 'Save',
    onOk: () => saveConfig(parsed),
  });
}

async function saveConfig(parsed: JsonValue) {
  savingConfig.value = true;
  error.value = '';
  const formatted = JSON.stringify(parsed, null, 2);
  try {
    await updateXraySetting(
      {
        xraySetting: formatted,
        outboundTestUrl: outboundTestUrl.value,
      },
      { notifyOnError: false },
    );
    configText.value = formatted;
    originalConfigText.value = formatted;
    loadedOutboundTestUrl.value = outboundTestUrl.value;
    void message.success('Xray configuration saved');
    confirmRestartAfterSave();
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to save Xray settings';
  } finally {
    savingConfig.value = false;
  }
}

async function runFirstOutboundTest() {
  const outbound = outboundsFromConfig.value[0];
  if (!outbound) {
    return;
  }

  testingOutbound.value = true;
  outboundToolResult.value = '';
  try {
    const result = await testOutbound(
      JSON.stringify(outbound),
      JSON.stringify(outboundsFromConfig.value),
      { notifyOnError: false },
    );
    outboundToolResult.value = result.success
      ? `Outbound test succeeded in ${result.delay} ms with status ${result.statusCode || '-'}`
      : `Outbound test failed: ${result.error || 'unknown error'}`;
  } catch (caught) {
    outboundToolResult.value = caught instanceof Error ? caught.message : 'Outbound test failed';
  } finally {
    testingOutbound.value = false;
  }
}

function confirmResetAllTraffic() {
  Modal.confirm({
    title: 'Reset all outbound traffic?',
    content: 'This resets stored traffic counters for all outbound tags.',
    okButtonProps: { danger: true },
    okText: 'Reset',
    onOk: () => resetTraffic('-alltags-'),
  });
}

function confirmResetOutboundTraffic(tag: string) {
  Modal.confirm({
    title: `Reset ${tag} traffic?`,
    content: 'This resets stored traffic counters for the selected outbound tag.',
    okButtonProps: { danger: true },
    okText: 'Reset',
    onOk: () => resetTraffic(tag),
  });
}

async function resetTraffic(tag: string) {
  resettingOutboundTraffic.value = tag === '-alltags-';
  resettingOutboundTag.value = tag === '-alltags-' ? '' : tag;
  try {
    await resetOutboundsTraffic(tag, { notifyOnError: false });
    void message.success('Outbound traffic reset');
    await loadOutboundsTraffic();
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to reset outbound traffic';
  } finally {
    resettingOutboundTraffic.value = false;
    resettingOutboundTag.value = '';
  }
}

async function runProviderData(target: 'nord' | 'nord-countries' | 'warp' | 'warp-config') {
  providerAction.value = `${target}-data`;
  outboundToolResult.value = '';
  try {
    const result =
      target === 'warp'
        ? await runWarpAction('data', {}, { notifyOnError: false })
        : target === 'warp-config'
          ? await runWarpAction('config', {}, { notifyOnError: false })
          : target === 'nord-countries'
            ? await runNordAction('countries', {}, { notifyOnError: false })
            : await runNordAction('data', {}, { notifyOnError: false });
    outboundToolResult.value = result || 'No provider data returned.';
  } catch (caught) {
    outboundToolResult.value = caught instanceof Error ? caught.message : 'Provider action failed';
  } finally {
    providerAction.value = '';
  }
}

function confirmRestartAfterSave() {
  Modal.confirm({
    title: 'Restart Xray now?',
    content: 'Restart applies the saved template through the existing Xray service path.',
    okText: 'Restart',
    onOk: runStartOrRestart,
  });
}

function parseJson(value: string): JsonValue | null {
  try {
    return JSON.parse(value) as JsonValue;
  } catch {
    return null;
  }
}

function isJsonObject(value: JsonValue | null | undefined): value is JsonObject {
  return Boolean(value && typeof value === 'object' && !Array.isArray(value));
}

function sectionArrayCount(config: JsonValue | null, key: string): string {
  if (!isJsonObject(config)) {
    return '-';
  }
  const section = config[key];
  return Array.isArray(section) ? formatCount(section.length) : objectState(config, key);
}

function nestedArrayCount(config: JsonValue | null, sectionKey: string, childKey: string): string {
  if (!isJsonObject(config)) {
    return '-';
  }
  const section = config[sectionKey];
  if (!isJsonObject(section)) {
    return '-';
  }
  const child = section[childKey];
  return Array.isArray(child) ? formatCount(child.length) : '-';
}

function objectState(config: JsonValue | null, key: string): string {
  if (!isJsonObject(config)) {
    return '-';
  }
  return config[key] ? 'Present' : 'Missing';
}

onMounted(() => {
  serverStore.connectRealtime();
  void refreshRuntime();
  void refreshConfig();
});
</script>
