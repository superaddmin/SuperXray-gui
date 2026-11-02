import { expect, test, type Page } from "@playwright/test";

const requiredEnv = [
  "SUPERXRAY_E2E_BASE_URL",
  "SUPERXRAY_E2E_USERNAME",
  "SUPERXRAY_E2E_PASSWORD",
];

const missingEnv = requiredEnv.filter((key) =>
  isUnsetOrPlaceholder(process.env[key]),
);
const mutationDisabled = process.env.SUPERXRAY_E2E_MUTATION !== "1";

test.skip(
  missingEnv.length > 0 || mutationDisabled,
  missingEnv.length > 0
    ? `Missing or placeholder E2E env: ${missingEnv.join(", ")}`
    : "Set SUPERXRAY_E2E_MUTATION=1 to run client toggle mutations",
);

const wireguardTestServerPrivate =
  "MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDE=";
const wireguardTestServerPublic =
  "QUJDREVGR0hJSktMTU5PUFFSU1RVVldYWVo1Njc4OTA=";
const wireguardPeerOnePrivate = "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY=";
const wireguardPeerOnePublic = "emFiY2RlZmdoaWprbG1ub3BxcnN0dXZ3eHl6MTIzNDU=";
const wireguardPeerTwoPrivate = "MTIzNDU2Nzg5MGFiY2RlZmdoaWprbG1ub3BxcnN0dXY=";
const wireguardPeerTwoPublic = "NTQzMjEwemFiY2RlZmdoaWprbG1ub3BxcnN0dXYxMjM=";
const csrfTokens = new WeakMap<Page, string>();

type APIInboundSnapshot = {
  id: number;
  protocol?: string;
  remark?: string;
  settings?: string | Record<string, unknown>;
};

type JsonResponse = {
  status(): number;
  json(): Promise<unknown>;
};

type LegacyResponseBody = {
  success?: boolean;
  msg?: string;
  obj?: unknown;
};

function isUnsetOrPlaceholder(value: string | undefined): boolean {
  return !value || /<[^>]+>/.test(value);
}

