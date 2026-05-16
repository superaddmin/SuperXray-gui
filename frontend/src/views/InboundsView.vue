<template>
  <section class="page-stack inbounds-page">
    <PageHeader
      eyebrow="Traffic"
      title="Inbounds"
      description="Manage Xray listeners, clients, live activity, traffic counters, and sharing tools."
    >
      <div
        class="page-header-actions--compact"
        :data-primary-actions="headerPrimaryActionKeys.length"
      >
        <AButton :loading="loading" @click="refreshInbounds">
          <template #icon><ReloadOutlined /></template>
          Refresh
        </AButton>
        <AButton :loading="loadingActivity" @click="refreshClientActivity">
          <template #icon><ReloadOutlined /></template>
          Refresh Activity
        </AButton>
        <ADropdown v-model:open="moreActionsOpen" :trigger="['click']">
          <AButton>
            <template #icon><EllipsisOutlined /></template>
            {{ translate('action.moreActions', appStore.locale) }}
          </AButton>
          <template #overlay>
            <AMenu :items="moreActionItems" @click="handleMoreActionClick" />
          </template>
        </ADropdown>
        <AButton type="primary" @click="openCreateInbound">
          <template #icon><PlusOutlined /></template>
          New Inbound
        </AButton>
      </div>
    </PageHeader>

    <AAlert v-if="error" banner type="warning" :message="error" />

    <div class="status-grid">
      <StatusTile
        label="Total"
        :value="formatCount(inbounds.length)"
        hint="Legacy Xray inbounds"
        tone="info"
      />
      <StatusTile
        label="Enabled"
        :value="formatCount(enabledInboundCount)"
        hint="Active listeners"
        tone="success"
      />
      <StatusTile
        label="Online Clients"
        :value="formatCount(onlineClients.length)"
        hint="Live activity from Xray"
        tone="success"
      />
      <StatusTile
        label="Clients"
        :value="formatCount(clientCount)"
        hint="Configured users"
        tone="info"
      />
      <StatusTile label="Traffic" :value="trafficTotal" hint="Inbound counters" tone="success" />
    </div>

    <ACard class="work-panel" :bordered="false">
      <div class="toolbar-grid inbounds-toolbar">
        <label class="visually-hidden" for="inbounds-protocol-filter">Protocol filter</label>
        <select
          id="inbounds-protocol-filter"
          v-model="protocolFilter"
          class="toolbar-select"
          aria-label="Protocol filter"
        >
          <option
            v-for="option in protocolFilterOptions"
            :key="option.value"
            :value="option.value"
          >
            {{ option.label }}
          </option>
        </select>
        <label class="visually-hidden" for="inbounds-state-filter">State filter</label>
        <select
          id="inbounds-state-filter"
          v-model="stateFilter"
          class="toolbar-select"
          aria-label="State filter"
        >
          <option v-for="option in stateFilterOptions" :key="option.value" :value="option.value">
            {{ option.label }}
          </option>
        </select>
        <AInput
          v-model:value="keyword"
          allow-clear
          aria-label="Search inbounds"
          placeholder="Search"
        />
      </div>

      <ATable
        :columns="inboundColumns"
        :data-source="filteredInbounds"
        :loading="loading"
        :pagination="{ pageSize: 10, showSizeChanger: true }"
        row-key="id"
        size="middle"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'inbound'">
            <ASpace direction="vertical" :size="2">
              <ASpace wrap>
                <ATag :color="protocolColor(record.protocol)">{{ record.protocol }}</ATag>
                <strong>{{ record.remark || record.tag || `Inbound ${record.id}` }}</strong>
              </ASpace>
              <span class="muted-text">{{ record.tag || '-' }}</span>
            </ASpace>
          </template>

          <template v-else-if="column.key === 'address'">
            <span>{{ inboundAddress(record) }}</span>
          </template>

          <template v-else-if="column.key === 'transport'">
            <ASpace wrap>
              <ATag>{{ inboundNetwork(record) }}</ATag>
              <ATag :color="inboundSecurity(record) === 'none' ? undefined : 'green'">
                {{ inboundSecurity(record) }}
              </ATag>
            </ASpace>
          </template>

          <template v-else-if="column.key === 'traffic'">
            <span>{{ formatTraffic(record.up, record.down, record.total) }}</span>
          </template>

          <template v-else-if="column.key === 'clients'">
            <span>{{ formatCount(inboundClientCount(record)) }}</span>
          </template>

          <template v-else-if="column.key === 'enable'">
            <ASwitch
              :aria-label="`${record.enable ? 'Disable' : 'Enable'} inbound ${record.remark || record.tag || record.id}`"
              :checked="record.enable"
              :loading="busyInboundId === record.id"
              @change="(checked) => toggleInbound(record, Boolean(checked))"
            />
          </template>

          <template v-else-if="column.key === 'actions'">
            <ASpace wrap>
              <AButton size="small" @click="openInboundDetail(record)">
                <template #icon><EyeOutlined /></template>
                Details
              </AButton>
              <AButton
                size="small"
                :disabled="inboundShareExportDisabled(record)"
                @click="exportInboundShareLinks(asInbound(record))"
              >
                <template #icon><CopyOutlined /></template>
                导出链接
              </AButton>
              <AButton
                size="small"
                :disabled="inboundClientCount(record) === 0"
                @click="exportInboundSubscriptionLinks(asInbound(record))"
              >
                <template #icon><LinkOutlined /></template>
                导出订阅
              </AButton>
              <AButton size="small" @click="exportInboundJson(asInbound(record))">
                <template #icon><CopyOutlined /></template>
                JSON
              </AButton>
              <AButton
                size="small"
                :disabled="inboundShareExportDisabled(record)"
                @click="openInboundQrcode(asInbound(record))"
              >
                <template #icon><QrcodeOutlined /></template>
                QR
              </AButton>
              <AButton
                size="small"
                :loading="busyInboundId === record.id"
                @click="confirmResetInboundTraffic(asInbound(record))"
              >
                <template #icon><ReloadOutlined /></template>
                Reset
              </AButton>
              <AButton size="small" @click="confirmCloneInbound(asInbound(record))">
                <template #icon><BlockOutlined /></template>
                Clone
              </AButton>
              <AButton size="small" @click="openEditInbound(record)">
                <template #icon><EditOutlined /></template>
                Edit
              </AButton>
              <AButton danger size="small" @click="confirmDeleteInbound(record)">
                <template #icon><DeleteOutlined /></template>
                Delete
              </AButton>
            </ASpace>
          </template>
        </template>
      </ATable>
    </ACard>

    <ADrawer v-model:open="detailOpen" destroy-on-close :title="selectedInboundTitle" width="760">
      <template v-if="selectedInbound">
        <div class="drawer-summary">
          <StatusTile
            label="Address"
            :value="inboundAddress(selectedInbound)"
            :hint="selectedInbound.tag || '-'"
          />
          <StatusTile
            label="Transport"
            :value="inboundNetwork(selectedInbound)"
            :hint="inboundSecurity(selectedInbound)"
          />
          <StatusTile
            label="Traffic"
            :value="formatTraffic(selectedInbound.up, selectedInbound.down, selectedInbound.total)"
            :hint="selectedInbound.enable ? 'Enabled' : 'Disabled'"
          />
          <StatusTile
            label="Expires"
            :value="formatTimestamp(selectedInbound.expiryTime)"
            :hint="selectedInbound.trafficReset || 'never'"
          />
        </div>

        <ACard class="work-panel drawer-panel" :bordered="false">
          <div class="panel-header">
            <div>
              <p class="page-eyebrow">Activity</p>
              <h2>Online / IP Management</h2>
            </div>
            <AButton :loading="loadingActivity" @click="refreshClientActivity">
              <template #icon><ReloadOutlined /></template>
              Refresh Activity
            </AButton>
          </div>

          <ATable
            :columns="activityColumns"
            :data-source="selectedClientRows"
            :loading="loadingActivity"
            :pagination="{ pageSize: 6 }"
            row-key="key"
            size="small"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'client'">
                <ASpace direction="vertical" :size="2">
                  <strong>{{ asClientRow(record).email || '-' }}</strong>
                  <span class="muted-text">{{ clientPrimaryText(selectedInbound, record) }}</span>
                </ASpace>
              </template>

              <template v-else-if="column.key === 'online'">
                <ATag :color="isClientOnline(asClientRow(record).email) ? 'green' : undefined">
                  {{ isClientOnline(asClientRow(record).email) ? 'Online' : 'Offline' }}
                </ATag>
              </template>

              <template v-else-if="column.key === 'lastOnline'">
                <span>{{ formatClientLastOnline(asClientRow(record)) }}</span>
              </template>

              <template v-else-if="column.key === 'actions'">
                <ASpace wrap>
                  <AButton
                    :disabled="!asClientRow(record).email"
                    size="small"
                    @click="openClientIps(asClientRow(record))"
                  >
                    <template #icon><EyeOutlined /></template>
                    View IPs
                  </AButton>
                  <AButton
                    danger
                    :disabled="!asClientRow(record).email"
                    :loading="clearingClientIpsEmail === asClientRow(record).email"
                    size="small"
                    @click="confirmClearClientIps(asClientRow(record))"
                  >
                    Clear IPs
                  </AButton>
                </ASpace>
              </template>
            </template>
          </ATable>
        </ACard>

        <ACard class="work-panel drawer-panel" :bordered="false">
          <div class="panel-header">
            <div>
              <p class="page-eyebrow">Users</p>
              <h2>Clients</h2>
            </div>
            <ASpace wrap>
              <AButton
                :disabled="copyClientSourceOptions.length === 0"
                @click="openCopyClientsModal(selectedInbound)"
              >
                <template #icon><CopyOutlined /></template>
                Copy Clients
              </AButton>
              <AButton
                :disabled="
                  !selectedInboundClientManageable || selectedInbound?.protocol === 'wireguard'
                "
                @click="openBulkAddClientsModal(selectedInbound)"
              >
                <template #icon><UserAddOutlined /></template>
                Bulk Add
              </AButton>
              <AButton
                :disabled="selectedInbound ? inboundShareExportDisabled(selectedInbound) : true"
                @click="exportInboundShareLinks(selectedInbound)"
              >
                <template #icon><CopyOutlined /></template>
                Export Share Links
              </AButton>
              <AButton
                :disabled="!selectedClientRows.length"
                :loading="loadingSubscriptionSettings"
                @click="exportInboundSubscriptionLinks(selectedInbound)"
              >
                <template #icon><LinkOutlined /></template>
                Export Subscription Links
              </AButton>
              <AButton
                :disabled="!canResetSelectedClients"
                :loading="busyClient"
                @click="confirmResetSelectedClients(selectedInbound)"
              >
                <template #icon><ReloadOutlined /></template>
                Reset Selected
              </AButton>
              <AButton
                danger
                :disabled="!canDeleteSelectedClients"
                :loading="busyClient"
                @click="confirmDeleteSelectedClients(selectedInbound)"
              >
                <template #icon><DeleteOutlined /></template>
                Delete Selected
              </AButton>
              <AButton
                :disabled="
                  !selectedInboundClientManageable || selectedInbound.protocol === 'wireguard'
                "
                :loading="busyClient"
                @click="confirmResetAllClients(selectedInbound)"
              >
                <template #icon><ReloadOutlined /></template>
                Reset All
              </AButton>
              <AButton
                danger
                :disabled="!selectedInboundClientManageable"
                :loading="deletingDepletedClients"
                @click="confirmDeleteDepletedClients(selectedInbound)"
              >
                Delete Depleted
              </AButton>
              <AButton
                :disabled="!selectedInboundClientManageable || clientAddDisabled(selectedInbound)"
                type="primary"
                @click="openCreateClient(selectedInbound)"
              >
                <template #icon><UserAddOutlined /></template>
                Add Client
              </AButton>
            </ASpace>
          </div>

          <ATable
            :columns="clientColumns"
            :data-source="selectedClientRows"
            :pagination="{ pageSize: 6 }"
            :row-selection="clientRowSelection"
            row-key="key"
            size="small"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'client'">
                <ASpace direction="vertical" :size="2">
                  <strong>{{ record.email || '-' }}</strong>
                  <span class="muted-text">{{ clientPrimaryText(selectedInbound, record) }}</span>
                </ASpace>
              </template>

              <template v-else-if="column.key === 'limit'">
                <span>{{ formatLimit(record.traffic?.total ?? record.totalGB) }}</span>
              </template>

              <template v-else-if="column.key === 'usage'">
                <span>{{
                  formatTraffic(
                    record.traffic?.up || 0,
                    record.traffic?.down || 0,
                    record.traffic?.total || 0,
                  )
                }}</span>
              </template>

              <template v-else-if="column.key === 'expiry'">
                <span>{{ formatTimestamp(record.traffic?.expiryTime ?? record.expiryTime) }}</span>
              </template>

              <template v-else-if="column.key === 'enable'">
                <ASwitch
                  :checked="record.enable !== false"
                  :disabled="clientActionDisabled(selectedInbound, record)"
                  :loading="busyClientKey === record.key"
                  @change="(checked) => toggleClient(selectedInbound, record, Boolean(checked))"
                />
              </template>

              <template v-else-if="column.key === 'actions'">
                <ASpace wrap>
                  <AButton
                    :disabled="clientActionDisabled(selectedInbound, record)"
                    size="small"
                    @click="openEditClient(selectedInbound, record)"
                  >
                    <template #icon><EditOutlined /></template>
                    Edit
                  </AButton>
                  <AButton
                    :disabled="clientShareDisabled(selectedInbound, record)"
                    size="small"
                    @click="previewShareLink(selectedInbound, record)"
                  >
                    <template #icon><CopyOutlined /></template>
                    Share
                  </AButton>
                  <AButton
                    :disabled="clientSubscriptionDisabled(selectedInbound, record)"
                    :loading="loadingSubscriptionSettings"
                    size="small"
                    @click="openClientAccessModal(record)"
                  >
                    <template #icon><QrcodeOutlined /></template>
                    Access
                  </AButton>
                  <AButton
                    :disabled="clientResetDisabled(selectedInbound, record)"
                    size="small"
                    @click="confirmResetClient(selectedInbound, record)"
                  >
                    <template #icon><ReloadOutlined /></template>
                    Reset
                  </AButton>
                  <AButton
                    danger
                    :disabled="clientActionDisabled(selectedInbound, record)"
                    size="small"
                    @click="confirmDeleteClient(selectedInbound, record)"
                  >
                    <template #icon><DeleteOutlined /></template>
                    Delete
                  </AButton>
                </ASpace>
              </template>
            </template>
          </ATable>
        </ACard>

        <ACard v-if="sharePreview" class="work-panel drawer-panel" :bordered="false">
          <div class="panel-header">
            <div>
              <p class="page-eyebrow">Export</p>
              <h2>{{ sharePreviewTitle }}</h2>
            </div>
            <ASpace wrap>
              <AButton @click="copySharePreview">
                <template #icon><CopyOutlined /></template>
                Copy
              </AButton>
              <AButton @click="downloadSharePreview">
                <template #icon><DownloadOutlined /></template>
                Download
              </AButton>
            </ASpace>
          </div>
          <textarea class="json-editor compact-json-editor" readonly :value="sharePreview" />
        </ACard>
      </template>
    </ADrawer>

    <AModal
      v-model:open="inboundModalOpen"
      destroy-on-close
      :confirm-loading="savingInbound"
      :title="inboundModalTitle"
      width="780px"
      @ok="submitInbound"
    >
      <AForm class="responsive-modal-form" layout="vertical">
        <AAlert
          class="mb-12"
          message="Inbound setup guide"
          description="Start with protocol, remark, listen address, and port. Keep 0.0.0.0 to accept connections on all interfaces; set traffic and expiry to 0 for no limit. Use the transport form for common TCP/WS/gRPC/TLS/Reality options, then click Sync JSON only when you need to inspect or manually adjust the raw legacy JSON."
          show-icon
          type="info"
        />
        <FormSection
          eyebrow="Inbound"
          title="Basic Inbound"
          description="Protocol, listening address, limits, and enable state are saved through the existing inbound submit path."
        >
          <div class="form-grid">
            <AFormItem label="Protocol">
              <ASelect
                v-model:value="inboundEditor.protocol"
                :disabled="inboundModalMode === 'edit'"
                :options="editableProtocolOptions"
              />
            </AFormItem>
            <AFormItem label="Remark">
              <AInput v-model:value="inboundEditor.remark" />
            </AFormItem>
            <AFormItem label="Listen">
              <AInput v-model:value="inboundEditor.listen" placeholder="0.0.0.0" />
            </AFormItem>
            <AFormItem label="Port">
              <AInputNumber
                v-model:value="inboundEditor.port"
                :max="65535"
                :min="1"
                class="full-width"
              />
            </AFormItem>
            <AFormItem label="Traffic Limit GB">
              <AInputNumber v-model:value="inboundEditor.totalGB" :min="0" class="full-width" />
            </AFormItem>
            <AFormItem label="Expiry Timestamp">
              <AInputNumber v-model:value="inboundEditor.expiryTime" :min="0" class="full-width" />
            </AFormItem>
            <AFormItem label="Traffic Reset">
              <AInput v-model:value="inboundEditor.trafficReset" />
            </AFormItem>
            <AFormItem label="Enable">
              <ASwitch v-model:checked="inboundEditor.enable" />
            </AFormItem>
          </div>
        </FormSection>

        <FormSection
          v-if="gatewayProxyUris.length > 0"
          eyebrow="Gateway"
          title="Gateway Proxy URI"
          description="Use these local-only proxy URIs in Super-Code-Gateway to route OpenAI access through this Xray exit."
        >
          <ASpace direction="vertical" class="full-width">
            <AAlert
              message="Local gateway exit"
              description="The template listens on 127.0.0.1 only. Keep Super-Code-Gateway on the same host or connect through a trusted local network path."
              show-icon
              type="success"
            />
            <ASpace
              v-for="item in gatewayProxyUris"
              :key="item.uri"
              class="gateway-proxy-uri-row"
              wrap
            >
              <ATag>{{ item.label }}</ATag>
              <AInput :value="item.uri" readonly class="gateway-proxy-uri-input" />
              <AButton size="small" @click="copyGatewayProxyUri(item.uri)">
                <template #icon><CopyOutlined /></template>
                Copy
              </AButton>
            </ASpace>
          </ASpace>
        </FormSection>

        <FormSection
          v-if="inboundEditor.protocol === 'wireguard'"
          eyebrow="Protocol"
          title="WireGuard Settings"
          description="Server keys and WireGuard-specific settings stay synchronized with the legacy settings JSON."
        >
          <template #actions>
            <AButton size="small" @click="syncWireguardEditorFromSettings">Sync JSON</AButton>
            <AButton size="small" @click="applyWireguardEditorToSettings">Apply</AButton>
          </template>
          <div class="form-grid">
            <AFormItem label="MTU">
              <AInputNumber v-model:value="wireguardEditor.mtu" :min="0" class="full-width" />
            </AFormItem>
            <AFormItem label="noKernelTun">
              <ASwitch v-model:checked="wireguardEditor.noKernelTun" />
            </AFormItem>
            <AFormItem label="Server Private Key">
              <ASpace.Compact class="full-width">
                <AInput v-model:value="wireguardEditor.secretKey" />
                <AButton @click="generateWireguardServerKeys">Generate</AButton>
              </ASpace.Compact>
            </AFormItem>
            <AFormItem label="Server Public Key">
              <AInput v-model:value="wireguardEditor.pubKey" />
            </AFormItem>
          </div>
        </FormSection>

        <FormSection
          v-if="protocolSupportsStream(inboundEditor.protocol)"
          eyebrow="Transport"
          title="Transport Settings"
          description="Network, security, TLS, Reality and sockopt controls continue to update the existing stream settings JSON."
        >
          <template #actions>
            <AButton size="small" @click="syncStreamEditorFromSettings">Sync JSON</AButton>
            <AButton size="small" @click="applyStreamEditorToSettings">Apply</AButton>
          </template>
          <div class="form-grid">
            <AFormItem label="Network">
              <ASelect v-model:value="streamEditor.network" :options="transportNetworkOptions" />
            </AFormItem>
            <AFormItem label="Security">
              <ASelect v-model:value="streamEditor.security" :options="transportSecurityOptions" />
            </AFormItem>
            <AFormItem v-if="streamEditor.network === 'tcp'" label="TCP Proxy Protocol">
              <ASwitch v-model:checked="streamEditor.tcpAcceptProxyProtocol" />
            </AFormItem>
            <AFormItem v-if="streamEditor.network === 'tcp'" label="TCP Header">
              <ASelect
                v-model:value="streamEditor.tcpHeaderType"
                :options="[
                  { label: 'none', value: 'none' },
                  { label: 'http', value: 'http' },
                ]"
              />
            </AFormItem>
            <AFormItem v-if="streamEditor.network === 'kcp'" label="mKCP MTU">
              <AInputNumber v-model:value="streamEditor.kcpMtu" :min="1" class="full-width" />
            </AFormItem>
            <AFormItem v-if="streamEditor.network === 'kcp'" label="mKCP TTI">
              <AInputNumber v-model:value="streamEditor.kcpTti" :min="1" class="full-width" />
            </AFormItem>
            <AFormItem v-if="streamEditor.network === 'kcp'" label="mKCP Uplink">
              <AInputNumber
                v-model:value="streamEditor.kcpUplinkCapacity"
                :min="0"
                class="full-width"
              />
            </AFormItem>
            <AFormItem v-if="streamEditor.network === 'kcp'" label="mKCP Downlink">
              <AInputNumber
                v-model:value="streamEditor.kcpDownlinkCapacity"
                :min="0"
                class="full-width"
              />
            </AFormItem>
            <AFormItem v-if="streamEditor.network === 'kcp'" label="mKCP CWND Multiplier">
              <AInputNumber
                v-model:value="streamEditor.kcpCwndMultiplier"
                :min="0"
                class="full-width"
              />
            </AFormItem>
            <AFormItem v-if="streamEditor.network === 'kcp'" label="mKCP Sending Window">
              <AInputNumber
                v-model:value="streamEditor.kcpMaxSendingWindow"
                :min="0"
                class="full-width"
              />
            </AFormItem>
            <AFormItem v-if="streamEditor.network === 'ws'" label="WS Proxy Protocol">
              <ASwitch v-model:checked="streamEditor.wsAcceptProxyProtocol" />
            </AFormItem>
            <AFormItem v-if="streamEditor.network === 'ws'" label="WS Path">
              <AInput v-model:value="streamEditor.wsPath" />
            </AFormItem>
            <AFormItem v-if="streamEditor.network === 'ws'" label="WS Host">
              <AInput v-model:value="streamEditor.wsHost" />
            </AFormItem>
            <AFormItem v-if="streamEditor.network === 'ws'" label="WS Heartbeat Period">
              <AInputNumber
                v-model:value="streamEditor.wsHeartbeatPeriod"
                :min="0"
                class="full-width"
              />
            </AFormItem>
            <AFormItem v-if="streamEditor.network === 'grpc'" label="gRPC Service">
              <AInput v-model:value="streamEditor.grpcServiceName" />
            </AFormItem>
            <AFormItem v-if="streamEditor.network === 'grpc'" label="gRPC Authority">
              <AInput v-model:value="streamEditor.grpcAuthority" />
            </AFormItem>
            <AFormItem v-if="streamEditor.network === 'grpc'" label="gRPC Multi Mode">
              <ASwitch v-model:checked="streamEditor.grpcMultiMode" />
            </AFormItem>
            <AFormItem
              v-if="streamEditor.network === 'httpupgrade'"
              label="HTTPUpgrade Proxy Protocol"
            >
              <ASwitch v-model:checked="streamEditor.httpupgradeAcceptProxyProtocol" />
            </AFormItem>
            <AFormItem v-if="streamEditor.network === 'httpupgrade'" label="HTTPUpgrade Path">
              <AInput v-model:value="streamEditor.httpupgradePath" />
            </AFormItem>
            <AFormItem v-if="streamEditor.network === 'httpupgrade'" label="HTTPUpgrade Host">
              <AInput v-model:value="streamEditor.httpupgradeHost" />
            </AFormItem>
            <AFormItem v-if="streamEditor.network === 'xhttp'" label="XHTTP Path">
              <AInput v-model:value="streamEditor.xhttpPath" />
            </AFormItem>
            <AFormItem v-if="streamEditor.network === 'xhttp'" label="XHTTP Host">
              <AInput v-model:value="streamEditor.xhttpHost" />
            </AFormItem>
            <AFormItem v-if="streamEditor.network === 'xhttp'" label="XHTTP Mode">
              <ASelect v-model:value="streamEditor.xhttpMode" :options="xhttpModeOptions" />
            </AFormItem>
            <AFormItem v-if="streamEditor.network === 'xhttp'" label="No SSE Header">
              <ASwitch v-model:checked="streamEditor.xhttpNoSseHeader" />
            </AFormItem>
            <AFormItem v-if="streamEditor.network === 'xhttp'" label="Max Buffered Posts">
              <AInputNumber
                v-model:value="streamEditor.xhttpScMaxBufferedPosts"
                :min="0"
                class="full-width"
              />
            </AFormItem>
            <AFormItem v-if="streamEditor.network === 'xhttp'" label="Each Post Bytes">
              <AInput v-model:value="streamEditor.xhttpScMaxEachPostBytes" />
            </AFormItem>
            <AFormItem v-if="streamEditor.network === 'xhttp'" label="Stream Up Server Secs">
              <AInput v-model:value="streamEditor.xhttpScStreamUpServerSecs" />
            </AFormItem>
            <AFormItem v-if="streamEditor.network === 'xhttp'" label="Padding Bytes">
              <AInput v-model:value="streamEditor.xhttpXPaddingBytes" />
            </AFormItem>
            <AFormItem v-if="streamEditor.security === 'tls'" label="TLS SNI">
              <AInput v-model:value="streamEditor.tlsServerName" />
            </AFormItem>
            <AFormItem v-if="streamEditor.security === 'tls'" label="TLS Min Version">
              <ASelect v-model:value="streamEditor.tlsMinVersion" :options="tlsVersionOptions" />
            </AFormItem>
            <AFormItem v-if="streamEditor.security === 'tls'" label="TLS Max Version">
              <ASelect v-model:value="streamEditor.tlsMaxVersion" :options="tlsVersionOptions" />
            </AFormItem>
            <AFormItem v-if="streamEditor.security === 'tls'" label="TLS ALPN">
              <AInput v-model:value="streamEditor.tlsAlpn" placeholder="h2,http/1.1" />
            </AFormItem>
            <AFormItem v-if="streamEditor.security === 'tls'" label="TLS Fingerprint">
              <AInput v-model:value="streamEditor.tlsFingerprint" />
            </AFormItem>
            <AFormItem v-if="streamEditor.security === 'tls'" label="Certificate File">
              <AInput v-model:value="streamEditor.tlsCertificateFile" />
            </AFormItem>
            <AFormItem v-if="streamEditor.security === 'tls'" label="Key File">
              <AInput v-model:value="streamEditor.tlsKeyFile" />
            </AFormItem>
            <AFormItem v-if="streamEditor.security === 'tls'" label="Reject Unknown SNI">
              <ASwitch v-model:checked="streamEditor.tlsRejectUnknownSni" />
            </AFormItem>
            <AFormItem v-if="streamEditor.security === 'tls'" label="Disable System Root">
              <ASwitch v-model:checked="streamEditor.tlsDisableSystemRoot" />
            </AFormItem>
            <AFormItem v-if="streamEditor.security === 'tls'" label="Session Resumption">
              <ASwitch v-model:checked="streamEditor.tlsEnableSessionResumption" />
            </AFormItem>
            <AFormItem v-if="streamEditor.security === 'tls'" label="ECH Server Keys">
              <AInput v-model:value="streamEditor.tlsEchServerKeys" />
            </AFormItem>
            <AFormItem v-if="streamEditor.security === 'tls'" label="ECH Config List">
              <AInput v-model:value="streamEditor.tlsEchConfigList" />
            </AFormItem>
            <AFormItem v-if="streamEditor.security === 'reality'" label="Reality Show">
              <ASwitch v-model:checked="streamEditor.realityShow" />
            </AFormItem>
            <AFormItem v-if="streamEditor.security === 'reality'" label="Reality Xver">
              <AInputNumber v-model:value="streamEditor.realityXver" :min="0" class="full-width" />
            </AFormItem>
            <AFormItem v-if="streamEditor.security === 'reality'" label="Reality Target">
              <AInput v-model:value="streamEditor.realityTarget" />
            </AFormItem>
            <AFormItem v-if="streamEditor.security === 'reality'" label="Reality Server Names">
              <AInput v-model:value="streamEditor.realityServerNames" />
            </AFormItem>
            <AFormItem v-if="streamEditor.security === 'reality'" label="Reality Private Key">
              <AInput v-model:value="streamEditor.realityPrivateKey" />
            </AFormItem>
            <AFormItem v-if="streamEditor.security === 'reality'" label="Reality Short IDs">
              <AInput v-model:value="streamEditor.realityShortIds" />
            </AFormItem>
            <AFormItem v-if="streamEditor.security === 'reality'" label="Reality Public Key">
              <AInput v-model:value="streamEditor.realityPublicKey" />
            </AFormItem>
            <AFormItem v-if="streamEditor.security === 'reality'" label="Reality SpiderX">
              <AInput v-model:value="streamEditor.realitySpiderX" />
            </AFormItem>
            <AFormItem v-if="streamEditor.security === 'reality'" label="Min Client Version">
              <AInput v-model:value="streamEditor.realityMinClientVer" />
            </AFormItem>
            <AFormItem v-if="streamEditor.security === 'reality'" label="Max Client Version">
              <AInput v-model:value="streamEditor.realityMaxClientVer" />
            </AFormItem>
            <AFormItem v-if="streamEditor.security === 'reality'" label="Max Time Diff">
              <AInputNumber
                v-model:value="streamEditor.realityMaxTimediff"
                :min="0"
                class="full-width"
              />
            </AFormItem>
            <AFormItem v-if="streamEditor.security === 'reality'" label="ML-DSA-65 Seed">
              <AInput v-model:value="streamEditor.realityMldsa65Seed" />
            </AFormItem>
            <AFormItem v-if="streamEditor.security === 'reality'" label="ML-DSA-65 Verify">
              <AInput v-model:value="streamEditor.realityMldsa65Verify" />
            </AFormItem>
            <AFormItem v-if="isHysteriaProtocol(inboundEditor.protocol)" label="Hysteria2 Auth">
              <AInput v-model:value="streamEditor.hysteriaAuth" />
            </AFormItem>
            <AFormItem v-if="isHysteriaProtocol(inboundEditor.protocol)" label="UDP Idle Timeout">
              <AInputNumber
                v-model:value="streamEditor.hysteriaUdpIdleTimeout"
                :min="0"
                class="full-width"
              />
            </AFormItem>
            <AFormItem label="Sockopt Enabled">
              <ASwitch v-model:checked="streamEditor.sockoptEnabled" />
            </AFormItem>
            <template v-if="streamEditor.sockoptEnabled">
              <AFormItem label="Sockopt Proxy Protocol">
                <ASwitch v-model:checked="streamEditor.sockoptAcceptProxyProtocol" />
              </AFormItem>
              <AFormItem label="TCP Fast Open">
                <ASwitch v-model:checked="streamEditor.sockoptTcpFastOpen" />
              </AFormItem>
              <AFormItem label="Multipath TCP">
                <ASwitch v-model:checked="streamEditor.sockoptTcpMptcp" />
              </AFormItem>
              <AFormItem label="Penetrate">
                <ASwitch v-model:checked="streamEditor.sockoptPenetrate" />
              </AFormItem>
              <AFormItem label="V6 Only">
                <ASwitch v-model:checked="streamEditor.sockoptV6Only" />
              </AFormItem>
              <AFormItem label="Domain Strategy">
                <ASelect
                  v-model:value="streamEditor.sockoptDomainStrategy"
                  :options="sockoptDomainStrategyOptions"
                />
              </AFormItem>
              <AFormItem label="TCP Congestion">
                <ASelect
                  v-model:value="streamEditor.sockoptTcpCongestion"
                  :options="sockoptTcpCongestionOptions"
                />
              </AFormItem>
              <AFormItem label="TProxy">
                <ASelect
                  v-model:value="streamEditor.sockoptTproxy"
                  :options="sockoptTproxyOptions"
                />
              </AFormItem>
              <AFormItem label="Mark">
                <AInputNumber
                  v-model:value="streamEditor.sockoptMark"
                  :min="0"
                  class="full-width"
                />
              </AFormItem>
              <AFormItem label="TCP Max Segment">
                <AInputNumber
                  v-model:value="streamEditor.sockoptTcpMaxSeg"
                  :min="0"
                  class="full-width"
                />
              </AFormItem>
              <AFormItem label="Dialer Proxy">
                <AInput v-model:value="streamEditor.sockoptDialerProxy" />
              </AFormItem>
              <AFormItem label="Interface Name">
                <AInput v-model:value="streamEditor.sockoptInterfaceName" />
              </AFormItem>
              <AFormItem label="Trusted X-Forwarded-For">
                <AInput
                  v-model:value="streamEditor.sockoptTrustedXForwardedFor"
                  placeholder="CF-Connecting-IP,X-Real-IP"
                />
              </AFormItem>
            </template>
          </div>
        </FormSection>

        <FormSection
          v-if="inboundClientSectionVisible"
          eyebrow="Client"
          title="Default Client"
          description="Create the first client for protocols that require one. Apply keeps the form and raw settings JSON in sync."
        >
          <template #actions>
            <AButton size="small" @click="syncInboundClientEditorFromSettings">Sync JSON</AButton>
            <AButton size="small" @click="applyInboundClientEditorToSettings">Apply</AButton>
          </template>
          <div class="form-grid client-form-grid">
            <AFormItem label="Email">
              <AInput v-model:value="inboundClientEditor.email" />
            </AFormItem>
            <AFormItem v-if="usesUuidClientId(inboundClientEditor.protocol)" label="UUID">
              <ASpace.Compact class="full-width">
                <AInput v-model:value="inboundClientEditor.id" />
                <AButton @click="inboundClientEditor.id = randomUuid()">Generate</AButton>
              </ASpace.Compact>
            </AFormItem>
            <AFormItem v-if="usesPasswordClientId(inboundClientEditor.protocol)" label="Password">
              <ASpace.Compact class="full-width">
                <AInput v-model:value="inboundClientEditor.password" />
                <AButton
                  @click="
                    inboundClientEditor.password = generateClientCredential(
                      inboundClientEditor.protocol,
                    )
                  "
                >
                  Generate
                </AButton>
              </ASpace.Compact>
            </AFormItem>
            <AFormItem v-if="usesAuthClientId(inboundClientEditor.protocol)" label="Auth">
              <ASpace.Compact class="full-width">
                <AInput v-model:value="inboundClientEditor.auth" />
                <AButton
                  @click="
                    inboundClientEditor.auth = generateClientCredential(
                      inboundClientEditor.protocol,
                    )
                  "
                >
                  Generate
                </AButton>
              </ASpace.Compact>
            </AFormItem>
            <AFormItem v-if="inboundClientEditor.protocol === 'vmess'" label="Security">
              <ASelect v-model:value="inboundClientEditor.security" :options="securityOptions" />
            </AFormItem>
            <AFormItem v-if="inboundVlessFlowVisible" label="Flow">
              <ASelect v-model:value="inboundClientEditor.flow" :options="flowOptions" />
            </AFormItem>
            <AFormItem v-if="inboundClientEditor.protocol === 'shadowsocks'" label="Method">
              <ASelect
                v-model:value="inboundClientEditor.method"
                :options="shadowsocksMethodOptions"
              />
            </AFormItem>
            <AFormItem label="Traffic Limit GB">
              <AInputNumber
                v-model:value="inboundClientEditor.totalGB"
                :min="0"
                class="full-width"
              />
            </AFormItem>
            <AFormItem label="Expiry Timestamp">
              <AInputNumber
                v-model:value="inboundClientEditor.expiryTime"
                :min="0"
                class="full-width"
              />
            </AFormItem>
            <AFormItem label="IP Limit">
              <AInputNumber
                v-model:value="inboundClientEditor.limitIp"
                :min="0"
                class="full-width"
              />
            </AFormItem>
            <AFormItem label="Reset Days">
              <AInputNumber v-model:value="inboundClientEditor.reset" :min="0" class="full-width" />
            </AFormItem>
            <AFormItem label="Sub ID">
              <AInput v-model:value="inboundClientEditor.subId" />
            </AFormItem>
            <AFormItem label="Enable">
              <ASwitch v-model:checked="inboundClientEditor.enable" />
            </AFormItem>
          </div>
          <AFormItem label="Comment">
            <AInput v-model:value="inboundClientEditor.comment" />
          </AFormItem>
        </FormSection>

        <FormSection
          eyebrow="Advanced"
          title="Advanced JSON"
          description="Raw legacy JSON remains editable for compatibility and advanced Xray options."
        >
          <div class="form-json-stack">
            <div class="json-section">
              <div class="json-section-title">
                <span>Settings JSON</span>
                <AButton size="small" @click="formatInboundJson('settings')">Format</AButton>
              </div>
              <textarea
                v-model="inboundEditor.settings"
                class="json-editor modal-json-editor"
                spellcheck="false"
              />
            </div>

            <div class="json-section">
              <div class="json-section-title">
                <span>Stream Settings JSON</span>
                <AButton size="small" @click="formatInboundJson('streamSettings')">Format</AButton>
              </div>
              <textarea
                v-model="inboundEditor.streamSettings"
                class="json-editor modal-json-editor"
                spellcheck="false"
              />
            </div>

            <div class="json-section">
              <div class="json-section-title">
                <span>Sniffing JSON</span>
                <AButton size="small" @click="formatInboundJson('sniffing')">Format</AButton>
              </div>
              <textarea
                v-model="inboundEditor.sniffing"
                class="json-editor modal-json-editor"
                spellcheck="false"
              />
            </div>
          </div>
        </FormSection>
      </AForm>
    </AModal>

    <AModal
      v-model:open="clientModalOpen"
      destroy-on-close
      :confirm-loading="savingClient"
      :title="clientModalTitle"
      width="620px"
      @ok="submitClient"
    >
      <AForm layout="vertical">
        <AFormItem label="Email">
          <AInput v-model:value="clientEditor.email" />
        </AFormItem>
        <AFormItem v-if="usesUuidClientId(clientEditor.protocol)" label="UUID">
          <ASpace.Compact class="full-width">
            <AInput v-model:value="clientEditor.id" />
            <AButton @click="clientEditor.id = randomUuid()">Generate</AButton>
          </ASpace.Compact>
        </AFormItem>
        <AFormItem v-if="usesPasswordClientId(clientEditor.protocol)" label="Password">
          <ASpace.Compact class="full-width">
            <AInput v-model:value="clientEditor.password" />
            <AButton @click="clientEditor.password = generateClientCredential()">Generate</AButton>
          </ASpace.Compact>
        </AFormItem>
        <AFormItem v-if="usesAuthClientId(clientEditor.protocol)" label="Auth">
          <ASpace.Compact class="full-width">
            <AInput v-model:value="clientEditor.auth" />
            <AButton @click="clientEditor.auth = generateClientCredential()">Generate</AButton>
          </ASpace.Compact>
        </AFormItem>
        <AFormItem v-if="clientEditor.protocol === 'vmess'" label="Security">
          <ASelect v-model:value="clientEditor.security" :options="securityOptions" />
        </AFormItem>
        <AFormItem v-if="clientEditor.protocol === 'vless'" label="Flow">
          <ASelect v-model:value="clientEditor.flow" :options="flowOptions" />
        </AFormItem>
        <AFormItem v-if="clientEditor.protocol === 'shadowsocks'" label="Method">
          <ASelect
            v-model:value="clientEditor.method"
            disabled
            :options="shadowsocksMethodOptions"
          />
        </AFormItem>
        <template v-if="clientEditor.protocol === 'wireguard'">
          <AFormItem label="Private Key">
            <ASpace.Compact class="full-width">
              <AInput v-model:value="clientEditor.privateKey" />
              <AButton @click="generateWireguardClientKeys">Generate</AButton>
            </ASpace.Compact>
          </AFormItem>
          <AFormItem label="Public Key">
            <AInput v-model:value="clientEditor.publicKey" />
          </AFormItem>
          <AFormItem label="Pre Shared Key">
            <ASpace.Compact class="full-width">
              <AInput v-model:value="clientEditor.preSharedKey" />
              <AButton @click="clientEditor.preSharedKey = generateWireguardPresharedKey()">
                Generate
              </AButton>
            </ASpace.Compact>
          </AFormItem>
          <AFormItem label="Allowed IPs">
            <textarea
              v-model="clientEditor.allowedIPs"
              class="json-editor compact-json-editor"
              spellcheck="false"
            />
          </AFormItem>
          <AFormItem label="Keep Alive">
            <AInputNumber v-model:value="clientEditor.keepAlive" :min="0" class="full-width" />
          </AFormItem>
        </template>
        <div class="form-grid client-form-grid">
          <AFormItem v-if="clientEditor.protocol !== 'wireguard'" label="Traffic Limit GB">
            <AInputNumber v-model:value="clientEditor.totalGB" :min="0" class="full-width" />
          </AFormItem>
          <AFormItem v-if="clientEditor.protocol !== 'wireguard'" label="Expiry Timestamp">
            <AInputNumber v-model:value="clientEditor.expiryTime" :min="0" class="full-width" />
          </AFormItem>
          <AFormItem v-if="clientEditor.protocol !== 'wireguard'" label="IP Limit">
            <AInputNumber v-model:value="clientEditor.limitIp" :min="0" class="full-width" />
          </AFormItem>
          <AFormItem v-if="clientEditor.protocol !== 'wireguard'" label="Reset Days">
            <AInputNumber v-model:value="clientEditor.reset" :min="0" class="full-width" />
          </AFormItem>
          <AFormItem label="Sub ID">
            <AInput v-model:value="clientEditor.subId" />
          </AFormItem>
          <AFormItem label="Enable">
            <ASwitch v-model:checked="clientEditor.enable" />
          </AFormItem>
        </div>
        <AFormItem label="Comment">
          <AInput v-model:value="clientEditor.comment" />
        </AFormItem>
      </AForm>
    </AModal>

    <AModal
      v-model:open="clientIpsModalOpen"
      :footer="null"
      :title="clientIpsModalTitle"
      destroy-on-close
      :focus-trigger-after-close="false"
      :get-container="false"
      :mask-closable="false"
    >
      <textarea
        aria-label="Client IP records"
        class="json-editor compact-json-editor"
        readonly
        :value="clientIpsText"
      />
    </AModal>

    <AModal
      v-model:open="clientAccessModalOpen"
      :title="clientAccessTitle"
      width="880px"
      destroy-on-close
      :focus-trigger-after-close="false"
      :get-container="false"
      :mask-closable="false"
    >
      <template #footer>
        <AButton @click="clientAccessModalOpen = false">Close</AButton>
      </template>
      <template v-if="clientAccessClient">
        <div class="drawer-summary">
          <StatusTile
            label="Email"
            :value="clientAccessClient.email || '-'"
            :hint="clientPrimaryText(selectedInbound, clientAccessClient)"
          />
          <StatusTile
            label="Subscription"
            :value="clientAccessClient.subId || '-'"
            :hint="
              formatTimestamp(
                clientAccessClient.traffic?.expiryTime ?? clientAccessClient.expiryTime,
              )
            "
          />
          <StatusTile
            label="Usage"
            :value="
              formatTraffic(
                clientAccessClient.traffic?.up || 0,
                clientAccessClient.traffic?.down || 0,
                clientAccessClient.traffic?.total || 0,
              )
            "
            :hint="formatLimit(clientAccessClient.traffic?.total ?? clientAccessClient.totalGB)"
          />
          <StatusTile
            label="Last Online"
            :value="formatClientLastOnline(clientAccessClient)"
            :hint="clientAccessClient.enable !== false ? 'Enabled' : 'Disabled'"
          />
        </div>

        <div v-if="clientAccessLinks.length === 0" class="form-section">
          <AAlert message="No access links are available for this client." show-icon type="info" />
        </div>

        <div v-else class="client-link-grid">
          <div
            v-for="(item, index) in clientAccessLinks"
            :key="`${item.kind}-${item.label}`"
            class="client-link-card"
          >
            <div class="client-link-title-row">
              <div>
                <strong>{{ item.label }}</strong>
                <p class="muted-text">
                  {{ item.kind === 'share' ? 'Share link' : 'Subscription endpoint' }}
                </p>
              </div>
              <span class="client-link-format">{{
                item.kind === 'share' ? 'Link' : 'Subscription'
              }}</span>
            </div>
            <AInput :value="item.url" readonly />
            <ASpace class="client-link-actions" wrap>
              <AButton size="small" @click="copyClientAccessLink(item.url)">Copy</AButton>
              <AButton
                size="small"
                :disabled="!isOpenablePublicLink(item.url)"
                @click="openPublicAccessLink(item.url)"
              >
                Open
              </AButton>
            </ASpace>
            <div class="qr-link-card">
              <canvas :id="`client-access-qr-${index}`" class="qr-canvas" />
            </div>
          </div>
        </div>
      </template>
    </AModal>

    <AModal
      v-model:open="copyClientsModalOpen"
      :confirm-loading="copyingClients"
      title="Copy Clients from Other Inbound"
      width="900px"
      destroy-on-close
      :focus-trigger-after-close="false"
      :get-container="false"
      :mask-closable="false"
      @ok="submitCopyClients"
    >
      <AForm layout="vertical">
        <AFormItem label="Source Inbound">
          <ASelect
            v-model:value="copySourceInboundId"
            :options="copyClientSourceOptions"
            placeholder="Select source inbound"
          />
        </AFormItem>
        <AFormItem v-if="selectedInboundFlowOverrideVisible" label="Flow Override">
          <ASelect v-model:value="copyFlowOverride" :options="flowOptions" />
        </AFormItem>
      </AForm>

      <ATable
        v-if="copySourceInbound"
        :columns="copyClientColumns"
        :data-source="copySourceClientRows"
        :pagination="false"
        :row-selection="copyClientRowSelection"
        row-key="key"
        size="small"
      />

      <AAlert
        v-else
        class="mt-16"
        message="Select a source inbound to preview copyable clients."
        show-icon
        type="info"
      />
    </AModal>

    <AModal
      v-model:open="bulkClientModalOpen"
      :confirm-loading="bulkAddingClients"
      title="Bulk Add Clients"
      width="760px"
      destroy-on-close
      :focus-trigger-after-close="false"
      :get-container="false"
      :mask-closable="false"
      @ok="submitBulkAddClients"
    >
      <AForm layout="vertical">
        <div class="form-grid">
          <AFormItem label="Quantity">
            <AInputNumber
              v-model:value="bulkClientForm.quantity"
              :min="1"
              :max="500"
              class="full-width"
            />
          </AFormItem>
          <AFormItem label="First Index">
            <AInputNumber v-model:value="bulkClientForm.firstIndex" :min="1" class="full-width" />
          </AFormItem>
          <AFormItem label="Email Prefix">
            <AInput v-model:value="bulkClientForm.emailPrefix" placeholder="client-" />
          </AFormItem>
          <AFormItem label="Email Postfix">
            <AInput v-model:value="bulkClientForm.emailPostfix" placeholder="@example.com" />
          </AFormItem>
          <AFormItem label="Traffic Limit GB">
            <AInputNumber v-model:value="bulkClientForm.totalGB" :min="0" class="full-width" />
          </AFormItem>
          <AFormItem label="Expiry Timestamp">
            <AInputNumber v-model:value="bulkClientForm.expiryTime" :min="0" class="full-width" />
          </AFormItem>
          <AFormItem label="IP Limit">
            <AInputNumber v-model:value="bulkClientForm.limitIp" :min="0" class="full-width" />
          </AFormItem>
          <AFormItem label="Reset Days">
            <AInputNumber v-model:value="bulkClientForm.reset" :min="0" class="full-width" />
          </AFormItem>
          <AFormItem v-if="selectedInboundFlowOverrideVisible" label="Flow">
            <ASelect v-model:value="bulkClientForm.flow" :options="flowOptions" />
          </AFormItem>
        </div>
      </AForm>
    </AModal>

    <AModal
      v-model:open="importModalOpen"
      :confirm-loading="importingInbound"
      destroy-on-close
      title="Import Inbound JSON"
      width="760px"
      @ok="submitImportInbound"
    >
      <AAlert
        class="mb-12"
        message="Paste a legacy inbound JSON object. It will be imported through the existing Xray API so old UI remains readable."
        show-icon
        type="info"
      />
      <textarea
        v-model="importInboundText"
        aria-label="Inbound import JSON"
        class="json-editor modal-json-editor"
        spellcheck="false"
      />
    </AModal>
  </section>
