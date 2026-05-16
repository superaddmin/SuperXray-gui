import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';

const source = readFileSync('frontend/src/views/XrayView.vue', 'utf8');

test('xray view exposes structured editors for outbounds, routing, dns, fakedns, balancers, and reverse', () => {
  for (const title of [
    'Outbounds',
    'Routing Rules',
    'DNS Servers',
    'FakeDNS Pools',
    'Balancers',
    'Reverse',
  ]) {
    assert.match(source, new RegExp(`title="${title.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}"`));
  }
});

test('xray view exposes CRUD actions for structured xray sections', () => {
  for (const action of [
    '@click="openOutboundModal()"',
    '@click="openRoutingRuleModal()"',
    '@click="openDnsServerModal()"',
    '@click="openFakeDnsModal()"',
    '@click="openBalancerModal()"',
    '@click="openReverseModal()"',
    '@ok="submitOutboundModal"',
    '@ok="submitRoutingRuleModal"',
    '@ok="submitDnsServerModal"',
    '@ok="submitFakeDnsModal"',
    '@ok="submitBalancerModal"',
    '@ok="submitReverseModal"',
  ]) {
    assert.match(source, new RegExp(action.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')));
  }
});

test('xray view imports xrayCompat helpers instead of mutating raw JSON inline everywhere', () => {
  assert.match(source, /from '@\/utils\/xrayCompat'/);
  for (const helper of [
    'getOutboundRows',
    'upsertOutbound',
    'deleteOutboundAt',
    'moveArrayItem',
    'getRoutingRuleRows',
    'upsertRoutingRule',
    'deleteRoutingRuleAt',
    'getDnsServerRows',
    'upsertDnsServer',
    'deleteDnsServerAt',
    'getFakeDnsRows',
    'upsertFakeDns',
    'deleteFakeDnsAt',
    'getBalancerRows',
    'upsertBalancer',
    'deleteBalancerAt',
    'getReverseRows',
    'upsertReverse',
    'deleteReverseAt',
  ]) {
    assert.match(source, new RegExp(`\\b${helper}\\b`));
  }
});

test('xray view exposes protocol tools and warp matrix workflows', () => {
  for (const title of ['Protocol Tools', 'WARP Matrix']) {
    assert.match(source, new RegExp(`title="${title.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}"`));
  }

  for (const action of [
    '@click="generateProtocolToolOutput"',
    '@click="copyProtocolToolOutput"',
    '@click="applyProtocolToolOutbound"',
    '@click="loadWarpMatrixConfig"',
    '@click="applyWarpMatrix"',
  ]) {
    assert.match(source, new RegExp(action.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')));
  }

  assert.match(source, /protocolTool\.combo/);
  assert.match(source, /warpMatrixSelected/);
});

test('xray view exposes dns presets and observatory log policy workflows', () => {
  for (const title of ['DNS Presets', 'Runtime Policy', 'Observatory']) {
    assert.match(source, new RegExp(`title="${title.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}"`));
  }

  for (const action of [
    '@click="applyDnsPresetOption',
    '@click="applyDnsPolicyChanges"',
    '@click="applyRuntimePolicyChanges"',
    '@click="applyObservatoryChanges"',
  ]) {
    assert.match(source, new RegExp(action.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')));
  }

  for (const stateField of [
    'dnsPolicyForm.enableDNS',
    'dnsPolicyForm.dnsTag',
    'runtimePolicyForm.freedomStrategy',
    'runtimePolicyForm.logLevel',
    'observatoryForm.observatoryEnable',
    'observatoryForm.burstObservatoryEnable',
  ]) {
    assert.match(source, new RegExp(stateField.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')));
  }
});

test('xray view exposes residential ip pool workflow for ai routing', () => {
  assert.match(source, /title="Residential IP Pool"/);
  for (const action of [
    '@click="openResidentialIpModal()"',
    '@click="applyAiResidentialRoutingChanges"',
    '@click="testResidentialIpOutbound(record.key)"',
    '@ok="submitResidentialIpModal"',
  ]) {
    assert.match(source, new RegExp(action.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')));
  }

  for (const stateField of [
    'residentialIpEditor.tag',
    'residentialIpEditor.server',
    'residentialIpEditor.port',
    'residentialIpEditor.username',
    'residentialIpEditor.password',
  ]) {
    assert.match(source, new RegExp(stateField.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')));
  }
});

test('xray view exposes gateway egress mvp generator with separate listen and manifest hosts', () => {
  assert.match(source, /Gateway Egress MVP/);
  assert.match(source, /gatewayEgressNetwork\.listenHost/);
  assert.match(source, /gatewayEgressNetwork\.manifestHost/);
  assert.match(source, /gatewayEgressNetwork\.strategyLabel/);
  assert.match(source, /mergeGatewayEgressMvpConfig/);
  assert.match(source, /buildGatewayEgressManifestCsv/);
  assert.match(source, /applyGatewayEgressMvp/);
  assert.match(source, /copyGatewayEgressManifest/);
  assert.match(source, /downloadGatewayEgressManifest/);
  assert.doesNotMatch(source, /panel\/api\/egress|egress_groups|egress_nodes|sing-box production/);
});
