import type { JsonObject, JsonValue } from '@/types/api';

function isRoutingObject(value: JsonValue): value is JsonObject {
  return value !== null && typeof value === 'object' && !Array.isArray(value);
}

export function isGeneratedGatewayRule(value: JsonValue): boolean {
  return isRoutingObject(value) && value._gatewayEgressMvp === true;
}

export function ensureBlockedOutbound(outbounds: JsonValue[]): void {
  const tagged = outbounds.filter(isRoutingObject).filter((outbound) => outbound.tag === 'blocked');
  if (tagged.length === 0) {
    outbounds.push({ tag: 'blocked', protocol: 'blackhole' });
  } else if (tagged.length !== 1 || tagged[0].protocol !== 'blackhole') {
    throw new Error('The blocked tag must identify a single blackhole outbound.');
  }
}