</template>

<script setup lang="ts">
import {
  BlockOutlined,
  CopyOutlined,
  DeleteOutlined,
  DownloadOutlined,
  EditOutlined,
  EllipsisOutlined,
  EyeOutlined,
  LinkOutlined,
  PlusOutlined,
  QrcodeOutlined,
  ReloadOutlined,
  UserAddOutlined,
} from '@ant-design/icons-vue';
import {
  Alert as AAlert,
  Button as AButton,
  Card as ACard,
  Drawer as ADrawer,
  Dropdown as ADropdown,
  Form as AForm,
  FormItem as AFormItem,
  Input as AInput,
  InputNumber as AInputNumber,
  Menu as AMenu,
  Modal as AModal,
  Select as ASelect,
  Space as ASpace,
  Switch as ASwitch,
  Table as ATable,
  Tag as ATag,
  message,
} from 'ant-design-vue';
import type { ItemType } from 'ant-design-vue';
import type { MenuInfo } from 'ant-design-vue/es/menu/src/interface';
import { computed, nextTick, onMounted, reactive, ref, watch } from 'vue';

import {
  addInbound,
  addInboundClient,
  clearClientIps,
  copyInboundClients,
  deleteDepletedInboundClients,
  deleteInbound,
  deleteInboundClient,
  getClientIps,
  getClientsLastOnline,
  getOnlineClients,
  importInbound,
  listInbounds,
  resetAllInboundTraffics,
  resetAllInboundClientTraffics,
  resetInboundClientTraffic,
  updateInbound,
  updateInboundClient,
} from '@/api/inbounds';
import { getAllSettings, getDefaultSettings } from '@/api/settings';
import PageHeader from '@/components/PageHeader.vue';
import FormSection from '@/components/FormSection.vue';
import StatusTile from '@/components/StatusTile.vue';
import {
  getProtocolRegistryEntry,
  isRegisteredEditableProtocol,
  protocolSupportsClients,
  protocolSupportsShareLink,
  protocolSupportsStream,
  xrayEditableProtocols,
} from '@/schemas/protocolRegistry';
import { getRuntimeConfig, hasInjectedRuntimeConfig } from '@/types/runtime';
import { translate, translateDomText } from '@/i18n/messages';
import { useAppStore } from '@/stores/app';
import type {
  ClientTraffic,
  Inbound,
  InboundClient,
  InboundForm,
  XrayEditableInboundProtocol,
} from '@/types/inbound';
import type { PanelSettings } from '@/types/settings';
import { formatBytes, formatCount } from '@/utils/format';
import {
  SHADOWSOCKS_METHOD_OPTIONS,
  buildClientShareLink,
  buildClientSubscriptionLinks,
  buildInboundShareLinks,
  defaultInboundSettings,
  defaultSniffingSettings,
  defaultStreamSettings,
  generateBulkClientProfiles,
  generateShadowsocksPassword,
  generateWireguardKeypair,
  generateWireguardPeer,
  generateWireguardPresharedKey,
  getClientPrimaryId,
  getInboundClients,
  getInboundNetwork,
  getInboundSecurity,
  getShadowsocksMethod,
  isShadowsocks2022Method,
  isSingleUserShadowsocks2022,
  mergeSubscriptionEndpointDefaults,
  parseInboundSettings,
  parseInboundSniffingSettings,
  parseInboundStreamSettings,
  resolveInboundHost,
  stringifyJson,
} from '@/utils/inboundCompat';
import {
  normalizeRealityServerSettings,
  validateRealityServerSettings,
} from '@/utils/realitySettings';
import { copyText, downloadText } from '@/utils/textExport';