function appUrl(path = ""): string {
  const base = new URL(process.env.SUPERXRAY_E2E_BASE_URL as string);
  if (!base.pathname.endsWith("/")) {
    base.pathname = `${base.pathname}/`;
  }
  return new URL(path.replace(/^\//, ""), base).toString();
}

async function readCsrfToken(page: Page): Promise<string> {
  return page.evaluate(() => {
    const globalWindow = window as typeof window & {
      __SUPERXRAY_CSRF_TOKEN__?: string;
      __SUPERXRAY_UI_CONFIG__?: { csrfToken?: string };
    };
    return (
      globalWindow.__SUPERXRAY_UI_CONFIG__?.csrfToken ||
      globalWindow.__SUPERXRAY_CSRF_TOKEN__ ||
      ""
    );
  });
}

function requestHeaders(page: Page): Record<string, string> {
  const token = csrfTokens.get(page);
  return {
    Origin: new URL(page.url()).origin,
    ...(token ? { "X-CSRF-Token": token } : {}),
  };
}

async function expectJsonSuccess(response: JsonResponse) {
  expect(response.status()).toBe(200);
  const body = (await response.json()) as LegacyResponseBody;
  expect(
    body.success,
    body.msg ?? "API response should be successful",
  ).not.toBe(false);
  return body;
}

async function login(page: Page) {
  await page.goto(appUrl("panel/login"));
  await page
    .getByLabel("Username", { exact: true })
    .fill(process.env.SUPERXRAY_E2E_USERNAME as string);
  await page
    .getByLabel("Password", { exact: true })
    .fill(process.env.SUPERXRAY_E2E_PASSWORD as string);

  const twoFactorInput = page.getByLabel("Two-factor code", { exact: true });
  if (
    (await twoFactorInput.count()) > 0 &&
    (await twoFactorInput.isVisible())
  ) {
    const code = process.env.SUPERXRAY_E2E_TOTP;
    test.skip(
      !code,
      "SUPERXRAY_E2E_TOTP is required when two-factor auth is enabled",
    );
    await twoFactorInput.fill(code as string);
  }

  await Promise.all([
    page.waitForURL(/\/panel\/?$/),
    page.getByRole("button", { name: "Sign in", exact: true }).click(),
  ]);
  await expect(page.locator("#app")).toBeVisible();
  csrfTokens.set(page, await readCsrfToken(page));
}

async function postForm(
  page: Page,
  path: string,
  form: Record<string, string | number | boolean>,
) {
  return expectJsonSuccess(
    await page.request.post(appUrl(path), {
      form,
      headers: requestHeaders(page),
    }),
  );
}

async function getInbound(
  page: Page,
  inboundId: number,
): Promise<APIInboundSnapshot> {
  const body = await expectJsonSuccess(
    await page.request.get(appUrl("panel/api/inbounds/list"), {
      headers: requestHeaders(page),
    }),
  );
  const inbound = (body.obj as APIInboundSnapshot[] | undefined)?.find(
    (item) => item.id === inboundId,
  );
  expect(inbound, `API inbound ${inboundId} should exist`).toBeTruthy();
  return inbound as APIInboundSnapshot;
}

function parseSettings(inbound: APIInboundSnapshot): Record<string, unknown> {
  if (typeof inbound.settings === "string") {
    return JSON.parse(inbound.settings) as Record<string, unknown>;
  }
  return inbound.settings ?? {};
}

function findClientRow(page: Page, email: string) {
  return page.locator(".ant-table-tbody tr").filter({ hasText: email }).first();
}

async function openClients(page: Page) {
  await page.goto(appUrl("panel/clients"));
  await expect(
    page.getByRole("heading", { name: "Clients", exact: true }),
  ).toBeVisible();
  await expect(page.locator(".ant-table")).toBeVisible();
}

function vlessSettings(email: string, id: string) {
  return {
    clients: [
      {
        id,
        flow: "",
        email,
        limitIp: 0,
        totalGB: 0,
        expiryTime: 0,
        enable: true,
        tgId: 0,
        subId: `toggle-${email}`,
        comment: "client toggle e2e",
        reset: 0,
      },
    ],
    decryption: "none",
    encryption: "none",
  };
}

function wireguardSettings(firstEmail: string, secondEmail: string) {
  return {
    mtu: 1420,
    secretKey: wireguardTestServerPrivate,
    pubKey: wireguardTestServerPublic,
    peers: [
      {
        privateKey: wireguardPeerOnePrivate,
        publicKey: wireguardPeerOnePublic,
        allowedIPs: ["10.0.0.2/32"],
        keepAlive: 0,
        email: firstEmail,
        enable: true,
        subId: "toggle-peer-one",
      },
      {
        privateKey: wireguardPeerTwoPrivate,
        publicKey: wireguardPeerTwoPublic,
        allowedIPs: ["10.0.0.3/32"],
        keepAlive: 0,
        email: secondEmail,
        enable: true,
        subId: "toggle-peer-two",
      },
    ],
    noKernelTun: false,
  };
}

async function addInbound(
  page: Page,
  input: {
    remark: string;
    port: number;
    protocol: string;
    settings: Record<string, unknown>;
  },
): Promise<number> {
  const body = await postForm(page, "panel/api/inbounds/add", {
    up: 0,
    down: 0,
    total: 0,
    remark: input.remark,
    enable: false,
    expiryTime: 0,
    trafficReset: "never",
    lastTrafficResetTime: 0,
    listen: "127.0.0.1",
    port: input.port,
    protocol: input.protocol,
    settings: JSON.stringify(input.settings),
    streamSettings: JSON.stringify(
      input.protocol === "vless"
        ? {
            network: "tcp",
            security: "none",
            tcpSettings: { header: { type: "none" } },
          }
        : {},
    ),
    sniffing: JSON.stringify({ enabled: false, destOverride: [] }),
  });
  const inbound = body.obj as { id?: number } | undefined;
  expect(inbound?.id).toBeGreaterThan(0);
  return inbound?.id as number;
}

test.describe("Clients overview client lifecycle controls", () => {
  test.beforeEach(async ({ page }) => {
    await page.addInitScript(() => {
      window.localStorage.setItem("superxray.locale", "en-US");
    });
  });

  test("toggles a VLESS client through the real API, recovers from failure, and prevents duplicate clicks", async ({
    page,
  }) => {
    await login(page);
    const stamp = Date.now();
    const email = `e2e-toggle-vless-${stamp}@example.invalid`;
    const clientId = crypto.randomUUID();
    const remark = `e2e-toggle-vless-${stamp}`;
    let inboundId: number | undefined;

    try {
      inboundId = await addInbound(page, {
        remark,
        port: 45_000 + Math.floor(Math.random() * 1_000),
        protocol: "vless",
        settings: vlessSettings(email, clientId),
      });

      await openClients(page);
      const row = findClientRow(page, email);
      await expect(row).toBeVisible({ timeout: 15_000 });
      const stopButton = row.getByRole("button", { name: /^Stop client / });
      await expect(stopButton).toBeVisible();

      let failNext = true;
      await page.route(
        "**/panel/api/inbounds/updateClient/**",
        async (route) => {
          if (failNext) {
            failNext = false;
            await route.fulfill({
              status: 500,
              contentType: "application/json",
              body: JSON.stringify({
                success: false,
                msg: "synthetic toggle failure",
                obj: null,
              }),
            });
            return;
          }
          await route.continue();
        },
      );

      await stopButton.click();
      await expect(page.locator(".ant-alert")).toContainText(
        "synthetic toggle failure",
      );
      await expect(stopButton).toBeEnabled();
      await page.unroute("**/panel/api/inbounds/updateClient/**");

      let forwardedRequests = 0;
      await page.route(
        "**/panel/api/inbounds/updateClient/**",
        async (route) => {
          forwardedRequests += 1;
          await new Promise((resolve) => setTimeout(resolve, 700));
          await route.continue();
        },
      );
      const updateResponse = page.waitForResponse(
        (response) =>
          response.url().includes("/panel/api/inbounds/updateClient/") &&
          response.request().method() === "POST",
      );
      await stopButton.click();
      await expect(stopButton).toBeDisabled();
      await stopButton.click({ force: true }).catch(() => undefined);
      await expectJsonSuccess(await updateResponse);
      await page.unroute("**/panel/api/inbounds/updateClient/**");
      expect(forwardedRequests).toBe(1);
      await expect(
        row.getByRole("button", { name: /^Start client / }),
      ).toBeVisible();

      const stoppedInbound = await getInbound(page, inboundId);
      const stoppedClient = (
        parseSettings(stoppedInbound).clients as Array<Record<string, unknown>>
      ).find((client) => client.email === email);
      expect(stoppedClient?.enable).toBe(false);

      const startResponse = page.waitForResponse(
        (response) =>
          response.url().includes("/panel/api/inbounds/updateClient/") &&
          response.request().method() === "POST",
      );
      await row.getByRole("button", { name: /^Start client / }).click();
      await expectJsonSuccess(await startResponse);
      const restartedInbound = await getInbound(page, inboundId);
      const restartedClient = (
        parseSettings(restartedInbound).clients as Array<
          Record<string, unknown>
        >
      ).find((client) => client.email === email);
      expect(restartedClient?.enable).toBe(true);
    } finally {
      if (inboundId) {
        await postForm(page, `panel/api/inbounds/del/${inboundId}`, {});
      }
    }
  });

  test("toggles one WireGuard peer while preserving the other peer", async ({
    page,
  }) => {
    await login(page);
    const stamp = Date.now();
    const firstEmail = `e2e-toggle-peer-one-${stamp}@example.invalid`;
    const secondEmail = `e2e-toggle-peer-two-${stamp}@example.invalid`;
    const remark = `e2e-toggle-wireguard-${stamp}`;
    let inboundId: number | undefined;

    try {
      inboundId = await addInbound(page, {
        remark,
        port: 46_000 + Math.floor(Math.random() * 1_000),
        protocol: "wireguard",
        settings: wireguardSettings(firstEmail, secondEmail),
      });

      await openClients(page);
      const firstRow = findClientRow(page, firstEmail);
      const secondRow = findClientRow(page, secondEmail);
      await expect(firstRow).toBeVisible({ timeout: 15_000 });
      await expect(secondRow).toBeVisible({ timeout: 15_000 });

      const updateResponse = page.waitForResponse(
        (response) =>
          response.url().includes(`/panel/api/inbounds/update/${inboundId}`) &&
          response.request().method() === "POST",
      );
      await firstRow.getByRole("button", { name: /^Stop client / }).click();
      await expectJsonSuccess(await updateResponse);
      await expect(
        firstRow.getByRole("button", { name: /^Start client / }),
      ).toBeVisible();
      await expect(
        secondRow.getByRole("button", { name: /^Stop client / }),
      ).toBeVisible();

      const inbound = await getInbound(page, inboundId);
      const peers = parseSettings(inbound).peers as Array<
        Record<string, unknown>
      >;
      expect(peers).toHaveLength(2);
      expect(peers.find((peer) => peer.email === firstEmail)?.enable).toBe(
        false,
      );
      expect(peers.find((peer) => peer.email === secondEmail)?.enable).toBe(
        true,
      );

      const restartResponse = page.waitForResponse(
        (response) =>
          response.url().includes(`/panel/api/inbounds/update/${inboundId}`) &&
          response.request().method() === "POST",
      );
      await firstRow.getByRole("button", { name: /^Start client / }).click();
      await expectJsonSuccess(await restartResponse);
    } finally {
      if (inboundId) {
        await postForm(page, `panel/api/inbounds/del/${inboundId}`, {});
      }
    }
  });
});
