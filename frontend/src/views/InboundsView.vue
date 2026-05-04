<template>
  <section class="page-stack">
    <PageHeader eyebrow="Traffic" title="Inbounds">
      <ASpace wrap>
        <AButton :loading="loading" @click="refreshInbounds">
          <template #icon><ReloadOutlined /></template>
          Refresh
        </AButton>
        <AButton :loading="loadingActivity" @click="refreshClientActivity">
          <template #icon><ReloadOutlined /></template>
          Refresh Activity
        </AButton>
        <AButton @click="openImportInbound">
          <template #icon><PlusOutlined /></template>
          Import JSON
        </AButton>
        <AButton danger :loading="resettingAllTraffic" @click="confirmResetAllInboundTraffic">
          Reset All Traffic
        </AButton>
        <AButton type="primary" @click="openCreateInbound">
          <template #icon><PlusOutlined /></template>
          New Inbound
        </AButton>
      </ASpace>
    </PageHeader>

    <AAlert v-if="error" banner type="warning" :message="error" />

    <div class="status-grid">
      <StatusTile label="Total" :value="formatCount(inbounds.length)" hint="Legacy Xray inbounds" />
      <StatusTile
        label="Enabled"
        :value="formatCount(enabledInboundCount)"
        hint="Active listeners"
      />
      <StatusTile
        label="Online Clients"
        :value="formatCount(onlineClients.length)"
        hint="Live activity from Xray"
      />
      <StatusTile label="Clients" :value="formatCount(clientCount)" hint="Configured users" />
      <StatusTile label="Traffic" :value="trafficTotal" hint="Inbound counters" />
    </div>

    <ACard class="work-panel" :bordered="false">
      <div class="toolbar-grid inbounds-toolbar">
        <ASelect
          v-model:value="protocolFilter"
          aria-label="Protocol filter"
          :options="protocolFilterOptions"
        />
        <ASelect
          v-model:value="stateFilter"
          aria-label="State filter"
          :options="stateFilterOptions"
        />
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
                :disabled="!selectedClientRows.length"
                @click="exportClientLinks(selectedInbound, selectedClientRows)"
              >
                <template #icon><CopyOutlined /></template>
                Export Links
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
                :disabled="!selectedInboundEditable || selectedInbound.protocol === 'wireguard'"
                :loading="busyClient"
                @click="confirmResetAllClients(selectedInbound)"
              >
                <template #icon><ReloadOutlined /></template>
                Reset All
              </AButton>
              <AButton
                danger
                :disabled="!selectedInboundEditable"
                :loading="deletingDepletedClients"
                @click="confirmDeleteDepletedClients(selectedInbound)"
              >
                Delete Depleted
              </AButton>
              <AButton
                :disabled="!selectedInboundEditable || clientAddDisabled(selectedInbound)"
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
                  :disabled="
                    !selectedInboundEditable || !hasClientPrimaryId(selectedInbound, record)
                  "
                  :loading="busyClientKey === record.key"
                  @change="(checked) => toggleClient(selectedInbound, record, Boolean(checked))"
                />
              </template>

              <template v-else-if="column.key === 'actions'">
                <ASpace wrap>
                  <AButton
                    :disabled="
                      !selectedInboundEditable || !hasClientPrimaryId(selectedInbound, record)
                    "
                    size="small"
                    @click="openEditClient(selectedInbound, record)"
                  >
                    <template #icon><EditOutlined /></template>
                    Edit
                  </AButton>
                  <AButton
                    :disabled="!hasClientPrimaryId(selectedInbound, record)"
                    size="small"
                    @click="previewShareLink(selectedInbound, record)"
                  >
                    <template #icon><CopyOutlined /></template>
                    Share
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
                    :disabled="
                      !selectedInboundEditable || !hasClientPrimaryId(selectedInbound, record)
                    "
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
              <h2>Share Link</h2>
            </div>
            <AButton @click="copySharePreview">
              <template #icon><CopyOutlined /></template>
              Copy
            </AButton>
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
      <AForm layout="vertical">
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

        <div v-if="inboundEditor.protocol === 'wireguard'" class="json-section">
          <div class="json-section-title">
            <span>WireGuard Settings</span>
            <ASpace>
              <AButton size="small" @click="syncWireguardEditorFromSettings">Sync JSON</AButton>
              <AButton size="small" @click="applyWireguardEditorToSettings">Apply</AButton>
            </ASpace>
          </div>
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
        </div>

        <div v-if="inboundEditor.protocol !== 'wireguard'" class="json-section">
          <div class="json-section-title">
            <span>Stream Settings Form</span>
            <ASpace>
              <AButton size="small" @click="syncStreamEditorFromSettings">Sync JSON</AButton>
              <AButton size="small" @click="applyStreamEditorToSettings">Apply</AButton>
            </ASpace>
          </div>
          <div class="form-grid">
            <AFormItem label="Network">
              <ASelect v-model:value="streamEditor.network" :options="transportNetworkOptions" />
            </AFormItem>
            <AFormItem label="Security">
              <ASelect v-model:value="streamEditor.security" :options="transportSecurityOptions" />
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
            <AFormItem v-if="streamEditor.network === 'ws'" label="WS Path">
              <AInput v-model:value="streamEditor.wsPath" />
            </AFormItem>
            <AFormItem v-if="streamEditor.network === 'ws'" label="WS Host">
              <AInput v-model:value="streamEditor.wsHost" />
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
            <AFormItem v-if="streamEditor.network === 'httpupgrade'" label="HTTPUpgrade Path">
              <AInput v-model:value="streamEditor.httpupgradePath" />
            </AFormItem>
            <AFormItem v-if="streamEditor.network === 'httpupgrade'" label="HTTPUpgrade Host">
              <AInput v-model:value="streamEditor.httpupgradeHost" />
            </AFormItem>
            <AFormItem v-if="streamEditor.security === 'tls'" label="TLS SNI">
              <AInput v-model:value="streamEditor.tlsServerName" />
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
          </div>
        </div>

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

    <AModal v-model:open="clientIpsModalOpen" :footer="null" :title="clientIpsModalTitle">
      <textarea
        aria-label="Client IP records"
        class="json-editor compact-json-editor"
        readonly
        :value="clientIpsText"
      />
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
  CopyOutlined,
  DeleteOutlined,
  EditOutlined,
  EyeOutlined,
  PlusOutlined,
  ReloadOutlined,
  UserAddOutlined,
} from '@ant-design/icons-vue';
import {
  Alert as AAlert,
  Button as AButton,
  Card as ACard,
  Drawer as ADrawer,
  Form as AForm,
  FormItem as AFormItem,
  Input as AInput,
  InputNumber as AInputNumber,
  Modal as AModal,
  Select as ASelect,
  Space as ASpace,
  Switch as ASwitch,
  Table as ATable,
  Tag as ATag,
  message,
} from 'ant-design-vue';
import { computed, onMounted, reactive, ref, watch } from 'vue';