type InboundModalMode = 'create' | 'edit';
type ClientModalMode = 'create' | 'edit';
type InboundJsonField = 'settings' | 'streamSettings' | 'sniffing';
type StateFilter = 'all' | 'enabled' | 'disabled';
type GatewayProxyTemplate = 'mixed' | 'http';
type HeaderPrimaryActionKey = 'refresh' | 'refreshActivity' | 'newInbound';
type HeaderMoreActionKey =
  | 'importJson'
  | 'exportAll'
  | 'exportAllSubscriptions'
  | 'gatewaySocks'
  | 'gatewayHttp'
  | 'resetAllTraffic'
  | 'resetAllClients'
  | 'deleteDepleted';
type HeaderActionKey = HeaderPrimaryActionKey | HeaderMoreActionKey;

const appStore = useAppStore();

interface InboundEditorState {
  id?: number;
  protocol: XrayEditableInboundProtocol;
  remark: string;
  listen: string;
  port: number;
  enable: boolean;
  totalGB: number;
  expiryTime: number;
  trafficReset: string;
  settings: string;
  streamSettings: string;
  sniffing: string;
}

interface WireguardEditorState {
  mtu: number;
  secretKey: string;
  pubKey: string;
  noKernelTun: boolean;
}

interface StreamEditorState {
  network: string;
  security: string;
  tcpAcceptProxyProtocol: boolean;
  tcpHeaderType: string;
  kcpMtu: number;
  kcpTti: number;
  kcpUplinkCapacity: number;
  kcpDownlinkCapacity: number;
  kcpCwndMultiplier: number;
  kcpMaxSendingWindow: number;
  wsAcceptProxyProtocol: boolean;
  wsPath: string;
  wsHost: string;
  wsHeartbeatPeriod: number;
  grpcServiceName: string;
  grpcAuthority: string;
  grpcMultiMode: boolean;
  httpupgradeAcceptProxyProtocol: boolean;
  httpupgradePath: string;
  httpupgradeHost: string;
  xhttpPath: string;
  xhttpHost: string;
  xhttpMode: string;
  xhttpNoSseHeader: boolean;
  xhttpScMaxBufferedPosts: number;
  xhttpScMaxEachPostBytes: string;
  xhttpScStreamUpServerSecs: string;
  xhttpXPaddingBytes: string;
  tlsServerName: string;
  tlsMinVersion: string;
  tlsMaxVersion: string;
  tlsAlpn: string;
  tlsFingerprint: string;
  tlsCertificateFile: string;
  tlsKeyFile: string;
  tlsRejectUnknownSni: boolean;
  tlsDisableSystemRoot: boolean;
  tlsEnableSessionResumption: boolean;
  tlsEchServerKeys: string;
  tlsEchConfigList: string;
  realityShow: boolean;
  realityXver: number;
  realityTarget: string;
  realityServerNames: string;
  realityPrivateKey: string;
  realityShortIds: string;
  realityMinClientVer: string;
  realityMaxClientVer: string;
  realityMaxTimediff: number;
  realityPublicKey: string;
  realitySpiderX: string;
  realityMldsa65Seed: string;
  realityMldsa65Verify: string;
  hysteriaAuth: string;
  hysteriaUdpIdleTimeout: number;
  sockoptEnabled: boolean;
  sockoptAcceptProxyProtocol: boolean;
  sockoptTcpFastOpen: boolean;
  sockoptTcpMptcp: boolean;
  sockoptPenetrate: boolean;
  sockoptV6Only: boolean;
  sockoptDomainStrategy: string;
  sockoptTcpCongestion: string;
  sockoptTproxy: string;
  sockoptMark: number;
  sockoptTcpMaxSeg: number;
  sockoptDialerProxy: string;
  sockoptInterfaceName: string;
  sockoptTrustedXForwardedFor: string;
}

interface ClientEditorState {
  protocol: XrayEditableInboundProtocol;
  originalClientId: string;
  id: string;
  password: string;
  method: string;
  auth: string;
  privateKey: string;
  publicKey: string;
  preSharedKey: string;
  allowedIPs: string;
  keepAlive: number;
  email: string;
  security: string;
  flow: string;
  limitIp: number;
  totalGB: number;
  expiryTime: number;
  enable: boolean;
  subId: string;
  comment: string;
  reset: number;
}

interface ClientRow extends InboundClient {
  key: string;
  traffic?: ClientTraffic;
}

interface GatewayProxyUriItem {
  label: string;
  uri: string;
}

interface AccessLinkItem {
  kind: 'share' | 'subscription';
  label: string;
  url: string;
}

interface CopyClientSourceRow extends ClientRow {
  expiryLabel: string;
  trafficLabel: string;
}

interface BulkClientFormState {
  quantity: number;
  firstIndex: number;
  emailPrefix: string;
  emailPostfix: string;
  totalGB: number;
  expiryTime: number;
  limitIp: number;
  reset: number;
  flow: string;
}

type SubscriptionLinkSettings = Pick<
  PanelSettings,
  'subEnable' | 'subJsonEnable' | 'subClashEnable' | 'subURI' | 'subJsonURI' | 'subClashURI'
>;

