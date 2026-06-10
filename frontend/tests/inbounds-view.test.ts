import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';

const source = readFileSync('frontend/src/views/InboundsView.vue', 'utf8');

test('new inbound form exposes default client flow fields for VLESS Vision', () => {
  assert.match(source, /title="Default Client"/);
  assert.match(source, /v-if="inboundVlessFlowVisible" label="Flow"/);
  assert.match(source, /v-model:value="inboundClientEditor\.flow"/);
  assert.match(source, /streamEditor\.network === 'tcp'/);
  assert.match(
    source,
    /streamEditor\.security === 'tls' \|\| streamEditor\.security === 'reality'/,
  );
});

test('new inbound submit syncs default client into settings JSON', () => {
  assert.match(
    source,
    /if \(inboundClientSectionVisible\.value\) \{\s*applyInboundClientEditorToSettings\(\);\s*\}/,
  );
  assert.match(source, /function applyInboundClientEditorToSettings\(\)/);
  assert.match(
    source,
    /settings\.clients = \[\{ \.\.\.existingClient, \.\.\.client \}, \.\.\.clients\.slice\(1\)\]/,
  );
  assert.match(source, /client\.flow = editor\.flow \|\| ''/);
});

test('gateway proxy templates expose local HTTP and SOCKS5 exits', () => {
  assert.match(source, /openGatewayProxyTemplate\('mixed'\)/);
  assert.match(source, /openGatewayProxyTemplate\('http'\)/);
  assert.match(source, /Gateway SOCKS5/);
  assert.match(source, /Gateway HTTP/);
  assert.match(source, /createInboundEditor\(protocol, \{/);
  assert.match(source, /settings: stringifyJson\(gatewayProxySettings\(template\)\)/);
  assert.match(source, /auth: 'noauth'/);
  assert.match(source, /listen: '127\.0\.0\.1'/);
  assert.match(source, /port: template === 'mixed' \? 1080 : 8081/);
});

test('gateway proxy URI section shows copyable Super-Code-Gateway proxy URIs', () => {
  assert.match(source, /title="Gateway Proxy URI"/);
  assert.match(source, /v-if="gatewayProxyUris\.length > 0"/);
  assert.match(source, /const gatewayProxyUris = computed<GatewayProxyUriItem\[\]>\(\(\) => \{/);
  assert.match(source, /socks5:\/\/\$\{auth\}\$\{host\}:\$\{port\}/);
  assert.match(source, /http:\/\/\$\{auth\}\$\{host\}:\$\{port\}/);
  assert.match(source, /function encodeUriCredential\(value: string\): string/);
  assert.match(source, /async function copyGatewayProxyUri\(uri: string\)/);
});

test('inbound modal is organized into reusable form sections', () => {
  assert.match(source, /import FormSection from '@\/components\/FormSection\.vue';/);
  assert.match(source, /class="responsive-modal-form"/);
  assert.match(source, /<FormSection\s+eyebrow="Inbound"\s+title="Basic Inbound"/);
  assert.match(
    source,
    /<FormSection\s+v-if="inboundEditor\.protocol === 'wireguard'"\s+eyebrow="Protocol"\s+title="WireGuard Settings"/,
  );
  assert.match(
    source,
    /<FormSection\s+v-if="protocolSupportsStream\(inboundEditor\.protocol\)"\s+eyebrow="Transport"\s+title="Transport Settings"/,
  );
  assert.match(
    source,
    /<FormSection\s+v-if="inboundClientSectionVisible"\s+eyebrow="Client"\s+title="Default Client"/,
  );
  assert.match(source, /<FormSection\s+eyebrow="Advanced"\s+title="Advanced JSON"/);
});

test('inbound form keeps default client and JSON action paths', () => {
  assert.match(source, /@click="syncInboundClientEditorFromSettings"/);
  assert.match(source, /@click="applyInboundClientEditorToSettings"/);
  assert.match(source, /@click="formatInboundJson\('settings'\)"/);
  assert.match(source, /@click="formatInboundJson\('streamSettings'\)"/);
  assert.match(source, /@click="formatInboundJson\('sniffing'\)"/);
  assert.match(source, /v-model="inboundEditor\.settings"/);
  assert.match(source, /v-model="inboundEditor\.streamSettings"/);
  assert.match(source, /v-model="inboundEditor\.sniffing"/);
});

test('inbound form validates and normalizes Reality server settings before saving', () => {
  assert.match(source, /normalizeRealityServerSettings/);
  assert.match(source, /validateRealityServerSettings/);
  assert.match(source, /streamEditor\.realityTarget = reality\.target/);
  assert.match(source, /return realityValidationError/);
});

test('hysteria inbound form applies panel default TLS certificate paths before validation', () => {
  assert.match(source, /applyPanelDefaultTlsCertificate/);
  assert.match(source, /const panelDefaultTlsCertificate/);
  assert.match(source, /defaults\.defaultCert/);
  assert.match(source, /defaults\.defaultKey/);
  assert.match(source, /await applyPanelDefaultTlsCertificateToEditor\(\)/);
});

test('hysteria default TLS async fill does not overwrite user-entered certificate fields', () => {
  assert.match(source, /const protocolSnapshot = inboundEditor\.protocol/);
  assert.match(source, /const streamSettingsSnapshot = inboundEditor\.streamSettings/);
  assert.match(source, /protocolSnapshot !== inboundEditor\.protocol/);
  assert.match(source, /streamSettingsSnapshot !== inboundEditor\.streamSettings/);
  assert.match(source, /streamEditor\.tlsCertificateFile\.trim\(\)/);
  assert.match(source, /streamEditor\.tlsKeyFile\.trim\(\)/);
});

test('hysteria stream editor keeps uTLS none instead of chrome fallback', () => {
  assert.match(source, /function defaultTlsFingerprintForProtocol/);
  assert.match(source, /isHysteriaProtocol\(protocol\) \? '' : 'chrome'/);
  assert.match(
    source,
    /tlsFingerprint:\s*stringField\(tlsClientSettings\.fingerprint\) \|\|\s*defaultTlsFingerprintForProtocol\(inboundEditor\.protocol\)/,
  );
  assert.match(
    source,
    /fingerprint:\s*streamEditor\.tlsFingerprint \|\| defaultTlsFingerprintForProtocol\(inboundEditor\.protocol\)/,
  );
});

test('inbounds detail keeps visible share and subscription export actions', () => {
  assert.match(source, /Export Share Links/);
  assert.match(source, /Export Subscription Links/);
  assert.match(source, /@click="exportInboundShareLinks\(selectedInbound\)"/);
  assert.match(source, /@click="exportInboundSubscriptionLinks\(selectedInbound\)"/);
  assert.match(source, /@click="openClientAccessModal\(record\)"/);
  assert.match(source, />\s*Access\s*</);
  assert.match(source, /sharePreviewTitle/);
});

test('inbounds row actions expose direct export entry points', () => {
  assert.match(source, /@click="exportInboundShareLinks\(asInbound\(record\)\)"/);
  assert.match(source, /@click="exportInboundSubscriptionLinks\(asInbound\(record\)\)"/);
  assert.match(source, />\s*导出订阅\s*</);
  assert.match(source, /@click="exportInboundJson\(asInbound\(record\)\)"/);
  assert.match(source, /@click="openInboundQrcode\(asInbound\(record\)\)"/);
  assert.match(source, /@click="confirmResetInboundTraffic\(asInbound\(record\)\)"/);
});

test('inbounds page keeps legacy general action handlers reachable from the header', () => {
  assert.match(source, /exportAllInboundShareLinks/);
  assert.match(source, /exportAllInboundSubscriptionLinks/);
  assert.match(source, /confirmResetAllInboundClientTraffic/);
  assert.match(source, /confirmDeleteAllDepletedClients/);
});

test('inbounds page groups secondary and destructive header actions in a more-actions menu', () => {
  assert.match(source, /<ADropdown/);
  assert.match(source, /<AMenu/);
  assert.match(source, /moreActionsOpen/);
  assert.match(source, /translate\('action\.moreActions'/);
  assert.match(source, /menuDangerActionKeys/);
  assert.match(source, /@click="handleMoreActionClick"/);
  assert.match(source, /const headerPrimaryActionKeys/);
  assert.match(source, /'newInbound'/);
  assert.match(source, /'refresh'/);
  assert.match(source, /'refreshActivity'/);
});

test('inbounds export preview keeps copy and download actions', () => {
  assert.match(source, /@click="downloadSharePreview"/);
  assert.match(source, /sharePreviewFilename/);
  assert.match(source, /downloadText\(sharePreviewFilename\.value, sharePreview\.value\)/);
});

test('inbounds view exposes client access, copy-clients, and clone flows', () => {
  assert.match(source, /Copy Clients/);
  assert.match(source, /@click="openCopyClientsModal\(selectedInbound\)"/);
  assert.match(source, /title="Copy Clients from Other Inbound"/);
  assert.match(source, /@ok="submitCopyClients"/);
  assert.match(source, /confirmCloneInbound\(asInbound\(record\)\)/);
  assert.match(source, />\s*Clone\s*</);
  assert.match(source, /v-model:open="clientAccessModalOpen"/);
  assert.match(source, /clientAccessTitle/);
  assert.match(source, /@click="openClientAccessModal\(record\)"/);
  assert.match(source, />\s*Access\s*</);
  assert.match(source, /client-access-qr-\$\{index\}/);
});

test('inbounds client access does not load retired legacy asset scripts', () => {
  assert.doesNotMatch(source, /\$\{runtime\.basePath\}assets\/qrcode/);
});

test('inbounds view exposes bulk add clients flow', () => {
  assert.match(source, /Bulk Add/);
  assert.match(source, /@click="openBulkAddClientsModal\(selectedInbound\)"/);
  assert.match(source, /title="Bulk Add Clients"/);
  assert.match(source, /@ok="submitBulkAddClients"/);
  assert.match(source, /v-model:value="bulkClientForm\.quantity"/);
  assert.match(source, /v-model:value="bulkClientForm\.emailPrefix"/);
  assert.match(source, /v-model:value="bulkClientForm\.firstIndex"/);
  assert.match(source, /v-model:value="bulkClientForm\.emailPostfix"/);
});
test('hysteria inbound form exposes QUIC Params UDP Hop controls and syncs finalmask JSON', () => {
  assert.match(source, /label="QUIC Params"/);
  assert.match(source, /v-model:checked="streamEditor\.hysteriaQuicParamsEnabled"/);
  assert.match(source, /label="UDP Hop"/);
  assert.match(source, /v-model:checked="streamEditor\.hysteriaUdpHopEnabled"/);
  assert.match(source, /label="Hop Ports"/);
  assert.match(source, /placeholder="40000-45000"/);
  assert.match(source, /label="Hop Interval"/);
  assert.match(source, /placeholder="5-10"/);
  assert.match(source, /hysteriaUdpHopPorts: string/);
  assert.match(source, /hysteriaUdpHopInterval: string/);
  assert.match(source, /const finalmask = objectField\(stream\.finalmask\)/);
  assert.match(source, /const quicParams = objectField\(finalmask\.quicParams\)/);
  assert.match(source, /const udpHop = objectField\(quicParams\.udpHop\)/);
  assert.match(source, /hysteriaUdpHopPorts:\s*stringField\(udpHop\.ports\)/);
  assert.match(source, /applyHysteriaFinalmaskUdpHop/);
  assert.match(source, /const streamWithUdpHop = applyHysteriaFinalmaskUdpHop\(stream, \{/);
  assert.match(source, /Object\.assign\(stream, streamWithUdpHop\)/);
  assert.match(source, /delete stream\.finalmask/);
});