import {
  addInbound,
  addInboundClient,
  clearClientIps,
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
import PageHeader from '@/components/PageHeader.vue';
import StatusTile from '@/components/StatusTile.vue';
import { hasInjectedRuntimeConfig } from '@/types/runtime';
import type {
  ClientTraffic,
  Inbound,
  InboundClient,
  InboundForm,
  XrayEditableInboundProtocol,
} from '@/types/inbound';
import { formatBytes, formatCount } from '@/utils/format';
import {
  SHADOWSOCKS_METHOD_OPTIONS,
  buildClientShareLink,
  defaultInboundSettings,
  defaultSniffingSettings,
  defaultStreamSettings,
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
  parseInboundSettings,
  parseInboundSniffingSettings,
  parseInboundStreamSettings,
  resolveInboundHost,
  stringifyJson,
} from '@/utils/inboundCompat';
import { copyText } from '@/utils/textExport';

type InboundModalMode = 'create' | 'edit';
type ClientModalMode = 'create' | 'edit';
type InboundJsonField = 'settings' | 'streamSettings' | 'sniffing';
type StateFilter = 'all' | 'enabled' | 'disabled';

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
  tcpHeaderType: string;
  wsPath: string;
  wsHost: string;
  grpcServiceName: string;
  grpcAuthority: string;
  grpcMultiMode: boolean;
  httpupgradePath: string;
  httpupgradeHost: string;
  tlsServerName: string;
  tlsAlpn: string;
  tlsFingerprint: string;
  tlsCertificateFile: string;
  tlsKeyFile: string;
  realityTarget: string;
  realityServerNames: string;
  realityPrivateKey: string;
  realityShortIds: string;
  hysteriaAuth: string;
  hysteriaUdpIdleTimeout: number;
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

const inbounds = ref<Inbound[]>([]);
const loading = ref(false);
const error = ref('');
const keyword = ref('');
const protocolFilter = ref('all');
const stateFilter = ref<StateFilter>('all');
const detailOpen = ref(false);
const selectedInbound = ref<Inbound | null>(null);
const sharePreview = ref('');
const inboundModalOpen = ref(false);
const inboundModalMode = ref<InboundModalMode>('create');
const savingInbound = ref(false);
const busyInboundId = ref<number | null>(null);
const importModalOpen = ref(false);
const importInboundText = ref('');
const importingInbound = ref(false);
const resettingAllTraffic = ref(false);
const clientModalOpen = ref(false);
const clientModalMode = ref<ClientModalMode>('create');
const clientInbound = ref<Inbound | null>(null);
const selectedClientRowKeys = ref<string[]>([]);
const savingClient = ref(false);
const busyClient = ref(false);
const busyClientKey = ref('');
const deletingDepletedClients = ref(false);
const onlineClients = ref<string[]>([]);
const lastOnlineMap = ref<Record<string, number>>({});
const loadingActivity = ref(false);
const clientIpsModalOpen = ref(false);
const clientIpsModalTitle = ref('Client IP Records');
const clientIpsText = ref('');
const clearingClientIpsEmail = ref('');

const inboundEditor = reactive<InboundEditorState>(createInboundEditor());
const wireguardEditor = reactive<WireguardEditorState>(createWireguardEditor());
const streamEditor = reactive<StreamEditorState>(createStreamEditor());
const clientEditor = reactive<ClientEditorState>(createClientEditor());

const editableProtocolOptions = [
  { label: 'VMess', value: 'vmess' },
  { label: 'VLESS', value: 'vless' },
  { label: 'Trojan', value: 'trojan' },
  { label: 'Shadowsocks', value: 'shadowsocks' },
  { label: 'Hysteria2', value: 'hysteria' },
  { label: 'WireGuard', value: 'wireguard' },
];
const protocolFilterOptions = computed(() => [
  { label: 'All protocols', value: 'all' },
  ...Array.from(new Set(inbounds.value.map((inbound) => inbound.protocol))).map((protocol) => ({
    label: protocol,
    value: protocol,
  })),
]);
const stateFilterOptions = [
  { label: 'All states', value: 'all' },
  { label: 'Enabled', value: 'enabled' },
  { label: 'Disabled', value: 'disabled' },
];
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
        { label: 'ws', value: 'ws' },
        { label: 'grpc', value: 'grpc' },
        { label: 'httpupgrade', value: 'httpupgrade' },
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
const selectedInboundEditable = computed(() =>
  selectedInbound.value ? isEditableProtocol(selectedInbound.value.protocol) : false,
);
const selectedClientRows = computed<ClientRow[]>(() => {
  const inbound = selectedInbound.value;
  if (!inbound) {
    return [];
  }
  return getInboundClients(inbound).map((client, index) => {
    const traffic = inbound.clientStats?.find((stats) => stats.email === client.email);
    const primaryId = getClientPrimaryId(inbound.protocol, client);
    return {
      ...client,
      key: primaryId || `${client.email}-${index}`,
      traffic,
    };
  });
});
const selectedBatchClientRows = computed(() =>
  selectedClientRows.value.filter((row) => selectedClientRowKeys.value.includes(row.key)),
);
const resettableSelectedClientRows = computed(() =>
  selectedBatchClientRows.value.filter((row) => !clientResetDisabled(selectedInbound.value, row)),
);
const canResetSelectedClients = computed(
  () => resettableSelectedClientRows.value.length > 0 && !busyClient.value,
);
const canDeleteSelectedClients = computed(() => {
  if (!selectedInboundEditable.value || busyClient.value) {
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
const inboundModalTitle = computed(() =>
  inboundModalMode.value === 'create' ? 'New Inbound' : 'Edit Inbound',
);
const clientModalTitle = computed(() =>
  clientModalMode.value === 'create' ? 'Add Client' : 'Edit Client',
);

watch(
  () => inboundEditor.protocol,
  (protocol) => {
    if (inboundModalMode.value === 'create') {
      inboundEditor.settings = stringifyJson(defaultInboundSettings(protocol));
      inboundEditor.streamSettings = stringifyJson(defaultStreamSettings(protocol));
      syncWireguardEditorFromSettings();
      syncStreamEditorFromSettings();
    }
  },
);

watch(selectedClientRows, (rows) => {
  const keys = new Set(rows.map((row) => row.key));
  selectedClientRowKeys.value = selectedClientRowKeys.value.filter((key) => keys.has(key));
});

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
  syncWireguardEditorFromSettings();
  syncStreamEditorFromSettings();
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
  inboundModalOpen.value = true;
}

async function submitInbound() {
  if (inboundEditor.protocol === 'wireguard') {
    applyWireguardEditorToSettings();
  } else {
    applyStreamEditorToSettings();
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

function openInboundDetail(record: Inbound | Record<string, unknown>) {
  const inbound = asInbound(record);
  selectedInbound.value = inbound;
  selectedClientRowKeys.value = [];
  sharePreview.value = '';
  detailOpen.value = true;
}

function openCreateClient(inbound: Inbound) {
  if (!isEditableProtocol(inbound.protocol)) {
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
  if (!isEditableProtocol(inbound.protocol)) {
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
  if (!inbound || !isEditableProtocol(inbound.protocol)) {
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
  sharePreview.value = link;
  if (!link) {
    error.value = 'Share link is only available for supported clients with complete credentials';
    return;
  }
  await copyText(link);
  void message.success('Share link copied');
}

async function exportClientLinks(inbound: Inbound | null, rows: ClientRow[]) {
  if (!inbound) {
    return;
  }
  const links = rows
    .map((row) => buildClientShareLink(inbound, row))
    .filter((link) => link.trim().length > 0);
  if (links.length === 0) {
    error.value = 'No share links are available for the selected clients';
    sharePreview.value = '';
    return;
  }
  const text = links.join('\n');
  sharePreview.value = text;
  await copyText(text);
  void message.success(`${links.length} share links copied`);
}

async function copySharePreview() {
  if (!sharePreview.value) {
    return;
  }
  await copyText(sharePreview.value);
  void message.success('Copied');
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
    tcpHeaderType: 'none',
    wsPath: '/',
    wsHost: '',
    grpcServiceName: '',
    grpcAuthority: '',
    grpcMultiMode: false,
    httpupgradePath: '/',
    httpupgradeHost: '',
    tlsServerName: '',
    tlsAlpn: 'h2,http/1.1',
    tlsFingerprint: 'chrome',
    tlsCertificateFile: '',
    tlsKeyFile: '',
    realityTarget: '',
    realityServerNames: '',
    realityPrivateKey: '',
    realityShortIds: '',
    hysteriaAuth: '',
    hysteriaUdpIdleTimeout: 60,
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

function syncStreamEditorFromSettings() {
  const stream = parseInboundStreamSettingsText(inboundEditor.streamSettings);
  const tlsSettings = objectField(stream.tlsSettings);
  const tlsClientSettings = objectField(tlsSettings.settings);
  const firstCertificate = Array.isArray(tlsSettings.certificates)
    ? objectField(tlsSettings.certificates[0])
    : {};
  const realitySettings = objectField(stream.realitySettings);
  const wsSettings = objectField(stream.wsSettings);
  const grpcSettings = objectField(stream.grpcSettings);
  const httpupgradeSettings = objectField(stream.httpupgradeSettings);
  const tcpSettings = objectField(stream.tcpSettings);
  const tcpHeader = objectField(tcpSettings.header);
  const hysteriaSettings = objectField(stream.hysteriaSettings);

  Object.assign(streamEditor, {
    network: stringField(stream.network) || defaultNetworkForProtocol(inboundEditor.protocol),
    security: stringField(stream.security) || defaultSecurityForProtocol(inboundEditor.protocol),
    tcpHeaderType: stringField(tcpHeader.type) || 'none',
    wsPath: stringField(wsSettings.path) || '/',
    wsHost: stringField(wsSettings.host),
    grpcServiceName: stringField(grpcSettings.serviceName),
    grpcAuthority: stringField(grpcSettings.authority),
    grpcMultiMode: Boolean(grpcSettings.multiMode),
    httpupgradePath: stringField(httpupgradeSettings.path) || '/',
    httpupgradeHost: stringField(httpupgradeSettings.host),
    tlsServerName: stringField(tlsSettings.serverName),
    tlsAlpn:
      arrayField(tlsSettings.alpn).join(',') || defaultTlsAlpnForProtocol(inboundEditor.protocol),
    tlsFingerprint: stringField(tlsClientSettings.fingerprint) || 'chrome',
    tlsCertificateFile: stringField(firstCertificate.certificateFile),
    tlsKeyFile: stringField(firstCertificate.keyFile),
    realityTarget: stringField(realitySettings.target),
    realityServerNames: arrayField(realitySettings.serverNames).join(','),
    realityPrivateKey: stringField(realitySettings.privateKey),
    realityShortIds: arrayField(realitySettings.shortIds).join(','),
    hysteriaAuth: stringField(hysteriaSettings.auth),
    hysteriaUdpIdleTimeout: Number(hysteriaSettings.udpIdleTimeout || 60),
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
  delete stream.wsSettings;
  delete stream.grpcSettings;
  delete stream.httpupgradeSettings;
  delete stream.tlsSettings;
  delete stream.realitySettings;
  delete stream.hysteriaSettings;

  if (network === 'tcp') {
    stream.tcpSettings = {
      acceptProxyProtocol: false,
      header: {
        type: streamEditor.tcpHeaderType || 'none',
      },
    };
  }
  if (network === 'ws') {
    stream.wsSettings = {
      acceptProxyProtocol: false,
      path: streamEditor.wsPath || '/',
      host: streamEditor.wsHost,
      headers: streamEditor.wsHost ? { Host: streamEditor.wsHost } : {},
      heartbeatPeriod: 0,
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
      acceptProxyProtocol: false,
      path: streamEditor.httpupgradePath || '/',
      host: streamEditor.httpupgradeHost,
      headers: streamEditor.httpupgradeHost ? { Host: streamEditor.httpupgradeHost } : {},
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
    stream.realitySettings = {
      show: false,
      xver: 0,
      target: streamEditor.realityTarget,
      serverNames: parseListText(streamEditor.realityServerNames),
      privateKey: streamEditor.realityPrivateKey,
      minClientVer: '',
      maxClientVer: '',
      maxTimediff: 0,
      shortIds: parseListText(streamEditor.realityShortIds),
      settings: {
        publicKey: '',
        fingerprint: streamEditor.tlsFingerprint || 'chrome',
        serverName: parseListText(streamEditor.realityServerNames)[0] || '',
        spiderX: '/',
      },
    };
  }

  inboundEditor.streamSettings = stringifyJson(stream);
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
    minVersion: '1.2',
    maxVersion: '1.3',
    cipherSuites: '',
    rejectUnknownSni: false,
    disableSystemRoot: false,
    enableSessionResumption: false,
    certificates: nextCertificates,
    alpn: parseListText(streamEditor.tlsAlpn),
    settings: {
      fingerprint: streamEditor.tlsFingerprint || 'chrome',
      echConfigList: '',
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
  return '';
}

function createInboundEditor(): InboundEditorState {
  const protocol: XrayEditableInboundProtocol = 'vless';
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
  const client: InboundClient = {
    email: clientEditor.email.trim(),
    limitIp: Math.max(0, Number(clientEditor.limitIp || 0)),
    totalGB: gbToBytes(clientEditor.totalGB),
    expiryTime: Math.max(0, Number(clientEditor.expiryTime || 0)),
    enable: clientEditor.enable,
    tgId: 0,
    subId: clientEditor.subId.trim() || randomToken(16),
    comment: clientEditor.comment.trim(),
    reset: Math.max(0, Number(clientEditor.reset || 0)),
  };

  if (clientEditor.protocol === 'vmess') {
    client.id = clientEditor.id.trim();
    client.security = clientEditor.security || 'auto';
  } else if (clientEditor.protocol === 'vless') {
    client.id = clientEditor.id.trim();
    client.flow = clientEditor.flow || '';
  } else if (clientEditor.protocol === 'trojan') {
    client.password = clientEditor.password.trim();
  } else if (clientEditor.protocol === 'shadowsocks') {
    client.password = clientEditor.password.trim();
    const method = clientEditor.method.trim();
    if (method && !isShadowsocks2022Method(method)) {
      client.method = method;
    }
  } else if (isHysteriaProtocol(clientEditor.protocol)) {
    client.auth = clientEditor.auth.trim();
  } else if (clientEditor.protocol === 'wireguard') {
    client.privateKey = clientEditor.privateKey.trim();
    client.publicKey = clientEditor.publicKey.trim();
    client.preSharedKey = clientEditor.preSharedKey.trim() || undefined;
    client.allowedIPs = parseListText(clientEditor.allowedIPs).map(normalizeAllowedIp);
    client.keepAlive = Math.max(0, Number(clientEditor.keepAlive || 0));
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
  if (protocol === 'vmess') {
    return 'blue';
  }
  if (protocol === 'vless') {
    return 'green';
  }
  if (protocol === 'trojan') {
    return 'purple';
  }
  if (protocol === 'shadowsocks') {
    return 'cyan';
  }
  if (protocol === 'hysteria' || protocol === 'hysteria2') {
    return 'gold';
  }
  if (protocol === 'wireguard') {
    return 'geekblue';
  }
  return 'default';
}

function isEditableProtocol(protocol: string): protocol is XrayEditableInboundProtocol {
  return (
    protocol === 'vmess' ||
    protocol === 'vless' ||
    protocol === 'trojan' ||
    protocol === 'shadowsocks' ||
    protocol === 'hysteria' ||
    protocol === 'hysteria2' ||
    protocol === 'wireguard'
  );
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
  return !selectedInboundEditable.value || !client.email || inbound?.protocol === 'wireguard';
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