type QriousConstructor = new (options: {
  element: HTMLCanvasElement;
  size: number;
  value: string;
}) => unknown;

let qriousLoader: Promise<QriousConstructor> | null = null;

const inbounds = ref<Inbound[]>([]);
const loading = ref(false);
const error = ref('');
const keyword = ref('');
const protocolFilter = ref('all');
const stateFilter = ref<StateFilter>('all');
const moreActionsOpen = ref(false);
const detailOpen = ref(false);
const selectedInbound = ref<Inbound | null>(null);
const sharePreview = ref('');
const sharePreviewTitle = ref('Share Links');
const sharePreviewFilename = ref('inbounds-export.txt');
const inboundModalOpen = ref(false);
const inboundModalMode = ref<InboundModalMode>('create');
const savingInbound = ref(false);
const busyInboundId = ref<number | null>(null);
const importModalOpen = ref(false);
const importInboundText = ref('');
const importingInbound = ref(false);
const resettingAllTraffic = ref(false);
const resettingAllClientTraffic = ref(false);
const clientModalOpen = ref(false);
const clientModalMode = ref<ClientModalMode>('create');
const clientInbound = ref<Inbound | null>(null);
const selectedClientRowKeys = ref<string[]>([]);
const savingClient = ref(false);
const busyClient = ref(false);
const busyClientKey = ref('');
const deletingDepletedClients = ref(false);
const deletingAllDepletedClients = ref(false);
const onlineClients = ref<string[]>([]);
const lastOnlineMap = ref<Record<string, number>>({});
const loadingActivity = ref(false);
const loadingSubscriptionSettings = ref(false);
const clientIpsModalOpen = ref(false);
const clientIpsModalTitle = ref('Client IP Records');
const clientIpsText = ref('');
const clearingClientIpsEmail = ref('');
const subscriptionSettings = ref<SubscriptionLinkSettings | null>(null);
const clientAccessModalOpen = ref(false);
const clientAccessTitle = ref('Client Access');
const clientAccessClient = ref<ClientRow | null>(null);
const clientAccessLinks = ref<AccessLinkItem[]>([]);
const bulkClientModalOpen = ref(false);
const bulkAddingClients = ref(false);
const copyClientsModalOpen = ref(false);
const copyingClients = ref(false);
const copySourceInboundId = ref<number>();
const copySelectedClientKeys = ref<string[]>([]);
const copyFlowOverride = ref('');

const headerPrimaryActionKeys: HeaderPrimaryActionKey[] = [
  'refresh',
  'refreshActivity',
  'newInbound',
];
const menuDangerActionKeys = new Set<HeaderActionKey>([
  'resetAllTraffic',
  'resetAllClients',
  'deleteDepleted',
]);

const inboundEditor = reactive<InboundEditorState>(createInboundEditor());
const wireguardEditor = reactive<WireguardEditorState>(createWireguardEditor());
const streamEditor = reactive<StreamEditorState>(createStreamEditor());
const inboundClientEditor = reactive<ClientEditorState>(createClientEditor());
const clientEditor = reactive<ClientEditorState>(createClientEditor());
const bulkClientForm = reactive<BulkClientFormState>(createBulkClientForm());

const editableProtocolOptions = xrayEditableProtocols.map((protocol) => {
  const entry = getProtocolRegistryEntry(protocol);
  return {
    label: entry?.label || protocol,
    value: protocol,
  };
});
const protocolFilterOptions = computed(() => [
  { label: translateDomText('All protocols', appStore.locale), value: 'all' },
  ...Array.from(new Set(inbounds.value.map((inbound) => inbound.protocol))).map((protocol) => ({
    label: protocol,
    value: protocol,
  })),
]);
const stateFilterOptions = computed(() => [
  { label: translateDomText('All states', appStore.locale), value: 'all' },
  { label: translateDomText('Enabled', appStore.locale), value: 'enabled' },
  { label: translateDomText('Disabled', appStore.locale), value: 'disabled' },
]);
const moreActionItems = computed<ItemType[]>(() => [
  { key: 'importJson', label: 'Import JSON' },
  {
    key: 'exportAll',
    label: 'Export All',
    disabled: inbounds.value.length === 0,
  },
  {
    key: 'exportAllSubscriptions',
    label: 'Export All Subscriptions',
    disabled: inbounds.value.length === 0 || loadingSubscriptionSettings.value,
  },
  { key: 'gatewaySocks', label: 'Gateway SOCKS5' },
  { key: 'gatewayHttp', label: 'Gateway HTTP' },
  { type: 'divider' },
  {
    key: 'resetAllTraffic',
    label: 'Reset All Traffic',
    danger: menuDangerActionKeys.has('resetAllTraffic'),
    disabled: resettingAllTraffic.value,
  },
  {
    key: 'resetAllClients',
    label: 'Reset All Clients',
    danger: menuDangerActionKeys.has('resetAllClients'),
    disabled: resettingAllClientTraffic.value,
  },
  {
    key: 'deleteDepleted',
    label: 'Delete Depleted Clients',
    danger: menuDangerActionKeys.has('deleteDepleted'),
    disabled: deletingAllDepletedClients.value,
  },
]);
const securityOptions = ['auto', 'aes-128-gcm', 'chacha20-poly1305', 'none', 'zero'].map(
  (value) => ({
    label: value,
    value,
  }),
);
const flowOptions = [
  { label: 'none', value: '' },
  { label: 'xtls-rprx-vision', value: 'xtls-rprx-vision' },
  { label: 'xtls-rprx-vision-udp443', value: 'xtls-rprx-vision-udp443' },
];
const shadowsocksMethodOptions = SHADOWSOCKS_METHOD_OPTIONS.map((value) => ({
  label: value,
  value,
}));
const transportNetworkOptions = computed(() =>
  isHysteriaProtocol(inboundEditor.protocol)
    ? [{ label: 'hysteria', value: 'hysteria' }]
    : [
        { label: 'tcp', value: 'tcp' },
        { label: 'kcp', value: 'kcp' },
        { label: 'ws', value: 'ws' },
        { label: 'grpc', value: 'grpc' },
        { label: 'httpupgrade', value: 'httpupgrade' },
        { label: 'xhttp', value: 'xhttp' },
      ],
);
const transportSecurityOptions = computed(() =>
  isHysteriaProtocol(inboundEditor.protocol)
    ? [{ label: 'tls', value: 'tls' }]
    : [
        { label: 'none', value: 'none' },
        { label: 'tls', value: 'tls' },
        { label: 'reality', value: 'reality' },
      ],
);
const xhttpModeOptions = ['auto', 'packet-up', 'stream-up', 'stream-one'].map((value) => ({
  label: value,
  value,
}));
const tlsVersionOptions = ['1.0', '1.1', '1.2', '1.3'].map((value) => ({
  label: value,
  value,
}));
const sockoptDomainStrategyOptions = ['AsIs', 'UseIP', 'UseIPv4', 'UseIPv6'].map((value) => ({
  label: value,
  value,
}));
const sockoptTcpCongestionOptions = ['bbr', 'cubic'].map((value) => ({
  label: value,
  value,
}));
const sockoptTproxyOptions = ['off', 'redirect', 'tproxy'].map((value) => ({
  label: value,
  value,
}));
const inboundColumns = [
  { title: 'Inbound', key: 'inbound' },
  { title: 'Address', key: 'address' },
  { title: 'Transport', key: 'transport' },
  { title: 'Traffic', key: 'traffic' },
  { title: 'Clients', key: 'clients', width: 92 },
  { title: 'Enabled', key: 'enable', width: 96 },
  { title: 'Actions', key: 'actions', width: 260 },
];
const clientColumns = [
  { title: 'Client', key: 'client' },
  { title: 'Limit', key: 'limit', width: 110 },
  { title: 'Usage', key: 'usage', width: 150 },
  { title: 'Expiry', key: 'expiry', width: 150 },
  { title: 'Enabled', key: 'enable', width: 90 },
  { title: 'Actions', key: 'actions', width: 330 },
];
const copyClientColumns = [
  { title: 'Client', dataIndex: 'email', key: 'email' },
  { title: 'Traffic', dataIndex: 'trafficLabel', key: 'trafficLabel', width: 180 },
  { title: 'Expiry', dataIndex: 'expiryLabel', key: 'expiryLabel', width: 180 },
];
const activityColumns = [
  { title: 'Client', key: 'client' },
  { title: 'Online', key: 'online', width: 96 },
  { title: 'Last Online', key: 'lastOnline', width: 170 },
  { title: 'IP Records', key: 'actions', width: 190 },
];

const filteredInbounds = computed(() => {
  const term = keyword.value.trim().toLowerCase();
  return inbounds.value.filter((inbound) => {
    if (protocolFilter.value !== 'all' && inbound.protocol !== protocolFilter.value) {
      return false;
    }
    if (stateFilter.value === 'enabled' && !inbound.enable) {
      return false;
    }
    if (stateFilter.value === 'disabled' && inbound.enable) {
      return false;
    }
    if (!term) {
      return true;
    }
    return [inbound.remark, inbound.tag, inbound.listen, inbound.protocol, String(inbound.port)]
      .join(' ')
      .toLowerCase()
      .includes(term);
  });
});
const enabledInboundCount = computed(
  () => inbounds.value.filter((inbound) => inbound.enable).length,
);
const clientCount = computed(() =>
  inbounds.value.reduce((total, inbound) => total + inboundClientCount(inbound), 0),
);
const trafficTotal = computed(() =>
  formatBytes(inbounds.value.reduce((total, inbound) => total + inbound.up + inbound.down, 0)),
);
const selectedInboundTitle = computed(() => {
  const inbound = selectedInbound.value;
  if (!inbound) {
    return 'Inbound Details';
  }
  return inbound.remark || inbound.tag || `Inbound ${inbound.id}`;
});
const selectedInboundClientManageable = computed(() =>
  selectedInbound.value ? protocolSupportsClients(selectedInbound.value.protocol) : false,
);
const selectedClientRows = computed<ClientRow[]>(() => {
  const inbound = selectedInbound.value;
  if (!inbound) {
    return [];
  }
  return buildClientRows(inbound);
});
const selectedBatchClientRows = computed(() =>
  selectedClientRows.value.filter((row) => selectedClientRowKeys.value.includes(row.key)),
);
const copyClientSourceOptions = computed(() =>
  inbounds.value
    .filter(
      (inbound) =>
        selectedInbound.value &&
        inbound.id !== selectedInbound.value.id &&
        protocolSupportsClients(inbound.protocol),
    )
    .map((inbound) => ({
      label: inbound.remark || inbound.tag || `Inbound ${inbound.id}`,
      value: inbound.id,
    })),
);
const copySourceInbound = computed(() =>
  copySourceInboundId.value
    ? inbounds.value.find((inbound) => inbound.id === copySourceInboundId.value) || null
    : null,
);
const copySourceClientRows = computed<CopyClientSourceRow[]>(() => {
  const inbound = copySourceInbound.value;
  if (!inbound) {
    return [];
  }
  return buildClientRows(inbound).map((row) => ({
    ...row,
    trafficLabel: formatTraffic(
      row.traffic?.up || 0,
      row.traffic?.down || 0,
      row.traffic?.total || 0,
    ),
    expiryLabel: formatTimestamp(row.traffic?.expiryTime ?? row.expiryTime),
  }));
});
const copySelectedSourceRows = computed(() =>
  copySourceClientRows.value.filter((row) => copySelectedClientKeys.value.includes(row.key)),
);
const resettableSelectedClientRows = computed(() =>
  selectedBatchClientRows.value.filter((row) => !clientResetDisabled(selectedInbound.value, row)),
);
const canResetSelectedClients = computed(
  () => resettableSelectedClientRows.value.length > 0 && !busyClient.value,
);
const canDeleteSelectedClients = computed(() => {
  if (!selectedInboundClientManageable.value || busyClient.value) {
    return false;
  }
  const selectedCount = selectedBatchClientRows.value.length;
  return selectedCount > 0 && selectedCount < selectedClientRows.value.length;
});
const clientRowSelection = computed(() => ({
  selectedRowKeys: selectedClientRowKeys.value,
  onChange: (keys: Array<string | number>) => {
    selectedClientRowKeys.value = keys.map(String);
  },
  getCheckboxProps: (record: ClientRow) => ({
    disabled: !hasClientPrimaryId(selectedInbound.value, record),
  }),
}));
const copyClientRowSelection = computed(() => ({
  selectedRowKeys: copySelectedClientKeys.value,
  onChange: (keys: Array<string | number>) => {
    copySelectedClientKeys.value = keys.map(String);
  },
}));
const inboundModalTitle = computed(() =>
  inboundModalMode.value === 'create' ? 'New Inbound' : 'Edit Inbound',
);
const inboundClientSectionVisible = computed(
  () => protocolSupportsClients(inboundEditor.protocol) && inboundEditor.protocol !== 'wireguard',
);
const inboundVlessFlowVisible = computed(
  () =>
    inboundEditor.protocol === 'vless' &&
    streamEditor.network === 'tcp' &&
    (streamEditor.security === 'tls' || streamEditor.security === 'reality'),
);
const gatewayProxyUris = computed<GatewayProxyUriItem[]>(() => {
  if (inboundEditor.protocol !== 'mixed' && inboundEditor.protocol !== 'http') {
    return [];
  }
  const host = inboundEditor.listen.trim() || '127.0.0.1';
  const port = Math.max(1, Math.min(65535, Number(inboundEditor.port || 0)));
  if (!port) {
    return [];
  }
  const settings = parseInboundSettingsText(inboundEditor.settings);
  const account = Array.isArray(settings.accounts) ? objectField(settings.accounts[0]) : {};
  const user = encodeUriCredential(stringField(account.user));
  const pass = encodeUriCredential(stringField(account.pass));
  const auth = user && pass ? `${user}:${pass}@` : '';
  if (inboundEditor.protocol === 'mixed') {
    return [
      { label: 'SOCKS5', uri: `socks5://${auth}${host}:${port}` },
      { label: 'SOCKS5H', uri: `socks5h://${auth}${host}:${port}` },
    ];
  }
  return [{ label: 'HTTP', uri: `http://${auth}${host}:${port}` }];
});
const selectedInboundFlowOverrideVisible = computed(() => {
  const inbound = selectedInbound.value;
  if (!inbound || inbound.protocol !== 'vless') {
    return false;
  }
  return inboundNetwork(inbound) === 'tcp' && ['tls', 'reality'].includes(inboundSecurity(inbound));
});
const clientModalTitle = computed(() =>
  clientModalMode.value === 'create' ? 'Add Client' : 'Edit Client',
);

watch(
  () => inboundEditor.protocol,
  (protocol) => {
    if (inboundModalMode.value === 'create') {
      inboundEditor.settings = stringifyJson(defaultInboundSettings(protocol));
      inboundEditor.streamSettings = stringifyJson(defaultStreamSettings(protocol));
      Object.assign(inboundClientEditor, createClientEditor(protocol));
      syncWireguardEditorFromSettings();
      syncStreamEditorFromSettings();
      syncInboundClientEditorFromSettings();
    }
  },
);

watch(selectedClientRows, (rows) => {
  const keys = new Set(rows.map((row) => row.key));
  selectedClientRowKeys.value = selectedClientRowKeys.value.filter((key) => keys.has(key));
});

watch(copySourceClientRows, (rows) => {
  const keys = new Set(rows.map((row) => row.key));
  copySelectedClientKeys.value = copySelectedClientKeys.value.filter((key) => keys.has(key));
});

watch(clientAccessModalOpen, (open) => {
  if (!open) {
    clientAccessClient.value = null;
    clientAccessLinks.value = [];
    return;
  }
  void renderClientAccessQrs();
});

function handleMoreActionClick({ key }: MenuInfo) {
  const actionKey = String(key) as HeaderMoreActionKey;
  moreActionsOpen.value = false;
  switch (actionKey) {
    case 'importJson':
      openImportInbound();
      break;
    case 'exportAll':
      void exportAllInboundShareLinks();
      break;
    case 'exportAllSubscriptions':
      void exportAllInboundSubscriptionLinks();
      break;
    case 'gatewaySocks':
      openGatewayProxyTemplate('mixed');
      break;
    case 'gatewayHttp':
      openGatewayProxyTemplate('http');
      break;
    case 'resetAllTraffic':
      confirmResetAllInboundTraffic();
      break;
    case 'resetAllClients':
      confirmResetAllInboundClientTraffic();
      break;
    case 'deleteDepleted':
      confirmDeleteAllDepletedClients();
      break;
  }
}

async function refreshInbounds() {
  if (!hasInjectedRuntimeConfig()) {
    return;
  }

  loading.value = true;
  error.value = '';
  try {
    inbounds.value = await listInbounds({ notifyOnError: false });
    syncSelectedInbound();
    await refreshClientActivity();
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to load inbounds';
  } finally {
    loading.value = false;
  }
}

async function refreshClientActivity() {
  if (!hasInjectedRuntimeConfig()) {
    return;
  }

  loadingActivity.value = true;
  try {
    const [online, lastOnline] = await Promise.all([
      getOnlineClients({ notifyOnError: false }),
      getClientsLastOnline({ notifyOnError: false }),
    ]);
    onlineClients.value = Array.isArray(online) ? online : [];
    lastOnlineMap.value = lastOnline || {};
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to load client activity';
  } finally {
    loadingActivity.value = false;
  }
}

function openCreateInbound() {
  inboundModalMode.value = 'create';
  Object.assign(inboundEditor, createInboundEditor());
  Object.assign(inboundClientEditor, createClientEditor(inboundEditor.protocol));
  syncWireguardEditorFromSettings();
  syncStreamEditorFromSettings();
  syncInboundClientEditorFromSettings();
  inboundModalOpen.value = true;
}

/** 打开 Gateway 本机代理出口模板，并预填固定回环监听地址与端口。 */
function openGatewayProxyTemplate(template: GatewayProxyTemplate) {
  const protocol: XrayEditableInboundProtocol = template === 'mixed' ? 'mixed' : 'http';
  Object.assign(
    inboundEditor,
    createInboundEditor(protocol, {
      remark:
        template === 'mixed' ? 'gateway-socks5-127.0.0.1-1080' : 'gateway-http-127.0.0.1-8081',
      listen: '127.0.0.1',
      port: template === 'mixed' ? 1080 : 8081,
      settings: stringifyJson(gatewayProxySettings(template)),
    }),
  );
  inboundModalMode.value = 'create';
  Object.assign(inboundClientEditor, createClientEditor(protocol));
  syncWireguardEditorFromSettings();
  syncStreamEditorFromSettings();
  syncInboundClientEditorFromSettings();
  inboundModalOpen.value = true;
}

function openImportInbound() {
  importInboundText.value = JSON.stringify(
    {
      down: 0,
      enable: false,
      expiryTime: 0,
      listen: '127.0.0.1',
      port: 443,
      protocol: 'vless',
      remark: 'imported-inbound',
      settings: JSON.stringify({ clients: [], decryption: 'none' }),
      sniffing: JSON.stringify({ destOverride: [], enabled: false }),
      streamSettings: JSON.stringify({
        network: 'tcp',
        security: 'none',
        tcpSettings: { acceptProxyProtocol: false, header: { type: 'none' } },
      }),
      total: 0,
      up: 0,
    },
    null,
    2,
  );
  importModalOpen.value = true;
}

