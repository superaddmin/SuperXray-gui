export interface RealityServerSettingsInput {
  target?: string;
  serverNames?: string | string[];
  privateKey?: string;
  shortIds?: string | string[];
  publicKey?: string;
  spiderX?: string;
  mldsa65Seed?: string;
  mldsa65Verify?: string;
}

export interface RealityServerSettingsNormalized {
  target: string;
  serverNames: string[];
  privateKey: string;
  shortIds: string[];
  publicKey: string;
  spiderX: string;
  mldsa65Seed: string;
  mldsa65Verify: string;
}

const DEFAULT_REALITY_TARGET = {
  target: 'www.apple.com:443',
  serverName: 'www.apple.com',
};

function cleanString(value: unknown): string {
  return typeof value === 'string' ? value.trim() : '';
}

function cleanList(value: string | string[] | undefined): string[] {
  if (Array.isArray(value)) {
    return value.map((item) => item.trim()).filter(Boolean);
  }
  return cleanString(value)
    .split(/[\n,]+/)
    .map((item) => item.trim())
    .filter(Boolean);
}

export function normalizeRealityServerSettings(
  input: RealityServerSettingsInput,
): RealityServerSettingsNormalized {
  const target = cleanString(input.target) || DEFAULT_REALITY_TARGET.target;
  const serverNames = cleanList(input.serverNames);

  return {
    target,
    serverNames: serverNames.length > 0 ? serverNames : [DEFAULT_REALITY_TARGET.serverName],
    privateKey: cleanString(input.privateKey),
    shortIds: cleanList(input.shortIds),
    publicKey: cleanString(input.publicKey),
    spiderX: cleanString(input.spiderX) || '/',
    mldsa65Seed: cleanString(input.mldsa65Seed),
    mldsa65Verify: cleanString(input.mldsa65Verify),
  };
}

export function validateRealityServerSettings(input: RealityServerSettingsInput): string {
  const normalized = normalizeRealityServerSettings(input);
  if (!normalized.privateKey) {
    return 'Reality private key is required';
  }
  if (!normalized.shortIds.length) {
    return 'Reality short IDs are required';
  }
  return '';
}