async function submitImportInbound() {
  const normalized = normalizeJsonEditorText(importInboundText.value, 'Inbound import JSON');
  if (!normalized) {
    return;
  }

  importingInbound.value = true;
  try {
    await importInbound(normalized, { notifyOnError: false });
    importModalOpen.value = false;
    void message.success('Inbound imported');
    await refreshInbounds();
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to import inbound';
  } finally {
    importingInbound.value = false;
  }
}

function openEditInbound(record: Inbound | Record<string, unknown>) {
  const inbound = asInbound(record);
  inboundModalMode.value = 'edit';
  Object.assign(inboundEditor, {
    id: inbound.id,
    protocol: isEditableProtocol(inbound.protocol) ? inbound.protocol : 'vless',
    remark: inbound.remark,
    listen: inbound.listen,
    port: inbound.port,
    enable: inbound.enable,
    totalGB: bytesToGb(inbound.total),
    expiryTime: inbound.expiryTime || 0,
    trafficReset: inbound.trafficReset || 'never',
    settings: formatJsonText(inbound.settings, parseInboundSettings(inbound)),
    streamSettings: formatJsonText(inbound.streamSettings, parseInboundStreamSettings(inbound)),
    sniffing: formatJsonText(inbound.sniffing, parseInboundSniffingSettings(inbound)),
  });
  syncWireguardEditorFromSettings();
  syncStreamEditorFromSettings();
  syncInboundClientEditorFromSettings();
  inboundModalOpen.value = true;
}

async function submitInbound() {
  if (inboundEditor.protocol === 'wireguard') {
    applyWireguardEditorToSettings();
  } else if (protocolSupportsStream(inboundEditor.protocol)) {
    applyStreamEditorToSettings();
  }
  if (inboundClientSectionVisible.value) {
    applyInboundClientEditorToSettings();
  }
  const settings = normalizeJsonEditorText(inboundEditor.settings, 'Settings JSON');
  const streamSettings = normalizeJsonEditorText(
    inboundEditor.streamSettings,
    'Stream Settings JSON',
  );
  const sniffing = normalizeJsonEditorText(inboundEditor.sniffing, 'Sniffing JSON');
  if (!settings || !streamSettings || !sniffing) {
    return;
  }
  if (!inboundEditor.port || inboundEditor.port < 1 || inboundEditor.port > 65535) {
    error.value = 'Port must be between 1 and 65535';
    return;
  }
  const inboundValidationError = validateInboundEditorSettings(settings, streamSettings);
  if (inboundValidationError) {
    error.value = inboundValidationError;
    return;
  }

  const existing = inboundEditor.id
    ? inbounds.value.find((item) => item.id === inboundEditor.id)
    : undefined;
  const payload: InboundForm = {
    id: inboundEditor.id,
    up: existing?.up || 0,
    down: existing?.down || 0,
    total: gbToBytes(inboundEditor.totalGB),
    allTime: existing?.allTime || 0,
    remark: inboundEditor.remark.trim(),
    enable: inboundEditor.enable,
    expiryTime: inboundEditor.expiryTime || 0,
    trafficReset: inboundEditor.trafficReset.trim() || 'never',
    lastTrafficResetTime: existing?.lastTrafficResetTime || 0,
    listen: inboundEditor.listen.trim(),
    port: inboundEditor.port,
    protocol: inboundEditor.protocol,
    settings,
    streamSettings,
    sniffing,
  };

  savingInbound.value = true;
  error.value = '';
  try {
    if (inboundModalMode.value === 'edit' && inboundEditor.id) {
      await updateInbound(inboundEditor.id, payload, { notifyOnError: false });
      void message.success('Inbound updated');
    } else {
      await addInbound(payload, { notifyOnError: false });
      void message.success('Inbound created');
    }
    inboundModalOpen.value = false;
    await refreshInbounds();
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to save inbound';
  } finally {
    savingInbound.value = false;
  }
}

function formatInboundJson(field: InboundJsonField) {
  const formatted = normalizeJsonEditorText(inboundEditor[field], `${field} JSON`);
  if (formatted) {
    inboundEditor[field] = formatted;
    if (field === 'settings') {
      syncWireguardEditorFromSettings();
    }
    if (field === 'streamSettings') {
      syncStreamEditorFromSettings();
    }
  }
}

async function toggleInbound(record: Inbound | Record<string, unknown>, enabled: boolean) {
  const inbound = asInbound(record);
  busyInboundId.value = inbound.id;
  error.value = '';
  try {
    await updateInbound(inbound.id, { ...inbound, enable: enabled }, { notifyOnError: false });
    void message.success(enabled ? 'Inbound enabled' : 'Inbound disabled');
    await refreshInbounds();
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to update inbound';
  } finally {
    busyInboundId.value = null;
  }
}

function confirmDeleteInbound(record: Inbound | Record<string, unknown>) {
  const inbound = asInbound(record);
  AModal.confirm({
    title: `Delete ${inbound.remark || inbound.tag || inbound.id}?`,
    content: 'This uses the existing Xray inbound delete endpoint.',
    okButtonProps: { danger: true },
    okText: 'Delete',
    onOk: () => runDeleteInbound(inbound),
  });
}

async function runDeleteInbound(inbound: Inbound) {
  error.value = '';
  try {
    await deleteInbound(inbound.id, { notifyOnError: false });
    void message.success('Inbound deleted');
    if (selectedInbound.value?.id === inbound.id) {
      detailOpen.value = false;
      selectedInbound.value = null;
    }
    await refreshInbounds();
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to delete inbound';
  }
}

function confirmResetAllInboundTraffic() {
  AModal.confirm({
    title: 'Reset all inbound traffic?',
    content: 'This resets stored inbound traffic counters through the existing Xray API.',
    okButtonProps: { danger: true },
    okText: 'Reset All Traffic',
    onOk: runResetAllInboundTraffic,
  });
}

async function runResetAllInboundTraffic() {
  resettingAllTraffic.value = true;
  error.value = '';
  try {
    await resetAllInboundTraffics({ notifyOnError: false });
    void message.success('All inbound traffic reset');
    await refreshInbounds();
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to reset all inbound traffic';
  } finally {
    resettingAllTraffic.value = false;
  }
}

function confirmResetAllInboundClientTraffic() {
  AModal.confirm({
    title: 'Reset all client traffic?',
    content: 'This resets every client traffic counter through the existing Xray API.',
    okButtonProps: { danger: true },
    okText: 'Reset All Clients',
    onOk: runResetAllInboundClientTraffic,
  });
}

async function runResetAllInboundClientTraffic() {
  resettingAllClientTraffic.value = true;
  error.value = '';
  try {
    await resetAllInboundClientTraffics(-1, { notifyOnError: false });
    void message.success('All client traffic reset');
    await refreshInbounds();
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to reset all client traffic';
  } finally {
    resettingAllClientTraffic.value = false;
  }
}

function confirmDeleteAllDepletedClients() {
  AModal.confirm({
    title: 'Delete all depleted clients?',
    content: 'Clients that exhausted their traffic limit will be removed from all inbounds.',
    okButtonProps: { danger: true },
    okText: 'Delete Depleted',
    onOk: runDeleteAllDepletedClients,
  });
}

async function runDeleteAllDepletedClients() {
  deletingAllDepletedClients.value = true;
  error.value = '';
  try {
    await deleteDepletedInboundClients(-1, { notifyOnError: false });
    void message.success('All depleted clients deleted');
    await refreshInbounds();
  } catch (caught) {
    error.value =
      caught instanceof Error ? caught.message : 'Failed to delete all depleted clients';
  } finally {
    deletingAllDepletedClients.value = false;
  }
}

function confirmResetInboundTraffic(inbound: Inbound) {
  AModal.confirm({
    title: `Reset traffic for ${inbound.remark || inbound.tag || inbound.id}?`,
    content: 'This resets the selected inbound up/down counters.',
    okButtonProps: { danger: true },
    okText: 'Reset',
    onOk: () => runResetInboundTraffic(inbound),
  });
}

async function runResetInboundTraffic(inbound: Inbound) {
  busyInboundId.value = inbound.id;
  error.value = '';
  try {
    await updateInbound(inbound.id, { ...inbound, up: 0, down: 0 }, { notifyOnError: false });
    void message.success('Inbound traffic reset');
    await refreshInbounds();
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to reset inbound traffic';
  } finally {
    busyInboundId.value = null;
  }
}

function openInboundDetail(record: Inbound | Record<string, unknown>) {
  const inbound = asInbound(record);
  selectedInbound.value = inbound;
  selectedClientRowKeys.value = [];
  sharePreviewTitle.value = 'Share Links';
  sharePreview.value = '';
  sharePreviewFilename.value = 'inbounds-export.txt';
  detailOpen.value = true;
}

function openCopyClientsModal(inbound: Inbound | null) {
  if (!inbound) {
    return;
  }
  if (copyClientSourceOptions.value.length === 0) {
    error.value = 'No other inbounds are available as copy sources';
    return;
  }
  copySourceInboundId.value = copyClientSourceOptions.value[0]?.value;
  copySelectedClientKeys.value = [];
  copyFlowOverride.value = '';
  copyClientsModalOpen.value = true;
}

function openBulkAddClientsModal(inbound: Inbound | null) {
  if (!inbound) {
    return;
  }
  if (inbound.protocol === 'wireguard') {
    error.value = 'WireGuard bulk add is not supported in the new UI yet';
    return;
  }
  resetBulkClientForm();
  bulkClientModalOpen.value = true;
}

function confirmCloneInbound(inbound: Inbound) {
  AModal.confirm({
    title: `Clone ${inbound.remark || inbound.tag || inbound.id}?`,
    content:
      'The clone keeps protocol, stream, sniffing, and limits, but starts with a fresh port and empty client list.',
    okText: 'Clone',
    onOk: () => runCloneInbound(inbound),
  });
}

async function runCloneInbound(inbound: Inbound) {
  error.value = '';
  try {
    await addInbound(buildClonedInboundForm(inbound), { notifyOnError: false });
    void message.success('Inbound cloned');
    await refreshInbounds();
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to clone inbound';
  }
}

function openCreateClient(inbound: Inbound) {
  if (!isEditableProtocol(inbound.protocol) || !protocolSupportsClients(inbound.protocol)) {
    return;
  }
  if (clientAddDisabled(inbound)) {
    error.value = 'This Shadowsocks 2022 method does not support clients';
    return;
  }
  clientModalMode.value = 'create';
  clientInbound.value = inbound;
  Object.assign(clientEditor, createClientEditor(inbound.protocol, inbound));
  clientModalOpen.value = true;
}

function openEditClient(inbound: Inbound | null, record: ClientRow | Record<string, unknown>) {
  if (!inbound) {
    return;
  }
  const row = asClientRow(record);
  if (!isEditableProtocol(inbound.protocol) || !protocolSupportsClients(inbound.protocol)) {
    return;
  }
  const client = stripClientRow(row);
  clientModalMode.value = 'edit';
  clientInbound.value = inbound;
  const shadowsocksMethod = getShadowsocksMethod(inbound);
  Object.assign(clientEditor, {
    protocol: inbound.protocol,
    originalClientId: getClientPrimaryId(inbound.protocol, client),
    id: client.id || (usesUuidClientId(inbound.protocol) ? randomUuid() : ''),
    password:
      client.password ||
      (usesPasswordClientId(inbound.protocol)
        ? generateClientCredential(inbound.protocol, inbound)
        : ''),
    auth:
      client.auth ||
      (usesAuthClientId(inbound.protocol)
        ? generateClientCredential(inbound.protocol, inbound)
        : ''),
    privateKey: client.privateKey || '',
    publicKey: client.publicKey || '',
    preSharedKey: client.preSharedKey || '',
    allowedIPs: (client.allowedIPs || []).join('\n'),
    keepAlive: Number(client.keepAlive || 0),
    method:
      inbound.protocol === 'shadowsocks' ? client.method || shadowsocksMethod : client.method || '',
    email: client.email || '',
    security: client.security || 'auto',
    flow: client.flow || '',
    limitIp: Number(client.limitIp || 0),
    totalGB: bytesToGb(Number(row.traffic?.total ?? client.totalGB ?? 0)),
    expiryTime: Number(row.traffic?.expiryTime ?? client.expiryTime ?? 0),
    enable: client.enable !== false,
    subId: client.subId || randomToken(16),
    comment: client.comment || '',
    reset: Number(client.reset || 0),
  });
  clientModalOpen.value = true;
}

async function submitClient() {
  const inbound = clientInbound.value;
  if (
    !inbound ||
    !isEditableProtocol(inbound.protocol) ||
    !protocolSupportsClients(inbound.protocol)
  ) {
    return;
  }

  const client = buildClientPayload();
  const validationError = validateClientPayload(inbound, client);
  if (validationError) {
    error.value = validationError;
    return;
  }

  if (inbound.protocol === 'wireguard') {
    await saveWireguardPeer(inbound, client);
    return;
  }

  savingClient.value = true;
  error.value = '';
  try {
    const payload = {
      id: inbound.id,
      settings: stringifyJson({ clients: [client] }),
    };
    if (clientModalMode.value === 'edit') {
      await updateInboundClient(clientEditor.originalClientId, payload, { notifyOnError: false });
      void message.success('Client updated');
    } else {
      await addInboundClient(payload, { notifyOnError: false });
      void message.success('Client added');
    }
    clientModalOpen.value = false;
    await refreshInbounds();
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to save client';
  } finally {
    savingClient.value = false;
  }
}

async function toggleClient(
  inbound: Inbound | null,
  record: ClientRow | Record<string, unknown>,
  enabled: boolean,
) {
  if (!inbound) {
    return;
  }
  const row = asClientRow(record);
  const clientId = getClientPrimaryId(inbound.protocol, row);
  if (!clientId) {
    return;
  }
  if (inbound.protocol === 'wireguard') {
    await saveWireguardPeer(inbound, {
      ...stripClientRow(row),
      enable: enabled,
    });
    return;
  }
  busyClientKey.value = row.key;
  error.value = '';
  try {
    const client = {
      ...stripClientRow(row),
      enable: enabled,
    };
    await updateInboundClient(
      clientId,
      {
        id: inbound.id,
        settings: stringifyJson({ clients: [client] }),
      },
      { notifyOnError: false },
    );
    void message.success(enabled ? 'Client enabled' : 'Client disabled');
    await refreshInbounds();
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to update client';
  } finally {
    busyClientKey.value = '';
  }
}

function confirmDeleteClient(inbound: Inbound | null, record: ClientRow | Record<string, unknown>) {
  if (!inbound) {
    return;
  }
  const row = asClientRow(record);
  const clientId = getClientPrimaryId(inbound.protocol, row);
  if (!clientId) {
    return;
  }
  AModal.confirm({
    title: `Delete ${row.email || clientId}?`,
    content: 'This uses the existing Xray client delete endpoint.',
    okButtonProps: { danger: true },
    okText: 'Delete',
    onOk: () => runDeleteClient(inbound, row),
  });
}

async function runDeleteClient(inbound: Inbound, row: ClientRow) {
  const clientId = getClientPrimaryId(inbound.protocol, row);
  if (!clientId) {
    return;
  }
  if (inbound.protocol === 'wireguard') {
    await deleteWireguardPeer(inbound, clientId);
    return;
  }
  busyClient.value = true;
  error.value = '';
  try {
    await deleteInboundClient(inbound.id, clientId, { notifyOnError: false });
    void message.success('Client deleted');
    await refreshInbounds();
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to delete client';
  } finally {
    busyClient.value = false;
  }
}

async function saveWireguardPeer(inbound: Inbound, client: InboundClient) {
  const settings = parseInboundSettings(inbound);
  const peers = Array.isArray(settings.peers) ? [...settings.peers] : [];
  const primaryId = clientEditor.originalClientId || getClientPrimaryId(inbound.protocol, client);
  const nextClient = {
    email: client.email,
    privateKey: client.privateKey,
    publicKey: client.publicKey,
    preSharedKey: client.preSharedKey || undefined,
    allowedIPs: client.allowedIPs || [],
    keepAlive: client.keepAlive || undefined,
    enable: client.enable !== false,
    subId: client.subId || randomToken(16),
  };
  const index = peers.findIndex((peer) => getClientPrimaryId('wireguard', peer) === primaryId);
  if (clientModalMode.value === 'edit' && index >= 0) {
    peers[index] = nextClient;
  } else {
    peers.push(nextClient);
  }
  settings.peers = peers;
  await saveInboundSettings(inbound, settings, 'WireGuard peer saved');
  clientModalOpen.value = false;
}

async function deleteWireguardPeer(inbound: Inbound, publicKey: string) {
  const settings = parseInboundSettings(inbound);
  const peers = Array.isArray(settings.peers) ? settings.peers : [];
  const nextPeers = peers.filter((peer) => getClientPrimaryId('wireguard', peer) !== publicKey);
  if (nextPeers.length === peers.length) {
    return;
  }
  if (nextPeers.length === 0) {
    error.value = 'WireGuard requires at least one peer';
    return;
  }
  settings.peers = nextPeers;
  await saveInboundSettings(inbound, settings, 'WireGuard peer deleted');
}

function confirmDeleteSelectedClients(inbound: Inbound | null) {
  if (!inbound || selectedBatchClientRows.value.length === 0) {
    return;
  }
  if (selectedBatchClientRows.value.length >= selectedClientRows.value.length) {
    error.value = 'At least one client must remain';
    return;
  }
  AModal.confirm({
    title: `Delete ${selectedBatchClientRows.value.length} selected clients?`,
    content: 'This uses the existing Xray inbound update/client delete endpoints.',
    okButtonProps: { danger: true },
    okText: 'Delete Selected',
    onOk: () => runDeleteSelectedClients(inbound),
  });
}

async function runDeleteSelectedClients(inbound: Inbound) {
  const rows = [...selectedBatchClientRows.value];
  if (inbound.protocol === 'wireguard') {
    const settings = parseInboundSettings(inbound);
    const selectedIds = new Set(rows.map((row) => getClientPrimaryId('wireguard', row)));
    const peers = Array.isArray(settings.peers) ? settings.peers : [];
    const nextPeers = peers.filter(
      (peer) => !selectedIds.has(getClientPrimaryId('wireguard', peer)),
    );
    if (nextPeers.length === 0) {
      error.value = 'WireGuard requires at least one peer';
      return;
    }
    settings.peers = nextPeers;
    selectedClientRowKeys.value = [];
    await saveInboundSettings(inbound, settings, 'Selected WireGuard peers deleted');
    return;
  }

  busyClient.value = true;
  error.value = '';
  try {
    for (const row of rows) {
      const clientId = getClientPrimaryId(inbound.protocol, row);
      if (clientId) {
        await deleteInboundClient(inbound.id, clientId, { notifyOnError: false });
      }
    }
    selectedClientRowKeys.value = [];
    void message.success('Selected clients deleted');
    await refreshInbounds();
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to delete selected clients';
  } finally {
    busyClient.value = false;
  }
}

async function saveInboundSettings(inbound: Inbound, settings: object, successText: string) {
  busyClient.value = true;
  error.value = '';
  try {
    await updateInbound(
      inbound.id,
      {
        ...inbound,
        settings: stringifyJson(settings),
      },
      { notifyOnError: false },
    );
    void message.success(successText);
    await refreshInbounds();
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to save inbound settings';
  } finally {
    busyClient.value = false;
  }
}

function confirmResetClient(inbound: Inbound | null, record: ClientRow | Record<string, unknown>) {
  if (!inbound) {
    return;
  }
  const row = asClientRow(record);
  if (!row.email) {
    return;
  }
  AModal.confirm({
    title: `Reset traffic for ${row.email}?`,
    okText: 'Reset',
    onOk: () => runResetClient(inbound, row),
  });
}

async function runResetClient(inbound: Inbound, row: ClientRow) {
  if (!row.email) {
    return;
  }
  busyClient.value = true;
  error.value = '';
  try {
    await resetInboundClientTraffic(inbound.id, row.email, { notifyOnError: false });
    void message.success('Client traffic reset');
    await refreshInbounds();
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to reset client traffic';
  } finally {
    busyClient.value = false;
  }
}

function confirmResetSelectedClients(inbound: Inbound | null) {
  if (!inbound || resettableSelectedClientRows.value.length === 0) {
    return;
  }
  AModal.confirm({
    title: `Reset traffic for ${resettableSelectedClientRows.value.length} selected clients?`,
    okText: 'Reset Selected',
    onOk: () => runResetSelectedClients(inbound),
  });
}

async function runResetSelectedClients(inbound: Inbound) {
  const rows = [...resettableSelectedClientRows.value];
  busyClient.value = true;
  error.value = '';
  try {
    for (const row of rows) {
      if (row.email) {
        await resetInboundClientTraffic(inbound.id, row.email, { notifyOnError: false });
      }
    }
    selectedClientRowKeys.value = [];
    void message.success('Selected client traffic reset');
    await refreshInbounds();
  } catch (caught) {
    error.value =
      caught instanceof Error ? caught.message : 'Failed to reset selected client traffic';
  } finally {
    busyClient.value = false;
  }
}

function confirmResetAllClients(inbound: Inbound) {
  AModal.confirm({
    title: `Reset all client traffic for ${inbound.remark || inbound.tag || inbound.id}?`,
    okText: 'Reset All',
    onOk: () => runResetAllClients(inbound),
  });
}

async function runResetAllClients(inbound: Inbound) {
  busyClient.value = true;
  error.value = '';
  try {
    await resetAllInboundClientTraffics(inbound.id, { notifyOnError: false });
    void message.success('All client traffic reset');
    await refreshInbounds();
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to reset clients';
  } finally {
    busyClient.value = false;
  }
}

function confirmDeleteDepletedClients(inbound: Inbound) {
  AModal.confirm({
    title: `Delete depleted clients for ${inbound.remark || inbound.tag || inbound.id}?`,
    content: 'Clients that exhausted their traffic limit will be removed.',
    okButtonProps: { danger: true },
    okText: 'Delete Depleted',
    onOk: () => runDeleteDepletedClients(inbound),
  });
}

async function runDeleteDepletedClients(inbound: Inbound) {
  deletingDepletedClients.value = true;
  error.value = '';
  try {
    await deleteDepletedInboundClients(inbound.id, { notifyOnError: false });
    void message.success('Depleted clients deleted');
    await refreshInbounds();
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to delete depleted clients';
  } finally {
    deletingDepletedClients.value = false;
  }
}

async function previewShareLink(
  inbound: Inbound | null,
  record: ClientRow | Record<string, unknown>,
) {
  if (!inbound) {
    return;
  }
  const row = asClientRow(record);
  const link = buildClientShareLink(inbound, row);
  sharePreviewTitle.value = 'Share Link';
  sharePreview.value = link;
  sharePreviewFilename.value = safeExportFilename(
    `${inbound.remark || inbound.tag || 'inbound'}-${row.email || row.key}`,
  );
  if (!link) {
    error.value = 'Share link is only available for supported clients with complete credentials';
    return;
  }
  await copyText(link);
  void message.success('Share link copied');
}

async function openClientAccessModal(record: ClientRow | Record<string, unknown>) {
  const inbound = selectedInbound.value;
  if (!inbound) {
    return;
  }
  const row = asClientRow(record);
  const settings = await ensureSubscriptionSettingsLoaded();
  if (!settings) {
    return;
  }

  const nextLinks: AccessLinkItem[] = [];
  const shareLink = buildClientShareLink(inbound, row);
  if (shareLink) {
    nextLinks.push({ kind: 'share', label: 'Share Link', url: shareLink });
  }
  buildClientSubscriptionLinks(row, settings).forEach((link) => {
    nextLinks.push({ kind: 'subscription', label: link.label, url: link.url });
  });

  clientAccessClient.value = row;
  clientAccessLinks.value = nextLinks;
  clientAccessTitle.value = row.email ? `Client Access - ${row.email}` : 'Client Access';
  clientAccessModalOpen.value = true;
  await renderClientAccessQrs();
}

async function exportInboundShareLinks(inbound: Inbound | null) {
  if (!inbound) {
    return;
  }
  const links = buildInboundShareLinks(inbound);
  await presentSharePreview({
    emptyError: 'No share links are available for this inbound',
    filename: safeExportFilename(inbound.remark || inbound.tag || `inbound-${inbound.id}`),
    messageText: `${links.length} share links copied`,
    text: links.join('\n'),
    title: 'Share Links',
  });
}

async function exportAllInboundShareLinks() {
  const links = inbounds.value.flatMap((inbound) => buildInboundShareLinks(inbound));
  await presentSharePreview({
    emptyError: 'No share links are available for current inbounds',
    filename: 'All-Inbounds.txt',
    messageText: `${links.length} share links copied`,
    text: links.join('\n'),
    title: 'All Share Links',
  });
}

async function exportInboundSubscriptionLinks(inbound: Inbound | null) {
  if (!inbound) {
    return;
  }
  const settings = await ensureSubscriptionSettingsLoaded();
  if (!settings) {
    return;
  }
  const blocks = buildClientRows(inbound)
    .map((row) => {
      const links = buildClientSubscriptionLinks(row, settings);
      if (links.length === 0) {
        return '';
      }
      return formatSubscriptionPreview(row.email || row.subId || row.key, links);
    })
    .filter((value) => value.trim().length > 0);

  if (blocks.length === 0) {
    error.value = 'No subscription links are available for this inbound';
    sharePreview.value = '';
    sharePreviewFilename.value = safeExportFilename(
      `${inbound.remark || inbound.tag || `inbound-${inbound.id}`}-Subs`,
    );
    return;
  }

  const text = blocks.join('\n\n');
  sharePreviewTitle.value = 'Subscription Links';
  sharePreview.value = text;
  sharePreviewFilename.value = safeExportFilename(
    `${inbound.remark || inbound.tag || `inbound-${inbound.id}`}-Subs`,
  );
  await copyText(text);
  void message.success(`${blocks.length} client subscription link groups copied`);
}

async function exportAllInboundSubscriptionLinks() {
  const settings = await ensureSubscriptionSettingsLoaded();
  if (!settings) {
    return;
  }

  const links = inbounds.value.flatMap((inbound) =>
    buildClientRows(inbound).flatMap((row) =>
      buildClientSubscriptionLinks(row, settings).map((link) => link.url),
    ),
  );
  const uniqueLinks = [...new Set(links)].filter((link) => link.trim().length > 0);
  await presentSharePreview({
    emptyError: 'No subscription links are available for current inbounds',
    filename: 'All-Inbounds-Subs.txt',
    messageText: `${uniqueLinks.length} subscription links copied`,
    text: uniqueLinks.join('\n'),
    title: 'All Subscription Links',
  });
}

async function exportInboundJson(inbound: Inbound) {
  const text = JSON.stringify(inbound, null, 2);
  sharePreviewTitle.value = 'Inbound JSON';
  sharePreview.value = text;
  sharePreviewFilename.value = safeExportFilename(
    inbound.remark || inbound.tag || `inbound-${inbound.id}`,
    'json',
  );
  await copyText(text);
  void message.success('Inbound JSON copied');
}

async function openInboundQrcode(inbound: Inbound) {
  const links = buildInboundShareLinks(inbound);
  if (links.length === 0) {
    error.value = 'No share links are available for this inbound';
    return;
  }

  selectedInbound.value = inbound;
  clientAccessClient.value = {
    key: `inbound-${inbound.id}`,
    email: inbound.remark || inbound.tag || `Inbound ${inbound.id}`,
    enable: inbound.enable,
    subId: '',
  };
  clientAccessLinks.value = links.map((url, index) => ({
    kind: 'share',
    label: links.length === 1 ? 'Share Link' : `Share Link ${index + 1}`,
    url,
  }));
  clientAccessTitle.value = `Inbound Access - ${inbound.remark || inbound.tag || inbound.id}`;
  clientAccessModalOpen.value = true;
  await renderClientAccessQrs();
}

async function presentSharePreview(input: {
  emptyError: string;
  filename: string;
  messageText: string;
  text: string;
  title: string;
}) {
  if (!input.text.trim()) {
    error.value = input.emptyError;
    sharePreview.value = '';
    sharePreviewFilename.value = input.filename;
    return;
  }

  sharePreviewTitle.value = input.title;
  sharePreview.value = input.text;
  sharePreviewFilename.value = input.filename;
  await copyText(input.text);
  void message.success(input.messageText);
}

async function submitCopyClients() {
  const inbound = selectedInbound.value;
  if (!inbound || !copySourceInboundId.value) {
    return;
  }
  if (copySelectedSourceRows.value.length === 0) {
    error.value = 'Select at least one client to copy';
    return;
  }

  copyingClients.value = true;
  error.value = '';
  try {
    const result = await copyInboundClients(
      inbound.id,
      {
        sourceInboundId: copySourceInboundId.value,
        clientEmails: copySelectedSourceRows.value.map((row) => row.email),
        flow: copyFlowOverride.value || undefined,
      },
      { notifyOnError: false },
    );
    const addedCount = Array.isArray(result.added)
      ? result.added.length
      : copySelectedSourceRows.value.length;
    if (Array.isArray(result.errors) && result.errors.length > 0) {
      error.value = result.errors.join('; ');
    }
    void message.success(`${addedCount} clients copied`);
    copyClientsModalOpen.value = false;
    await refreshInbounds();
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to copy clients';
  } finally {
    copyingClients.value = false;
  }
}

async function submitBulkAddClients() {
  const inbound = selectedInbound.value;
  if (!inbound) {
    return;
  }
  if (!bulkClientForm.emailPrefix.trim() && !bulkClientForm.emailPostfix.trim()) {
    error.value = 'Bulk add requires an email prefix or postfix';
    return;
  }

  const protocol = isEditableProtocol(inbound.protocol) ? inbound.protocol : null;
  if (!protocol) {
    error.value = 'Bulk add is only available for editable protocols';
    return;
  }

  const clients = generateBulkClientProfiles({
    protocol,
    quantity: bulkClientForm.quantity,
    firstIndex: bulkClientForm.firstIndex,
    emailPrefix: bulkClientForm.emailPrefix.trim(),
    emailPostfix: bulkClientForm.emailPostfix.trim(),
    flow: bulkClientForm.flow,
    security: 'auto',
    shadowsocksMethod: inbound.protocol === 'shadowsocks' ? getShadowsocksMethod(inbound) : '',
    limitIp: bulkClientForm.limitIp,
    totalGB: bulkClientForm.totalGB,
    expiryTime: bulkClientForm.expiryTime,
    reset: bulkClientForm.reset,
  });

  bulkAddingClients.value = true;
  error.value = '';
  try {
    await addInboundClient(
      {
        id: inbound.id,
        settings: stringifyJson({ clients }),
      },
      { notifyOnError: false },
    );
    void message.success(`${clients.length} clients added`);
    bulkClientModalOpen.value = false;
    await refreshInbounds();
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to bulk add clients';
  } finally {
    bulkAddingClients.value = false;
  }
}

async function copySharePreview() {
  if (!sharePreview.value) {
    return;
  }
  await copyText(sharePreview.value);
  void message.success('Copied');
}

function downloadSharePreview() {
  if (!sharePreview.value) {
    return;
  }
  downloadText(sharePreviewFilename.value, sharePreview.value);
  void message.success('Downloaded');
}

async function copyClientAccessLink(uri: string) {
  await copyText(uri);
  void message.success('Link copied');
}

function openPublicAccessLink(uri: string) {
  if (!isOpenablePublicLink(uri)) {
    return;
  }
  window.open(uri, '_blank', 'noopener,noreferrer');
}

/** 复制当前 Gateway 代理 URI，便于粘贴到 Super-Code-Gateway 代理配置。 */
async function copyGatewayProxyUri(uri: string) {
  await copyText(uri);
  void message.success('Gateway proxy URI copied');
}

async function openClientIps(record: ClientRow | Record<string, unknown>) {
  const row = asClientRow(record);
  if (!row.email) {
    return;
  }
  clientIpsModalTitle.value = `IP Records - ${row.email}`;
  clientIpsText.value = 'Loading...';
  clientIpsModalOpen.value = true;
  error.value = '';
  try {
    clientIpsText.value = formatClientIpRecords(
      await getClientIps(row.email, { notifyOnError: false }),
    );
  } catch (caught) {
    const messageText = caught instanceof Error ? caught.message : 'Failed to load client IPs';
    clientIpsText.value = messageText;
    error.value = messageText;
  }
}

function confirmClearClientIps(record: ClientRow | Record<string, unknown>) {
  const row = asClientRow(record);
  if (!row.email) {
    return;
  }
  AModal.confirm({
    title: `Clear IP records for ${row.email}?`,
    okButtonProps: { danger: true },
    okText: 'Clear IPs',
    onOk: () => runClearClientIps(row.email),
  });
}

async function runClearClientIps(email: string) {
  clearingClientIpsEmail.value = email;
  error.value = '';
  try {
    await clearClientIps(email, { notifyOnError: false });
    if (clientIpsModalOpen.value && clientIpsModalTitle.value.includes(email)) {
      clientIpsText.value = 'No IP Record';
    }
    void message.success('Client IP records cleared');
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to clear client IPs';
  } finally {
    clearingClientIpsEmail.value = '';
  }
}

function createWireguardEditor(): WireguardEditorState {
  return {
    mtu: 1420,
    secretKey: '',
    pubKey: '',
    noKernelTun: false,
  };
}

function createStreamEditor(): StreamEditorState {
  return {
    network: 'tcp',
    security: 'none',
    tcpAcceptProxyProtocol: false,
    tcpHeaderType: 'none',
    kcpMtu: 1350,
    kcpTti: 20,
    kcpUplinkCapacity: 5,
    kcpDownlinkCapacity: 20,
    kcpCwndMultiplier: 1,
    kcpMaxSendingWindow: 2_097_152,
    wsAcceptProxyProtocol: false,
    wsPath: '/',
    wsHost: '',
    wsHeartbeatPeriod: 0,
    grpcServiceName: '',
    grpcAuthority: '',
    grpcMultiMode: false,
    httpupgradeAcceptProxyProtocol: false,
    httpupgradePath: '/',
    httpupgradeHost: '',
    xhttpPath: '/',
    xhttpHost: '',
    xhttpMode: 'auto',
    xhttpNoSseHeader: false,
    xhttpScMaxBufferedPosts: 30,
    xhttpScMaxEachPostBytes: '1000000',
    xhttpScStreamUpServerSecs: '20-80',
    xhttpXPaddingBytes: '100-1000',
    tlsServerName: '',
    tlsMinVersion: '1.2',
    tlsMaxVersion: '1.3',
    tlsAlpn: 'h2,http/1.1',
    tlsFingerprint: 'chrome',
    tlsCertificateFile: '',
    tlsKeyFile: '',
    tlsRejectUnknownSni: false,
    tlsDisableSystemRoot: false,
    tlsEnableSessionResumption: false,
    tlsEchServerKeys: '',
    tlsEchConfigList: '',
    realityShow: false,
    realityXver: 0,
    realityTarget: '',
    realityServerNames: '',
    realityPrivateKey: '',
    realityShortIds: '',
    realityMinClientVer: '',
    realityMaxClientVer: '',
    realityMaxTimediff: 0,
    realityPublicKey: '',
    realitySpiderX: '/',
    realityMldsa65Seed: '',
    realityMldsa65Verify: '',
    hysteriaAuth: '',
    hysteriaUdpIdleTimeout: 60,
    sockoptEnabled: false,
    sockoptAcceptProxyProtocol: false,
    sockoptTcpFastOpen: false,
    sockoptTcpMptcp: false,
    sockoptPenetrate: false,
    sockoptV6Only: false,
    sockoptDomainStrategy: 'UseIP',
    sockoptTcpCongestion: 'bbr',
    sockoptTproxy: 'off',
    sockoptMark: 0,
    sockoptTcpMaxSeg: 1440,
    sockoptDialerProxy: '',
    sockoptInterfaceName: '',
    sockoptTrustedXForwardedFor: '',
  };
}

function syncWireguardEditorFromSettings() {
  const settings = parseInboundSettingsText(inboundEditor.settings);
  Object.assign(wireguardEditor, {
    mtu: Number(settings.mtu || 1420),
    secretKey: stringField(settings.secretKey),
    pubKey: stringField(settings.pubKey),
    noKernelTun: Boolean(settings.noKernelTun),
  });
}

function applyWireguardEditorToSettings() {
  const settings = parseInboundSettingsText(inboundEditor.settings);
  settings.mtu = Math.max(0, Number(wireguardEditor.mtu || 0));
  settings.secretKey = wireguardEditor.secretKey.trim();
  settings.pubKey = wireguardEditor.pubKey.trim();
  settings.noKernelTun = wireguardEditor.noKernelTun;
  if (!Array.isArray(settings.peers) || settings.peers.length === 0) {
    settings.peers = [generateWireguardPeer(0)];
  }
  inboundEditor.settings = stringifyJson(settings);
}

function syncInboundClientEditorFromSettings() {
  const settings = parseInboundSettingsText(inboundEditor.settings);
  const clients = Array.isArray(settings.clients) ? settings.clients : [];
  const client = objectField(clients[0]);
  const fallback = createClientEditor(inboundEditor.protocol);
  Object.assign(inboundClientEditor, {
    ...fallback,
    protocol: inboundEditor.protocol,
    id: stringField(client.id) || fallback.id,
    password: stringField(client.password) || fallback.password,
    auth: stringField(client.auth) || fallback.auth,
    method: stringField(client.method) || fallback.method,
    email: stringField(client.email),
    security: stringField(client.security) || fallback.security,
    flow: stringField(client.flow),
    limitIp: Number(client.limitIp || 0),
    totalGB: bytesToGb(Number(client.totalGB || 0)),
    expiryTime: Number(client.expiryTime || 0),
    enable: client.enable !== false,
    subId: stringField(client.subId) || fallback.subId,
    comment: stringField(client.comment),
    reset: Number(client.reset || 0),
  });
}

function applyInboundClientEditorToSettings() {
  const settings = parseInboundSettingsText(inboundEditor.settings);
  const clients = Array.isArray(settings.clients) ? settings.clients : [];
  const existingClient = objectField(clients[0]);
  const client = buildClientPayloadFromEditor(inboundClientEditor);
  settings.clients = [{ ...existingClient, ...client }, ...clients.slice(1)];
  inboundEditor.settings = stringifyJson(settings);
}

function syncStreamEditorFromSettings() {
  const stream = parseInboundStreamSettingsText(inboundEditor.streamSettings);
  const tlsSettings = objectField(stream.tlsSettings);
  const tlsClientSettings = objectField(tlsSettings.settings);
  const firstCertificate = Array.isArray(tlsSettings.certificates)
    ? objectField(tlsSettings.certificates[0])
    : {};
  const realitySettings = objectField(stream.realitySettings);
  const realityClientSettings = objectField(realitySettings.settings);
  const wsSettings = objectField(stream.wsSettings);
  const grpcSettings = objectField(stream.grpcSettings);
  const httpupgradeSettings = objectField(stream.httpupgradeSettings);
  const xhttpSettings = objectField(stream.xhttpSettings);
  const kcpSettings = objectField(stream.kcpSettings);
  const sockopt = objectField(stream.sockopt);
  const tcpSettings = objectField(stream.tcpSettings);
  const tcpHeader = objectField(tcpSettings.header);
  const hysteriaSettings = objectField(stream.hysteriaSettings);

  Object.assign(streamEditor, {
    network: stringField(stream.network) || defaultNetworkForProtocol(inboundEditor.protocol),
    security: stringField(stream.security) || defaultSecurityForProtocol(inboundEditor.protocol),
    tcpAcceptProxyProtocol: Boolean(tcpSettings.acceptProxyProtocol),
    tcpHeaderType: stringField(tcpHeader.type) || 'none',
    kcpMtu: Number(kcpSettings.mtu || 1350),
    kcpTti: Number(kcpSettings.tti || 20),
    kcpUplinkCapacity: Number(kcpSettings.uplinkCapacity || 5),
    kcpDownlinkCapacity: Number(kcpSettings.downlinkCapacity || 20),
    kcpCwndMultiplier: Number(kcpSettings.cwndMultiplier || 1),
    kcpMaxSendingWindow: Number(kcpSettings.maxSendingWindow || 2_097_152),
    wsAcceptProxyProtocol: Boolean(wsSettings.acceptProxyProtocol),
    wsPath: stringField(wsSettings.path) || '/',
    wsHost: stringField(wsSettings.host),
    wsHeartbeatPeriod: Number(wsSettings.heartbeatPeriod || 0),
    grpcServiceName: stringField(grpcSettings.serviceName),
    grpcAuthority: stringField(grpcSettings.authority),
    grpcMultiMode: Boolean(grpcSettings.multiMode),
    httpupgradeAcceptProxyProtocol: Boolean(httpupgradeSettings.acceptProxyProtocol),
    httpupgradePath: stringField(httpupgradeSettings.path) || '/',
    httpupgradeHost: stringField(httpupgradeSettings.host),
    xhttpPath: stringField(xhttpSettings.path) || '/',
    xhttpHost: stringField(xhttpSettings.host),
    xhttpMode: stringField(xhttpSettings.mode) || 'auto',
    xhttpNoSseHeader: Boolean(xhttpSettings.noSSEHeader),
    xhttpScMaxBufferedPosts: Number(xhttpSettings.scMaxBufferedPosts || 30),
    xhttpScMaxEachPostBytes: stringField(xhttpSettings.scMaxEachPostBytes) || '1000000',
    xhttpScStreamUpServerSecs: stringField(xhttpSettings.scStreamUpServerSecs) || '20-80',
    xhttpXPaddingBytes: stringField(xhttpSettings.xPaddingBytes) || '100-1000',
    tlsServerName: stringField(tlsSettings.serverName),
    tlsMinVersion: stringField(tlsSettings.minVersion) || '1.2',
    tlsMaxVersion: stringField(tlsSettings.maxVersion) || '1.3',
    tlsAlpn:
      arrayField(tlsSettings.alpn).join(',') || defaultTlsAlpnForProtocol(inboundEditor.protocol),
    tlsFingerprint: stringField(tlsClientSettings.fingerprint) || 'chrome',
    tlsCertificateFile: stringField(firstCertificate.certificateFile),
    tlsKeyFile: stringField(firstCertificate.keyFile),
    tlsRejectUnknownSni: Boolean(tlsSettings.rejectUnknownSni),
    tlsDisableSystemRoot: Boolean(tlsSettings.disableSystemRoot),
    tlsEnableSessionResumption: Boolean(tlsSettings.enableSessionResumption),
    tlsEchServerKeys: stringField(tlsSettings.echServerKeys),
    tlsEchConfigList: stringField(tlsClientSettings.echConfigList),
    realityShow: Boolean(realitySettings.show),
    realityXver: Number(realitySettings.xver || 0),
    realityTarget: stringField(realitySettings.target),
    realityServerNames: arrayField(realitySettings.serverNames).join(','),
    realityPrivateKey: stringField(realitySettings.privateKey),
    realityShortIds: arrayField(realitySettings.shortIds).join(','),
    realityMinClientVer: stringField(realitySettings.minClientVer),
    realityMaxClientVer: stringField(realitySettings.maxClientVer),
    realityMaxTimediff: Number(realitySettings.maxTimediff || 0),
    realityPublicKey: stringField(realityClientSettings.publicKey),
    realitySpiderX: stringField(realityClientSettings.spiderX) || '/',
    realityMldsa65Seed: stringField(realitySettings.mldsa65Seed),
    realityMldsa65Verify: stringField(realityClientSettings.mldsa65Verify),
    hysteriaAuth: stringField(hysteriaSettings.auth),
    hysteriaUdpIdleTimeout: Number(hysteriaSettings.udpIdleTimeout || 60),
    sockoptEnabled: Object.keys(sockopt).length > 0,
    sockoptAcceptProxyProtocol: Boolean(sockopt.acceptProxyProtocol),
    sockoptTcpFastOpen: Boolean(sockopt.tcpFastOpen),
    sockoptTcpMptcp: Boolean(sockopt.tcpMptcp),
    sockoptPenetrate: Boolean(sockopt.penetrate),
    sockoptV6Only: Boolean(sockopt.V6Only),
    sockoptDomainStrategy: stringField(sockopt.domainStrategy) || 'UseIP',
    sockoptTcpCongestion: stringField(sockopt.tcpcongestion) || 'bbr',
    sockoptTproxy: stringField(sockopt.tproxy) || 'off',
    sockoptMark: Number(sockopt.mark || 0),
    sockoptTcpMaxSeg: Number(sockopt.tcpMaxSeg || 1440),
    sockoptDialerProxy: stringField(sockopt.dialerProxy),
    sockoptInterfaceName: stringField(sockopt.interface),
    sockoptTrustedXForwardedFor: arrayField(sockopt.trustedXForwardedFor).join(','),
  });

  if (isHysteriaProtocol(inboundEditor.protocol)) {
    streamEditor.network = 'hysteria';
    streamEditor.security = 'tls';
    streamEditor.tlsAlpn = streamEditor.tlsAlpn || 'h3';
  }
}

function applyStreamEditorToSettings() {
  const stream = parseInboundStreamSettingsText(inboundEditor.streamSettings);
  const network = isHysteriaProtocol(inboundEditor.protocol) ? 'hysteria' : streamEditor.network;
  const security = isHysteriaProtocol(inboundEditor.protocol) ? 'tls' : streamEditor.security;
  const existingTlsSettings = objectField(stream.tlsSettings);

  stream.network = network;
  stream.security = security;
  stream.externalProxy = Array.isArray(stream.externalProxy) ? stream.externalProxy : [];

  delete stream.tcpSettings;
  delete stream.kcpSettings;
  delete stream.wsSettings;
  delete stream.grpcSettings;
  delete stream.httpupgradeSettings;
  delete stream.xhttpSettings;
  delete stream.tlsSettings;
  delete stream.realitySettings;
  delete stream.hysteriaSettings;
  delete stream.sockopt;

  if (network === 'tcp') {
    stream.tcpSettings = {
      acceptProxyProtocol: streamEditor.tcpAcceptProxyProtocol,
      header: {
        type: streamEditor.tcpHeaderType || 'none',
      },
    };
  }
  if (network === 'kcp') {
    stream.kcpSettings = {
      mtu: Math.max(1, Number(streamEditor.kcpMtu || 1350)),
      tti: Math.max(1, Number(streamEditor.kcpTti || 20)),
      uplinkCapacity: Math.max(0, Number(streamEditor.kcpUplinkCapacity || 0)),
      downlinkCapacity: Math.max(0, Number(streamEditor.kcpDownlinkCapacity || 0)),
      cwndMultiplier: Math.max(0, Number(streamEditor.kcpCwndMultiplier || 0)),
      maxSendingWindow: Math.max(0, Number(streamEditor.kcpMaxSendingWindow || 0)),
    };
  }
  if (network === 'ws') {
    stream.wsSettings = {
      acceptProxyProtocol: streamEditor.wsAcceptProxyProtocol,
      path: streamEditor.wsPath || '/',
      host: streamEditor.wsHost,
      headers: streamEditor.wsHost ? { Host: streamEditor.wsHost } : {},
      heartbeatPeriod: Math.max(0, Number(streamEditor.wsHeartbeatPeriod || 0)),
    };
  }
  if (network === 'grpc') {
    stream.grpcSettings = {
      serviceName: streamEditor.grpcServiceName,
      authority: streamEditor.grpcAuthority,
      multiMode: streamEditor.grpcMultiMode,
    };
  }
  if (network === 'httpupgrade') {
    stream.httpupgradeSettings = {
      acceptProxyProtocol: streamEditor.httpupgradeAcceptProxyProtocol,
      path: streamEditor.httpupgradePath || '/',
      host: streamEditor.httpupgradeHost,
      headers: streamEditor.httpupgradeHost ? { Host: streamEditor.httpupgradeHost } : {},
    };
  }
  if (network === 'xhttp') {
    stream.xhttpSettings = {
      path: streamEditor.xhttpPath || '/',
      host: streamEditor.xhttpHost,
      headers: streamEditor.xhttpHost ? { Host: streamEditor.xhttpHost } : {},
      scMaxBufferedPosts: Math.max(0, Number(streamEditor.xhttpScMaxBufferedPosts || 0)),
      scMaxEachPostBytes: streamEditor.xhttpScMaxEachPostBytes || '1000000',
      scStreamUpServerSecs: streamEditor.xhttpScStreamUpServerSecs || '20-80',
      noSSEHeader: streamEditor.xhttpNoSseHeader,
      xPaddingBytes: streamEditor.xhttpXPaddingBytes || '100-1000',
      mode: streamEditor.xhttpMode || 'auto',
    };
  }
  if (network === 'hysteria') {
    stream.hysteriaSettings = {
      protocol: 'hysteria',
      version: 2,
      auth: streamEditor.hysteriaAuth,
      udpIdleTimeout: Math.max(0, Number(streamEditor.hysteriaUdpIdleTimeout || 0)),
    };
  }

  if (security === 'tls') {
    stream.tlsSettings = buildTlsSettings(existingTlsSettings);
  }
  if (security === 'reality') {
    const reality = normalizeRealityServerSettings({
      target: streamEditor.realityTarget,
      serverNames: streamEditor.realityServerNames,
      privateKey: streamEditor.realityPrivateKey,
      shortIds: streamEditor.realityShortIds,
      publicKey: streamEditor.realityPublicKey,
      spiderX: streamEditor.realitySpiderX,
      mldsa65Seed: streamEditor.realityMldsa65Seed,
      mldsa65Verify: streamEditor.realityMldsa65Verify,
    });
    streamEditor.realityTarget = reality.target;
    streamEditor.realityServerNames = reality.serverNames.join(',');
    streamEditor.realityPrivateKey = reality.privateKey;
    streamEditor.realityShortIds = reality.shortIds.join(',');
    streamEditor.realityPublicKey = reality.publicKey;
    streamEditor.realitySpiderX = reality.spiderX;
    streamEditor.realityMldsa65Seed = reality.mldsa65Seed;
    streamEditor.realityMldsa65Verify = reality.mldsa65Verify;
    stream.realitySettings = {
      show: streamEditor.realityShow,
      xver: Math.max(0, Number(streamEditor.realityXver || 0)),
      target: reality.target,
      serverNames: reality.serverNames,
      privateKey: reality.privateKey,
      minClientVer: streamEditor.realityMinClientVer,
      maxClientVer: streamEditor.realityMaxClientVer,
      maxTimediff: Math.max(0, Number(streamEditor.realityMaxTimediff || 0)),
      shortIds: reality.shortIds,
      mldsa65Seed: reality.mldsa65Seed,
      settings: {
        publicKey: reality.publicKey,
        fingerprint: streamEditor.tlsFingerprint || 'chrome',
        serverName: reality.serverNames[0] || '',
        spiderX: reality.spiderX,
        mldsa65Verify: reality.mldsa65Verify,
      },
    };
  }

  if (streamEditor.sockoptEnabled) {
    stream.sockopt = buildSockoptSettings();
  }

  inboundEditor.streamSettings = stringifyJson(stream);
}

function buildSockoptSettings(): Record<string, unknown> {
  return {
    acceptProxyProtocol: streamEditor.sockoptAcceptProxyProtocol,
    tcpFastOpen: streamEditor.sockoptTcpFastOpen,
    mark: Math.max(0, Number(streamEditor.sockoptMark || 0)),
    tproxy: streamEditor.sockoptTproxy || 'off',
    tcpMptcp: streamEditor.sockoptTcpMptcp,
    penetrate: streamEditor.sockoptPenetrate,
    domainStrategy: streamEditor.sockoptDomainStrategy || 'UseIP',
    tcpMaxSeg: Math.max(0, Number(streamEditor.sockoptTcpMaxSeg || 0)),
    dialerProxy: streamEditor.sockoptDialerProxy,
    tcpcongestion: streamEditor.sockoptTcpCongestion || 'bbr',
    V6Only: streamEditor.sockoptV6Only,
    interface: streamEditor.sockoptInterfaceName,
    trustedXForwardedFor: parseListText(streamEditor.sockoptTrustedXForwardedFor),
  };
}

function buildTlsSettings(existingTlsSettings: Record<string, unknown>): Record<string, unknown> {
  const certificates = Array.isArray(existingTlsSettings.certificates)
    ? existingTlsSettings.certificates
    : [];
  const certificateFile = streamEditor.tlsCertificateFile.trim();
  const keyFile = streamEditor.tlsKeyFile.trim();
  const nextCertificates =
    certificateFile || keyFile
      ? [
          {
            certificateFile,
            keyFile,
            oneTimeLoading: false,
            usage: 'encipherment',
            buildChain: false,
          },
        ]
      : certificates;

  return {
    serverName: streamEditor.tlsServerName,
    minVersion: streamEditor.tlsMinVersion || '1.2',
    maxVersion: streamEditor.tlsMaxVersion || '1.3',
    cipherSuites: '',
    rejectUnknownSni: streamEditor.tlsRejectUnknownSni,
    disableSystemRoot: streamEditor.tlsDisableSystemRoot,
    enableSessionResumption: streamEditor.tlsEnableSessionResumption,
    certificates: nextCertificates,
    alpn: parseListText(streamEditor.tlsAlpn),
    echServerKeys: streamEditor.tlsEchServerKeys,
    echForceQuery: stringField(existingTlsSettings.echForceQuery) || 'none',
    settings: {
      fingerprint: streamEditor.tlsFingerprint || 'chrome',
      echConfigList: streamEditor.tlsEchConfigList,
    },
  };
}

function validateInboundEditorSettings(settingsText: string, streamSettingsText: string): string {
  const settings = parseInboundSettingsText(settingsText);
  const stream = parseInboundStreamSettingsText(streamSettingsText);
  if (isHysteriaProtocol(inboundEditor.protocol)) {
    if (Number(settings.version || 2) !== 2) {
      return 'Hysteria2 settings.version must be 2';
    }
    if (stream.security !== 'tls' || stream.network !== 'hysteria') {
      return 'Hysteria2 requires hysteria network with tls security';
    }
    if (!hasUsableTlsCertificate(stream)) {
      return 'Hysteria2 requires TLS certificate file paths or inline certificate content';
    }
  }
  if (inboundEditor.protocol === 'wireguard') {
    if (!settings.secretKey || !settings.pubKey) {
      return 'WireGuard server keys are required';
    }
    if (!Array.isArray(settings.peers) || settings.peers.length === 0) {
      return 'WireGuard requires at least one peer';
    }
  }
  if (stream.security === 'reality') {
    const realitySettings = objectField(stream.realitySettings);
    const realityValidationError = validateRealityServerSettings({
      target: stringField(realitySettings.target),
      serverNames: arrayField(realitySettings.serverNames),
      privateKey: stringField(realitySettings.privateKey),
      shortIds: arrayField(realitySettings.shortIds),
    });
    if (realityValidationError) {
      return realityValidationError;
    }
  }
  return '';
}

/** 生成供 Super-Code-Gateway 使用的本机 HTTP/SOCKS5 代理入站设置。 */
function gatewayProxySettings(template: GatewayProxyTemplate): Record<string, unknown> {
  if (template === 'mixed') {
    return {
      auth: 'noauth',
      accounts: [],
      udp: false,
      ip: '127.0.0.1',
    };
  }
  return {
    accounts: [],
    allowTransparent: false,
  };
}

/** 创建入站编辑器状态，并允许模板覆盖默认字段。 */
function createInboundEditor(
  protocol: XrayEditableInboundProtocol = 'vless',
  overrides: Partial<Omit<InboundEditorState, 'protocol'>> = {},
): InboundEditorState {
  return {
    protocol,
    remark: '',
    listen: '',
    port: randomPort(),
    enable: true,
    totalGB: 0,
    expiryTime: 0,
    trafficReset: 'never',
    settings: stringifyJson(defaultInboundSettings(protocol)),
    streamSettings: stringifyJson(defaultStreamSettings(protocol)),
    sniffing: stringifyJson(defaultSniffingSettings()),
    ...overrides,
  };
}

function createClientEditor(
  protocol: XrayEditableInboundProtocol = 'vless',
  inbound?: Inbound,
): ClientEditorState {
  const wireguardPeer =
    protocol === 'wireguard'
      ? generateWireguardPeer(inbound ? inboundClientCount(inbound) : 0)
      : null;
  return {
    protocol,
    originalClientId: '',
    id: usesUuidClientId(protocol) ? randomUuid() : '',
    password: usesPasswordClientId(protocol) ? generateClientCredential(protocol, inbound) : '',
    method: protocol === 'shadowsocks' && inbound ? getShadowsocksMethod(inbound) : '',
    auth: usesAuthClientId(protocol) ? generateClientCredential(protocol, inbound) : '',
    privateKey: wireguardPeer?.privateKey || '',
    publicKey: wireguardPeer?.publicKey || '',
    preSharedKey: '',
    allowedIPs: (wireguardPeer?.allowedIPs || ['10.0.0.2/32']).join('\n'),
    keepAlive: 0,
    email: wireguardPeer?.email || '',
    security: 'auto',
    flow: '',
    limitIp: 0,
    totalGB: 0,
    expiryTime: 0,
    enable: true,
    subId: wireguardPeer?.subId || randomToken(16),
    comment: '',
    reset: 0,
  };
}

function buildClientPayload(): InboundClient {
  return buildClientPayloadFromEditor(clientEditor);
}

function buildClientPayloadFromEditor(editor: ClientEditorState): InboundClient {
  const client: InboundClient = {
    email: editor.email.trim(),
    limitIp: Math.max(0, Number(editor.limitIp || 0)),
    totalGB: gbToBytes(editor.totalGB),
    expiryTime: Math.max(0, Number(editor.expiryTime || 0)),
    enable: editor.enable,
    tgId: 0,
    subId: editor.subId.trim() || randomToken(16),
    comment: editor.comment.trim(),
    reset: Math.max(0, Number(editor.reset || 0)),
  };

  if (editor.protocol === 'vmess') {
    client.id = editor.id.trim();
    client.security = editor.security || 'auto';
  } else if (editor.protocol === 'vless') {
    client.id = editor.id.trim();
    client.flow = editor.flow || '';
  } else if (editor.protocol === 'trojan') {
    client.password = editor.password.trim();
  } else if (editor.protocol === 'shadowsocks') {
    client.password = editor.password.trim();
    const method = editor.method.trim();
    if (method && !isShadowsocks2022Method(method)) {
      client.method = method;
    }
  } else if (isHysteriaProtocol(editor.protocol)) {
    client.auth = editor.auth.trim();
  } else if (editor.protocol === 'wireguard') {
    client.privateKey = editor.privateKey.trim();
    client.publicKey = editor.publicKey.trim();
    client.preSharedKey = editor.preSharedKey.trim() || undefined;
    client.allowedIPs = parseListText(editor.allowedIPs).map(normalizeAllowedIp);
    client.keepAlive = Math.max(0, Number(editor.keepAlive || 0));
    delete client.limitIp;
    delete client.totalGB;
    delete client.expiryTime;
    delete client.tgId;
    delete client.comment;
    delete client.reset;
  }
  return client;
}

function validateClientPayload(inbound: Inbound, client: InboundClient): string {
  if (!client.email) {
    return 'Client email is required';
  }
  if ((inbound.protocol === 'vmess' || inbound.protocol === 'vless') && !client.id) {
    return 'Client UUID is required';
  }
  if (inbound.protocol === 'trojan' && !client.password) {
    return 'Client password is required';
  }
  if (inbound.protocol === 'shadowsocks') {
    const method = getShadowsocksMethod(inbound);
    if (isSingleUserShadowsocks2022(method)) {
      return 'This Shadowsocks 2022 method does not support clients';
    }
    if (!method) {
      return 'Shadowsocks method is required';
    }
    if (!client.password) {
      return 'Shadowsocks password is required';
    }
    if (!isShadowsocks2022Method(method) && !client.method) {
      return 'Shadowsocks client method is required';
    }
  }
  if (isHysteriaProtocol(inbound.protocol) && !client.auth) {
    return 'Hysteria2 auth is required';
  }
  if (inbound.protocol === 'wireguard') {
    if (!client.privateKey || !client.publicKey) {
      return 'WireGuard private key and public key are required';
    }
    if (!client.allowedIPs?.length) {
      return 'WireGuard allowed IPs are required';
    }
  }
  return '';
}

function normalizeJsonEditorText(text: string, label: string): string | null {
  try {
    const parsed = JSON.parse(text) as unknown;
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      error.value = `${label} must be a JSON object`;
      return null;
    }
    error.value = '';
    return stringifyJson(parsed);
  } catch {
    error.value = `${label} is not valid JSON`;
    return null;
  }
}

function formatJsonText(text: string, fallback: object): string {
  if (!text.trim()) {
    return stringifyJson(fallback);
  }
  try {
    return stringifyJson(JSON.parse(text));
  } catch {
    return text;
  }
}

function syncSelectedInbound() {
  if (!selectedInbound.value) {
    return;
  }
  selectedInbound.value =
    inbounds.value.find((inbound) => inbound.id === selectedInbound.value?.id) ||
    selectedInbound.value;
}

function inboundAddress(record: Inbound | Record<string, unknown>): string {
  const inbound = asInbound(record);
  return `${resolveInboundHost(inbound)}:${inbound.port}`;
}

function inboundNetwork(record: Inbound | Record<string, unknown>): string {
  const inbound = asInbound(record);
  if (inbound.protocol === 'wireguard') {
    return 'wireguard';
  }
  return getInboundNetwork(inbound);
}

function inboundSecurity(record: Inbound | Record<string, unknown>): string {
  const inbound = asInbound(record);
  if (inbound.protocol === 'wireguard') {
    return 'none';
  }
  return getInboundSecurity(inbound);
}

function inboundClientCount(record: Inbound | Record<string, unknown>): number {
  const inbound = asInbound(record);
  return getInboundClients(inbound).length;
}

function inboundShareExportDisabled(record: Inbound | Record<string, unknown>): boolean {
  return buildInboundShareLinks(asInbound(record)).length === 0;
}

function formatTraffic(up: number, down: number, total: number): string {
  const used = Math.max(0, Number(up || 0) + Number(down || 0));
  return `${formatBytes(used)} / ${formatLimit(total)}`;
}

function formatLimit(value: number | undefined): string {
  const limit = Number(value || 0);
  return limit > 0 ? formatBytes(limit) : 'Unlimited';
}

function formatTimestamp(value: number | undefined): string {
  const timestamp = Number(value || 0);
  if (!timestamp) {
    return '-';
  }
  if (timestamp < 0) {
    return `${Math.abs(Math.round(timestamp / 86_400_000))} days`;
  }
  const date = new Date(timestamp);
  return Number.isNaN(date.getTime()) ? String(timestamp) : date.toLocaleString();
}

function isClientOnline(email: string | undefined): boolean {
  return Boolean(email && onlineClients.value.includes(email));
}

function formatClientLastOnline(row: ClientRow): string {
  const email = row.email || '';
  return formatTimestamp(lastOnlineMap.value[email] || row.traffic?.lastOnline || 0);
}

function formatClientIpRecords(value: string | string[]): string {
  if (Array.isArray(value)) {
    return value.length > 0 ? value.join('\n') : 'No IP Record';
  }
  return value?.trim() || 'No IP Record';
}

function protocolColor(protocol: string): string {
  return getProtocolRegistryEntry(protocol)?.color || 'default';
}

function isEditableProtocol(protocol: string): protocol is XrayEditableInboundProtocol {
  return isRegisteredEditableProtocol(protocol);
}

function usesUuidClientId(protocol: XrayEditableInboundProtocol): boolean {
  return protocol === 'vmess' || protocol === 'vless';
}

function usesPasswordClientId(protocol: XrayEditableInboundProtocol): boolean {
  return protocol === 'trojan' || protocol === 'shadowsocks';
}

function usesAuthClientId(protocol: XrayEditableInboundProtocol): boolean {
  return isHysteriaProtocol(protocol);
}

function isHysteriaProtocol(protocol: string): boolean {
  return protocol === 'hysteria' || protocol === 'hysteria2';
}

function hasClientPrimaryId(
  inbound: Inbound | null,
  row: InboundClient | Record<string, unknown>,
): boolean {
  return Boolean(inbound && getClientPrimaryId(inbound.protocol, row as InboundClient));
}

function clientAddDisabled(inbound: Inbound | null): boolean {
  if (!inbound || inbound.protocol !== 'shadowsocks') {
    return false;
  }
  return isSingleUserShadowsocks2022(getShadowsocksMethod(inbound));
}

function clientResetDisabled(
  inbound: Inbound | null,
  row: InboundClient | Record<string, unknown>,
): boolean {
  const client = row as InboundClient;
  return (
    !selectedInboundClientManageable.value || !client.email || inbound?.protocol === 'wireguard'
  );
}

function clientActionDisabled(
  inbound: Inbound | null,
  row: InboundClient | Record<string, unknown>,
): boolean {
  return !selectedInboundClientManageable.value || !hasClientPrimaryId(inbound, row);
}

function clientShareDisabled(
  inbound: Inbound | null,
  row: InboundClient | Record<string, unknown>,
): boolean {
  return !protocolSupportsShareLink(inbound?.protocol || '') || !hasClientPrimaryId(inbound, row);
}

function clientSubscriptionDisabled(
  inbound: Inbound | null,
  row: InboundClient | Record<string, unknown>,
): boolean {
  const client = row as InboundClient;
  return !inbound || !hasClientPrimaryId(inbound, row) || !client.subId;
}

function clientPrimaryText(
  inbound: Inbound | null,
  row: InboundClient | Record<string, unknown>,
): string {
  if (!inbound) {
    return '-';
  }
  const client = row as InboundClient;
  if (inbound.protocol === 'shadowsocks') {
    return client.method || getShadowsocksMethod(inbound) || '-';
  }
  if (isHysteriaProtocol(inbound.protocol)) {
    return client.auth || '-';
  }
  if (inbound.protocol === 'wireguard') {
    return client.allowedIPs?.join(', ') || client.publicKey || '-';
  }
  return getClientPrimaryId(inbound.protocol, client) || '-';
}

function generateClientCredential(
  protocol: XrayEditableInboundProtocol = clientEditor.protocol,
  inbound: Inbound | null | undefined = clientInbound.value,
): string {
  if (protocol === 'vmess' || protocol === 'vless') {
    return randomUuid();
  }
  if (protocol === 'shadowsocks') {
    const method = inbound ? getShadowsocksMethod(inbound) : clientEditor.method;
    return generateShadowsocksPassword(method);
  }
  if (isHysteriaProtocol(protocol)) {
    return randomToken(32);
  }
  return randomToken(32);
}

function parseInboundSettingsText(text: string): Record<string, unknown> {
  try {
    const parsed = JSON.parse(text) as unknown;
    return objectField(parsed);
  } catch {
    return {};
  }
}

function parseInboundStreamSettingsText(text: string): Record<string, unknown> {
  try {
    const parsed = JSON.parse(text) as unknown;
    return objectField(parsed);
  } catch {
    return defaultStreamSettings(inboundEditor.protocol) as Record<string, unknown>;
  }
}

function objectField(value: unknown): Record<string, unknown> {
  return value && typeof value === 'object' && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : {};
}

function stringField(value: unknown): string {
  return typeof value === 'string' ? value : '';
}

/** 编码 URI 中的用户名和密码字段，避免特殊字符破坏代理地址。 */
function encodeUriCredential(value: string): string {
  return value ? encodeURIComponent(value) : '';
}

function arrayField(value: unknown): string[] {
  if (Array.isArray(value)) {
    return value.filter((item): item is string => typeof item === 'string');
  }
  if (typeof value === 'string' && value.trim()) {
    return parseListText(value);
  }
  return [];
}

function parseListText(value: string): string[] {
  return value
    .split(/[\n,]/)
    .map((item) => item.trim())
    .filter(Boolean);
}

function normalizeAllowedIp(value: string): string {
  const trimmed = value.trim();
  if (!trimmed || trimmed.includes('/')) {
    return trimmed;
  }
  return trimmed.includes(':') ? `${trimmed}/128` : `${trimmed}/32`;
}

function defaultNetworkForProtocol(protocol: XrayEditableInboundProtocol): string {
  return isHysteriaProtocol(protocol) ? 'hysteria' : 'tcp';
}

function defaultSecurityForProtocol(protocol: XrayEditableInboundProtocol): string {
  return isHysteriaProtocol(protocol) ? 'tls' : 'none';
}

function defaultTlsAlpnForProtocol(protocol: XrayEditableInboundProtocol): string {
  return isHysteriaProtocol(protocol) ? 'h3' : 'h2,http/1.1';
}

function hasUsableTlsCertificate(stream: Record<string, unknown>): boolean {
  const tlsSettings = objectField(stream.tlsSettings);
  const certificates = Array.isArray(tlsSettings.certificates) ? tlsSettings.certificates : [];
  return certificates.some((entry) => {
    const certificate = objectField(entry);
    const certificateFile = stringField(certificate.certificateFile).trim();
    const keyFile = stringField(certificate.keyFile).trim();
    if (certificateFile && keyFile) {
      return true;
    }
    return arrayField(certificate.certificate).length > 0 && arrayField(certificate.key).length > 0;
  });
}

function generateWireguardServerKeys() {
  const keypair = generateWireguardKeypair();
  wireguardEditor.secretKey = keypair.privateKey;
  wireguardEditor.pubKey = keypair.publicKey;
}

function generateWireguardClientKeys() {
  const keypair = generateWireguardKeypair();
  clientEditor.privateKey = keypair.privateKey;
  clientEditor.publicKey = keypair.publicKey;
}

function asInbound(record: Inbound | Record<string, unknown>): Inbound {
  return record as Inbound;
}

function asClientRow(record: ClientRow | Record<string, unknown>): ClientRow {
  return record as ClientRow;
}

function buildClientRows(inbound: Inbound): ClientRow[] {
  return getInboundClients(inbound).map((client, index) => {
    const traffic = inbound.clientStats?.find((stats) => stats.email === client.email);
    const primaryId = getClientPrimaryId(inbound.protocol, client);
    return {
      ...client,
      key: primaryId || `${client.email}-${index}`,
      traffic,
    };
  });
}

function pickSubscriptionSettings(payload: PanelSettings): SubscriptionLinkSettings {
  return {
    subEnable: payload.subEnable,
    subJsonEnable: payload.subJsonEnable,
    subClashEnable: payload.subClashEnable,
    subURI: payload.subURI,
    subJsonURI: payload.subJsonURI,
    subClashURI: payload.subClashURI,
  };
}

function safeExportFilename(value: string, extension = 'txt'): string {
  const withoutControls = Array.from(value)
    .filter((char) => char.charCodeAt(0) >= 32)
    .join('');
  const stem = withoutControls
    .trim()
    .replace(/[<>:"/\\|?*]/g, '-')
    .replace(/\s+/g, '-')
    .replace(/-+/g, '-')
    .replace(/^-|-$/g, '');
  return `${stem || 'inbounds-export'}.${extension}`;
}

async function ensureSubscriptionSettingsLoaded(): Promise<SubscriptionLinkSettings | null> {
  if (subscriptionSettings.value) {
    return subscriptionSettings.value;
  }
  if (!hasInjectedRuntimeConfig()) {
    return null;
  }

  loadingSubscriptionSettings.value = true;
  error.value = '';
  try {
    const payload = await getAllSettings({ notifyOnError: false });
    let settings = pickSubscriptionSettings(payload);
    if (hasMissingEnabledSubscriptionEndpoint(settings)) {
      const defaults = pickSubscriptionSettings(
        await getDefaultSettings({ notifyOnError: false }),
      );
      settings = mergeSubscriptionEndpointDefaults(settings, defaults);
    }
    subscriptionSettings.value = settings;
    return subscriptionSettings.value;
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to load subscription settings';
    return null;
  } finally {
    loadingSubscriptionSettings.value = false;
  }
}

function hasMissingEnabledSubscriptionEndpoint(settings: SubscriptionLinkSettings): boolean {
  return Boolean(
    (settings.subEnable && !settings.subURI.trim()) ||
      (settings.subJsonEnable && !settings.subJsonURI.trim()) ||
      (settings.subClashEnable && !settings.subClashURI.trim()),
  );
}

function formatSubscriptionPreview(
  title: string,
  links: Array<{ label: string; url: string }>,
): string {
  return [`[${title}]`, ...links.map((link) => `${link.label}: ${link.url}`)].join('\n');
}

function createBulkClientForm(): BulkClientFormState {
  return {
    quantity: 1,
    firstIndex: 1,
    emailPrefix: 'client-',
    emailPostfix: '',
    totalGB: 0,
    expiryTime: 0,
    limitIp: 0,
    reset: 0,
    flow: '',
  };
}

function resetBulkClientForm() {
  Object.assign(bulkClientForm, createBulkClientForm());
}

function buildClonedInboundForm(inbound: Inbound): InboundForm {
  const protocol = isEditableProtocol(inbound.protocol) ? inbound.protocol : null;
  return {
    up: 0,
    down: 0,
    total: inbound.total,
    allTime: 0,
    remark: `${inbound.remark || inbound.tag || `Inbound ${inbound.id}`} - Cloned`,
    enable: inbound.enable,
    expiryTime: inbound.expiryTime,
    trafficReset: inbound.trafficReset,
    lastTrafficResetTime: inbound.lastTrafficResetTime,
    listen: '',
    port: randomPort(),
    protocol: inbound.protocol,
    settings: protocol
      ? stringifyJson(defaultInboundSettings(protocol))
      : formatJsonText(inbound.settings, parseInboundSettings(inbound)),
    streamSettings: formatJsonText(inbound.streamSettings, parseInboundStreamSettings(inbound)),
    sniffing: formatJsonText(inbound.sniffing, parseInboundSniffingSettings(inbound)),
  };
}

function isOpenablePublicLink(value: string): boolean {
  try {
    const url = new URL(value);
    return url.protocol === 'http:' || url.protocol === 'https:';
  } catch {
    return false;
  }
}

async function loadQrious(): Promise<QriousConstructor> {
  if (qriousLoader) {
    return qriousLoader;
  }

  qriousLoader = new Promise<QriousConstructor>((resolve, reject) => {
    const existing = (window as Window & { QRious?: QriousConstructor }).QRious;
    if (existing) {
      resolve(existing);
      return;
    }

    const runtime = getRuntimeConfig();
    const script = document.createElement('script');
    script.src = `${runtime.basePath}assets/qrcode/qrious2.min.js?${runtime.version}`;
    script.async = true;
    script.onload = () => {
      const loaded = (window as Window & { QRious?: QriousConstructor }).QRious;
      if (loaded) {
        resolve(loaded);
        return;
      }
      reject(new Error('QRious did not load'));
    };
    script.onerror = () => reject(new Error('Failed to load QRious script'));
    document.head.appendChild(script);
  });

  return qriousLoader;
}

async function renderClientAccessQrs() {
  if (!clientAccessModalOpen.value || clientAccessLinks.value.length === 0) {
    return;
  }
  try {
    const QRious = await loadQrious();
    await nextTick();
    clientAccessLinks.value.forEach((item, index) => {
      const canvas = document.getElementById(`client-access-qr-${index}`);
      if (!(canvas instanceof HTMLCanvasElement)) {
        return;
      }
      new QRious({
        element: canvas,
        size: 220,
        value: item.url,
      });
    });
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Failed to render QR codes';
  }
}

function stripClientRow(row: ClientRow): InboundClient {
  const client: InboundClient = { ...row };
  delete client.key;
  delete client.traffic;
  return client;
}

function bytesToGb(value: number): number {
  if (!Number.isFinite(value) || value <= 0) {
    return 0;
  }
  return Number((value / 1024 ** 3).toFixed(2));
}

function gbToBytes(value: number): number {
  if (!Number.isFinite(value) || value <= 0) {
    return 0;
  }
  return Math.round(value * 1024 ** 3);
}

function randomPort(): number {
  return Math.floor(10_000 + Math.random() * 50_000);
}

function randomUuid(): string {
  if (crypto.randomUUID) {
    return crypto.randomUUID();
  }
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (token) => {
    const value = Math.floor(Math.random() * 16);
    const digit = token === 'x' ? value : (value & 0x3) | 0x8;
    return digit.toString(16);
  });
}

function randomToken(length: number): string {
  const alphabet = 'abcdefghijklmnopqrstuvwxyz0123456789';
  let token = '';
  for (let index = 0; index < length; index += 1) {
    token += alphabet[Math.floor(Math.random() * alphabet.length)];
  }
  return token;
}

onMounted(() => {
  void refreshInbounds();
});
</script>
