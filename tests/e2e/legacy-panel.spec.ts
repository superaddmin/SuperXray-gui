import {
  expect,
  test,
  type APIResponse,
  type Locator,
  type Page,
} from "@playwright/test";

const requiredEnv = [
  "SUPERXRAY_E2E_BASE_URL",
  "SUPERXRAY_E2E_USERNAME",
  "SUPERXRAY_E2E_PASSWORD",
];

const missingEnv = requiredEnv.filter((key) =>
  isUnsetOrPlaceholder(process.env[key]),
);

const wireguardTestServerPrivate =
  "MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDE=";
const wireguardTestServerPublic =
  "QUJDREVGR0hJSktMTU5PUFFSU1RVVldYWVo1Njc4OTA=";
const wireguardTestPeerPrivate = "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY=";
const wireguardTestPeerPublic = "emFiY2RlZmdoaWprbG1ub3BxcnN0dXZ3eHl6MTIzNDU=";
const csrfTokens = new WeakMap<Page, string>();

test.skip(
  missingEnv.length > 0,
  `Missing or placeholder E2E env: ${missingEnv.join(", ")}`,
);

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

function originHeader(page: Page): Record<string, string> {
  const token = csrfTokens.get(page);
  return {
    Origin: new URL(page.url()).origin,
    ...(token ? { "X-CSRF-Token": token } : {}),
  };
}

async function expectJsonSuccess(response: APIResponse) {
  expect(response.status()).toBe(200);
  const body = await response.json();
  expect(
    body.success,
    body.msg ?? "legacy response should be successful",
  ).not.toBe(false);
  return body;
}

async function expectJsonFailure(response: APIResponse) {
  expect(response.status()).toBe(200);
  const body = await response.json();
  expect(body.success).toBe(false);
  return body;
}

async function login(page: Page) {
  await loginWith(
    page,
    process.env.SUPERXRAY_E2E_USERNAME as string,
    process.env.SUPERXRAY_E2E_PASSWORD as string,
  );
}

async function loginWith(page: Page, username: string, password: string) {
  await page.goto(appUrl());

  if (!/\/panel\/?$/.test(page.url())) {
    const usernameInput = page.locator('input[name="username"]');
    await expect(usernameInput).toBeVisible();
    await usernameInput.fill(username);
    await page.locator('input[name="password"]').fill(password);

    const twoFactorInput = page.locator('input[name="twoFactorCode"]');
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
      page.locator('button[type="submit"]').click(),
    ]);
  }

  await expect(page).toHaveURL(/\/panel\/?$/);
  await expect(page.locator("#app")).toBeVisible();
  csrfTokens.set(page, await readCsrfToken(page));
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

async function postForm(
  page: Page,
  path: string,
  form: Record<string, string | number | boolean>,
) {
  return expectJsonSuccess(
    await page.request.post(appUrl(path), {
      form,
      headers: originHeader(page),
    }),
  );
}

async function fillFormItemInput(
  dialog: Locator,
  label: string,
  value: string,
) {
  await dialog
    .locator(".ant-form-item")
    .filter({ hasText: label })
    .locator("input")
    .first()
    .fill(value);
}

async function setFormItemSwitch(
  dialog: Locator,
  label: string,
  checked: boolean,
) {
  const switchControl = dialog
    .locator(".ant-form-item")
    .filter({ hasText: label })
    .getByRole("switch")
    .first();
  const current = (await switchControl.getAttribute("aria-checked")) === "true";
  if (current !== checked) {
    await switchControl.click();
  }
}

async function selectFormItemOption(
  page: Page,
  root: Locator,
  label: string,
  option: string,
) {
  await root
    .locator(".ant-form-item")
    .filter({ hasText: label })
    .locator(".ant-select-selector")
    .first()
    .click();
  const dropdown = page.locator(
    ".ant-select-dropdown:not(.ant-select-dropdown-hidden)",
  );
  await dropdown.getByText(option, { exact: true }).click();
}

async function fillFormItemTextarea(
  root: Locator,
  label: string,
  value: string,
) {
  await root
    .locator(".ant-form-item")
    .filter({ hasText: label })
    .locator("textarea")
    .first()
    .fill(value);
}

async function waitForLegacyInbounds(
  page: Page,
  expected: Array<{ id: number; protocol: string; remark: string }>,
) {
  await page.goto(appUrl("panel/legacy/inbounds"));
  await expect(page.locator("#app")).toBeVisible();
  for (const record of expected) {
    await expect(page.getByText(record.remark, { exact: true })).toBeVisible({
      timeout: 15_000,
    });
  }
}

async function expectLegacySettingsValues(
  page: Page,
  values: { announce: string; title: string },
) {
  await page.goto(appUrl("panel/legacy/settings"));
  await expect(page.locator("#app")).toBeVisible();
  await page.getByRole("tab", { name: "Subscription", exact: true }).click();
  await page.getByRole("button", { name: "Information" }).click();
  await page.waitForFunction(
    (expected) => {
      const formValues = Array.from(
        document.querySelectorAll("input, textarea"),
      ).map(
        (element) => (element as HTMLInputElement | HTMLTextAreaElement).value,
      );
      return (
        formValues.includes(expected.title) &&
        formValues.includes(expected.announce)
      );
    },
    values,
    { timeout: 15_000 },
  );
}

type LegacyInboundSnapshot = {
  id: number;
  enable?: boolean;
  listen?: string;
  port?: number;
  protocol?: string;
  remark?: string;
  settings?: string | Record<string, unknown>;
};

async function openNewUiInbounds(page: Page) {
  await page.goto(appUrl("panel/inbounds"));
  await expect(
    page.getByRole("heading", { name: "Inbounds", exact: true }),
  ).toBeVisible();
  await expect(page.locator(".ant-table")).toBeVisible();
}

function newUiInboundRow(page: Page, remark: string): Locator {
  return page
    .locator(".ant-table-tbody tr")
    .filter({ hasText: remark })
    .first();
}

async function openNewUiInboundDetail(page: Page, remark: string) {
  await openNewUiInbounds(page);
  const row = newUiInboundRow(page, remark);
  await expect(row).toBeVisible({ timeout: 15_000 });
  await row.getByRole("button", { name: "Details" }).click();
  const drawer = page.locator(".ant-drawer").filter({ hasText: "Clients" });
  await expect(drawer).toBeVisible();
  return drawer;
}

function newUiDrawerClientRow(drawer: Locator, email: string): Locator {
  const clientsPanel = drawer
    .locator(
      'xpath=.//*[contains(concat(" ", normalize-space(@class), " "), " drawer-panel ")][.//h2[normalize-space()="Clients"]]',
    )
    .first();
  return clientsPanel
    .locator(".ant-table-tbody tr")
    .filter({ hasText: email })
    .first();
}

async function getLegacyInbound(
  page: Page,
  inboundId: number,
): Promise<LegacyInboundSnapshot> {
  const list = await expectJsonSuccess(
    await page.request.get(appUrl("panel/api/inbounds/list")),
  );
  const inbound = (list.obj as LegacyInboundSnapshot[]).find(
    (item) => item.id === inboundId,
  );
  expect(inbound, `legacy inbound ${inboundId} should exist`).toBeTruthy();
  return inbound as LegacyInboundSnapshot;
}

function parseLegacySettings(
  inbound: LegacyInboundSnapshot,
): Record<string, unknown> {
  if (typeof inbound.settings === "string") {
    return JSON.parse(inbound.settings) as Record<string, unknown>;
  }
  return inbound.settings ?? {};
}

function legacySettingsArray(
  inbound: LegacyInboundSnapshot,
  key: "clients" | "peers",
): Array<Record<string, unknown>> {
  const value = parseLegacySettings(inbound)[key];
  return Array.isArray(value)
    ? value.filter(
        (item): item is Record<string, unknown> =>
          item !== null && typeof item === "object" && !Array.isArray(item),
      )
    : [];
}

async function expectLegacyExpandedClientText(
  page: Page,
  inboundRemark: string,
  clientText: string,
) {
  await page.goto(appUrl("panel/legacy/inbounds"));
  await expect(page.locator("#app")).toBeVisible();
  const inboundRow = page
    .locator("tr")
    .filter({ hasText: inboundRemark })
    .first();
  await expect(inboundRow).toBeVisible({ timeout: 15_000 });
  await inboundRow.locator(".ant-table-row-expand-icon").first().click({
    force: true,
  });
  await expect(page.getByText(clientText, { exact: true })).toBeVisible({
    timeout: 15_000,
  });
}

function expectSQLiteBackup(buffer: Buffer) {
  expect(buffer.length).toBeGreaterThan(16);
  expect(buffer.subarray(0, 16).toString("ascii")).toBe("SQLite format 3\0");
}

function pseudoSQLiteBuffer(): Buffer {
  return Buffer.concat([
    Buffer.from("SQLite format 3\0", "ascii"),
    Buffer.from("phase9-not-a-real-sqlite-database"),
  ]);
}

function cspDirective(csp: string | undefined, name: string): string {
  return (
    csp
      ?.split(";")
      .map((directive) => directive.trim())
      .find((directive) => directive.startsWith(`${name} `)) ?? ""
  );
}

test.describe("legacy Xray UI parity baseline", () => {
  test("can log in through the new Vue UI login page", async ({ page }) => {
    await page.goto(appUrl("panel/login"));

    await expect(
      page.getByRole("heading", { name: "Welcome back", exact: true }),
    ).toBeVisible();
    await page
      .getByLabel("Username")
      .fill(process.env.SUPERXRAY_E2E_USERNAME as string);
    await page
      .getByLabel("Password")
      .fill(process.env.SUPERXRAY_E2E_PASSWORD as string);

    const twoFactorInput = page.getByLabel("Two-factor code");
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
      page.getByRole("button", { name: "Sign in" }).click(),
    ]);

    await expect(
      page.getByRole("heading", { name: "Dashboard", exact: true }),
    ).toBeVisible();
    csrfTokens.set(page, await readCsrfToken(page));
  });

  test("can open new UI geo maintenance controls", async ({ page }) => {
    await login(page);
    await page.goto(appUrl("panel/dashboard"));

    await expect(page.getByText("Geo Maintenance")).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Update geoip.dat" }),
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Update geosite.dat" }),
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Add Resource" }),
    ).toBeVisible();
    await expect(page.getByText("Custom Geo")).toBeVisible();
  });

  test("can open new UI Xray outbound tools", async ({ page }) => {
    await login(page);
    await page.goto(appUrl("panel/xray"));

    await expect(page.getByText("Outbound Tools")).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Refresh Traffic" }),
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Test First Outbound" }),
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Reset All Traffic" }),
    ).toBeVisible();
    await expect(page.getByText("Warp / Nord")).toBeVisible();
  });

  test("can open new UI inbound import and batch controls", async ({
    page,
  }) => {
    await login(page);
    await page.goto(appUrl("panel/inbounds"));

    await expect(
      page.getByRole("button", { name: "Import JSON" }),
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Reset All Traffic" }),
    ).toBeVisible();
  });

  test("can open new UI online and IP management controls", async ({
    page,
  }) => {
    test.skip(
      process.env.SUPERXRAY_E2E_MUTATION !== "1",
      "Set SUPERXRAY_E2E_MUTATION=1 to run online and IP management controls",
    );

    await login(page);

    const stamp = Date.now();
    const remark = `e2e-activity-vless-${stamp}`;
    const email = `e2e-activity-${stamp}@example.invalid`;
    let inboundId: number | undefined;

    try {
      const added = await postForm(page, "panel/api/inbounds/add", {
        up: 0,
        down: 0,
        total: 0,
        remark,
        enable: false,
        expiryTime: 0,
        trafficReset: "never",
        lastTrafficResetTime: 0,
        listen: "127.0.0.1",
        port: 49_000 + Math.floor(Math.random() * 1000),
        protocol: "vless",
        settings: JSON.stringify({
          clients: [
            {
              id: crypto.randomUUID(),
              flow: "",
              email,
              limitIp: 0,
              totalGB: 0,
              expiryTime: 0,
              enable: true,
              tgId: 0,
              subId: `activity${stamp}`,
              comment: "activity fixture",
              reset: 0,
            },
          ],
          decryption: "none",
          encryption: "none",
        }),
        streamSettings: JSON.stringify({
          network: "tcp",
          security: "none",
          tcpSettings: { acceptProxyProtocol: false, header: { type: "none" } },
        }),
        sniffing: JSON.stringify({ enabled: false, destOverride: [] }),
      });
      inboundId = added.obj.id;

      await page.goto(appUrl("panel/inbounds"));

      await expect(page.getByText("Online Clients")).toBeVisible();
      await expect(
        page.getByRole("button", { name: "Refresh Activity" }),
      ).toBeVisible();
      const row = page.locator(".ant-table-row").filter({ hasText: remark });
      await expect(row).toBeVisible();
      await row.getByRole("button", { name: "Details" }).click();
      await expect(page.getByText("Online / IP Management")).toBeVisible();
      await expect(
        page.getByRole("button", { name: "View IPs" }),
      ).toBeVisible();
      await expect(
        page.getByRole("button", { name: "Clear IPs" }),
      ).toBeVisible();
    } finally {
      if (inboundId) {
        await postForm(page, `panel/api/inbounds/del/${inboundId}`, {});
      }
    }
  });

  test("can open new UI two-factor setup and subscription public links", async ({
    page,
  }) => {
    await login(page);
    await page.goto(appUrl("panel/settings"));

    await page.getByRole("tab", { name: "Security", exact: true }).click();
    await expect(page.getByText("Two Factor Setup")).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Generate Token" }),
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Disable Two Factor" }),
    ).toBeVisible();
    await expect(page.getByLabel("Two-factor setup URI")).toBeVisible();

    await page.getByRole("tab", { name: "Subscription", exact: true }).click();
    await expect(page.getByText("Subscription Public Links")).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Copy Links" }),
    ).toBeVisible();
    await expect(page.getByLabel("Subscription public links")).toBeVisible();
  });

  test("can log in, navigate core legacy pages, and read status/log APIs", async ({
    page,
  }) => {
    await login(page);

    const status = await expectJsonSuccess(
      await page.request.get(appUrl("panel/api/server/status")),
    );
    expect(status).toHaveProperty("obj");

    await page.goto(appUrl("panel/legacy/inbounds"));
    await expect(page.locator("#app")).toBeVisible();
    await expectJsonSuccess(
      await page.request.get(appUrl("panel/api/inbounds/list")),
    );

    await page.goto(appUrl("panel/legacy/xray"));
    await expect(page.locator("#app")).toBeVisible();
    await expectJsonSuccess(
      await page.request.post(appUrl("panel/xray/"), {
        headers: originHeader(page),
      }),
    );

    await page.goto(appUrl("panel/legacy/settings"));
    await expect(page.locator("#app")).toBeVisible();
    await expectJsonSuccess(
      await page.request.post(appUrl("panel/setting/all"), {
        headers: originHeader(page),
      }),
    );

    await postForm(page, "panel/api/server/logs/20", {
      level: "all",
      syslog: false,
    });
    await postForm(page, "panel/api/server/xraylogs/20", {
      filter: "",
      showDirect: true,
      showBlocked: true,
      showProxy: true,
    });
  });

  test("can open embedded Vue UI and Phase 6 inbound management shell", async ({
    page,
  }) => {
    await login(page);

    const response = await page.goto(appUrl("panel/"));
    expect(response?.status()).toBe(200);
    const csp = response?.headers()["content-security-policy"];
    expect(cspDirective(csp, "script-src")).not.toContain("'unsafe-eval'");
    expect(cspDirective(csp, "script-src")).not.toContain("'unsafe-inline'");
    expect(cspDirective(csp, "style-src")).not.toContain("'unsafe-inline'");
    expect(cspDirective(csp, "style-src")).toContain("'nonce-");
    expect(cspDirective(csp, "style-src-attr")).toBe("style-src-attr 'none'");

    const legacyResponse = await page.goto(appUrl("panel/legacy/"));
    expect(
      cspDirective(
        legacyResponse?.headers()["content-security-policy"],
        "script-src",
      ),
    ).toContain("'unsafe-eval'");

    await page.goto(appUrl("panel/"));
    await expect(page.getByRole("link", { name: "SuperXray" })).toBeVisible();
    await expect(
      page.getByRole("heading", { name: "Dashboard", exact: true }),
    ).toBeVisible();

    const runtimeConfig = await page.evaluate(() => {
      return (
        window as typeof window & {
          __SUPERXRAY_UI_CONFIG__?: {
            csrfToken?: string;
            cspNonce?: string;
            uiBasePath?: string;
          };
        }
      ).__SUPERXRAY_UI_CONFIG__;
    });
    expect(runtimeConfig?.uiBasePath).toMatch(/\/panel\/$/);
    expect(runtimeConfig?.cspNonce).toBeTruthy();
    expect(runtimeConfig?.csrfToken).toBeTruthy();
    const styleNonceState = await page.evaluate(() => {
      const runtimeNonce = (
        window as typeof window & {
          __SUPERXRAY_UI_CONFIG__?: { cspNonce?: string };
        }
      ).__SUPERXRAY_UI_CONFIG__?.cspNonce;
      const styles = Array.from(document.querySelectorAll("style"));
      return {
        nonceStyleCount: styles.filter(
          (style) => style.nonce || style.getAttribute("nonce"),
        ).length,
        runtimeNonce,
        styleCount: styles.length,
      };
    });
    expect(styleNonceState.runtimeNonce).toBe(runtimeConfig?.cspNonce);
    expect(styleNonceState.styleCount).toBeGreaterThan(0);
    expect(styleNonceState.nonceStyleCount).toBe(styleNonceState.styleCount);

    const compatibilityResponse = await page.goto(appUrl("panel/ui/"));
    expect(compatibilityResponse?.status()).toBe(200);
    const compatibilityRuntimeConfig = await page.evaluate(() => {
      return (
        window as typeof window & {
          __SUPERXRAY_UI_CONFIG__?: { uiBasePath?: string };
        }
      ).__SUPERXRAY_UI_CONFIG__;
    });
    expect(compatibilityRuntimeConfig?.uiBasePath).toMatch(/\/panel\/ui\/$/);

    await page.goto(appUrl("panel/logs"));
    await expect(
      page.getByRole("heading", { name: "Logs", exact: true }),
    ).toBeVisible();
    await expect(page.getByText("Panel")).toBeVisible();

    await page.goto(appUrl("panel/xray"));
    await expect(
      page.getByRole("heading", { name: "Xray", exact: true }),
    ).toBeVisible();
    await expect(page.getByText("Xray Runtime Control")).toBeVisible();
    await expect(page.getByText("Xray Version Management")).toBeVisible();
    await expect(page.locator(".code-preview")).toBeVisible();
    await expect(page.locator(".json-editor")).toBeVisible();

    await page.goto(appUrl("panel/inbounds"));
    await expect(
      page.getByRole("heading", { name: "Inbounds", exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: /New Inbound/ }),
    ).toBeVisible();
    await expect(page.getByText("All protocols")).toBeVisible();
    await expect(page.locator(".ant-table")).toBeVisible();

    await page.getByRole("button", { name: /New Inbound/ }).click();
    const newInboundDialog = page.getByRole("dialog", { name: "New Inbound" });
    await expect(newInboundDialog).toBeVisible();
    await newInboundDialog.locator(".ant-select-selector").first().click();
    const protocolDropdown = page.locator(
      ".ant-select-dropdown:not(.ant-select-dropdown-hidden)",
    );
    await expect(
      protocolDropdown.getByText("Trojan", { exact: true }),
    ).toBeVisible();
    await expect(
      protocolDropdown.getByText("Shadowsocks", { exact: true }),
    ).toBeVisible();
    await expect(
      protocolDropdown.getByText("Hysteria2", { exact: true }),
    ).toBeVisible();
    await expect(
      protocolDropdown.getByText("WireGuard", { exact: true }),
    ).toBeVisible();
    await expect(
      newInboundDialog.getByText("Stream Settings Form"),
    ).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(newInboundDialog).toBeHidden();
  });

  test("legacy UI can read protocol matrix inbounds created from the new Vue UI", async ({
    page,
  }) => {
    test.skip(
      process.env.SUPERXRAY_E2E_MUTATION !== "1",
      "Set SUPERXRAY_E2E_MUTATION=1 to run new UI to legacy compatibility matrix",
    );

    await login(page);

    const stamp = Date.now();
    const protocols = [
      { label: "VMess", protocol: "vmess", slug: "vmess" },
      { label: "VLESS", protocol: "vless", slug: "vless" },
      { label: "Trojan", protocol: "trojan", slug: "trojan" },
      {
        label: "Shadowsocks",
        protocol: "shadowsocks",
        slug: "shadowsocks",
      },
      {
        label: "Hysteria2",
        protocol: "hysteria",
        slug: "hysteria2",
        configure: async (dialog: Locator) => {
          await fillFormItemInput(dialog, "Certificate File", "/tmp/e2e.crt");
          await fillFormItemInput(dialog, "Key File", "/tmp/e2e.key");
          await fillFormItemInput(dialog, "Hysteria2 Auth", `hy2-${stamp}`);
        },
      },
      { label: "WireGuard", protocol: "wireguard", slug: "wireguard" },
    ];
    const created: Array<{ id: number; protocol: string; remark: string }> = [];

    try {
      for (const [index, protocol] of protocols.entries()) {
        const remark = `e2e-new-ui-${protocol.slug}-${stamp}`;
        const port = 42_000 + index + Math.floor(Math.random() * 2000);

        await page.goto(appUrl("panel/inbounds"));
        await expect(
          page.getByRole("heading", { name: "Inbounds", exact: true }),
        ).toBeVisible();

        await page.getByRole("button", { name: /New Inbound/ }).click();
        const dialog = page.getByRole("dialog", { name: "New Inbound" });
        await expect(dialog).toBeVisible();

        await selectFormItemOption(page, dialog, "Protocol", protocol.label);
        await fillFormItemInput(dialog, "Remark", remark);
        await fillFormItemInput(dialog, "Listen", "127.0.0.1");
        await fillFormItemInput(dialog, "Port", String(port));
        await setFormItemSwitch(dialog, "Enable", false);
        if (protocol.configure) {
          await protocol.configure(dialog);
        }

        const addResponsePromise = page.waitForResponse(
          (response) =>
            response.url().includes("/panel/api/inbounds/add") &&
            response.request().method() === "POST",
        );
        await dialog.getByRole("button", { name: "OK" }).click();
        const addResponse = await addResponsePromise;
        const addBody = await expectJsonSuccess(addResponse);
        const inboundId = addBody.obj.id;
        expect(inboundId).toBeGreaterThan(0);
        created.push({
          id: inboundId,
          protocol: protocol.protocol,
          remark,
        });

        await expect(page.getByText(remark)).toBeVisible();
      }

      await waitForLegacyInbounds(page, created);
      const list = await expectJsonSuccess(
        await page.request.get(appUrl("panel/api/inbounds/list")),
      );
      for (const expected of created) {
        const inbound = list.obj.find(
          (item: { id: number; remark?: string; protocol?: string }) =>
            item.id === expected.id,
        );
        expect(inbound?.remark).toBe(expected.remark);
        expect(inbound?.protocol).toBe(expected.protocol);
      }
    } finally {
      for (const inbound of created.reverse()) {
        await postForm(page, `panel/api/inbounds/del/${inbound.id}`, {});
      }
    }
  });

  test("legacy settings UI can read subscription settings saved from the new Vue UI", async ({
    page,
  }) => {
    test.skip(
      process.env.SUPERXRAY_E2E_MUTATION !== "1",
      "Set SUPERXRAY_E2E_MUTATION=1 to run new UI settings compatibility baseline",
    );

    await login(page);

    let originalSettings: Record<string, string | number | boolean> | undefined;
    const stamp = Date.now();
    const title = `SuperXray New UI ${stamp}`;
    const announce = `new-ui-settings-${stamp}`;

    try {
      const settingsResponse = await expectJsonSuccess(
        await page.request.post(appUrl("panel/setting/all"), {
          headers: originHeader(page),
        }),
      );
      originalSettings = settingsResponse.obj;

      await page.goto(appUrl("panel/settings"));
      await expect(
        page.getByRole("heading", { name: "Settings", exact: true }),
      ).toBeVisible();
      await page
        .getByRole("tab", { name: "Subscription", exact: true })
        .click();
      const settingsRoot = page.locator("section.page-stack");
      await fillFormItemInput(settingsRoot, "Title", title);
      await fillFormItemTextarea(settingsRoot, "Announce", announce);

      await expect(
        page.getByRole("button", { name: "Save" }).first(),
      ).toBeEnabled();
      await page.getByRole("button", { name: "Save" }).first().click();
      const confirmDialog = page
        .locator(".ant-modal-confirm")
        .filter({ hasText: "Save panel settings?" });
      await expect(confirmDialog).toBeVisible();
      const saveResponsePromise = page.waitForResponse(
        (response) =>
          response.url().includes("/panel/setting/update") &&
          response.request().method() === "POST",
      );
      await confirmDialog.getByRole("button", { name: "Save" }).click();
      await expectJsonSuccess(await saveResponsePromise);

      await expectLegacySettingsValues(page, { announce, title });
    } finally {
      if (originalSettings) {
        await postForm(page, "panel/setting/update", originalSettings);
      }
    }
  });

  test("legacy UI can read inbounds, clients, and peers edited from the new Vue UI", async ({
    page,
  }) => {
    test.skip(
      process.env.SUPERXRAY_E2E_MUTATION !== "1",
      "Set SUPERXRAY_E2E_MUTATION=1 to run new UI edit compatibility matrix",
    );

    await login(page);

    const stamp = Date.now();
    const vlessOriginalRemark = `e2e-edit-vless-${stamp}`;
    const vlessEditedRemark = `e2e-edited-vless-${stamp}`;
    const vlessOriginalClientId = crypto.randomUUID();
    const vlessOriginalEmail = `e2e-old-client-${stamp}@example.invalid`;
    const vlessEditedEmail = `e2e-edited-client-${stamp}@example.invalid`;
    const vlessEditedComment = `edited-client-${stamp}`;
    const vlessEditedSubId = `editclient${stamp}`;
    const wireguardRemark = `e2e-edit-wg-${stamp}`;
    const wireguardOriginalEmail = `e2e-old-wg-${stamp}@example.invalid`;
    const wireguardEditedEmail = `e2e-edited-wg-${stamp}@example.invalid`;
    const wireguardEditedSubId = `editwg${stamp}`;
    let vlessInboundId: number | undefined;
    let wireguardInboundId: number | undefined;

    try {
      const vlessAdded = await postForm(page, "panel/api/inbounds/add", {
        up: 0,
        down: 0,
        total: 0,
        remark: vlessOriginalRemark,
        enable: false,
        expiryTime: 0,
        trafficReset: "never",
        lastTrafficResetTime: 0,
        listen: "127.0.0.1",
        port: 47_000 + Math.floor(Math.random() * 1000),
        protocol: "vless",
        settings: JSON.stringify({
          clients: [
            {
              id: vlessOriginalClientId,
              flow: "",
              email: vlessOriginalEmail,
              limitIp: 0,
              totalGB: 0,
              expiryTime: 0,
              enable: true,
              tgId: 0,
              subId: `oldclient${stamp}`,
              comment: "before client edit",
              reset: 0,
            },
          ],
          decryption: "none",
          encryption: "none",
        }),
        streamSettings: JSON.stringify({
          network: "tcp",
          security: "none",
          tcpSettings: { acceptProxyProtocol: false, header: { type: "none" } },
        }),
        sniffing: JSON.stringify({ enabled: false, destOverride: [] }),
      });
      vlessInboundId = vlessAdded.obj.id;

      const wireguardAdded = await postForm(page, "panel/api/inbounds/add", {
        up: 0,
        down: 0,
        total: 0,
        remark: wireguardRemark,
        enable: false,
        expiryTime: 0,
        trafficReset: "never",
        lastTrafficResetTime: 0,
        listen: "127.0.0.1",
        port: 48_000 + Math.floor(Math.random() * 1000),
        protocol: "wireguard",
        settings: JSON.stringify({
          mtu: 1420,
          secretKey: wireguardTestServerPrivate,
          pubKey: wireguardTestServerPublic,
          peers: [
            {
              privateKey: wireguardTestPeerPrivate,
              publicKey: wireguardTestPeerPublic,
              allowedIPs: ["10.0.0.2/32"],
              keepAlive: 0,
              email: wireguardOriginalEmail,
              enable: true,
              subId: `oldwg${stamp}`,
            },
          ],
          noKernelTun: false,
        }),
        streamSettings: JSON.stringify({}),
        sniffing: JSON.stringify({ enabled: false, destOverride: [] }),
      });
      wireguardInboundId = wireguardAdded.obj.id;
      if (!vlessInboundId || !wireguardInboundId) {
        throw new Error("Failed to create edit compatibility inbounds");
      }
      const vlessId = vlessInboundId;
      const wireguardId = wireguardInboundId;

      await openNewUiInbounds(page);
      const inboundRow = newUiInboundRow(page, vlessOriginalRemark);
      await expect(inboundRow).toBeVisible({ timeout: 15_000 });
      await inboundRow.getByRole("button", { name: "Edit" }).click();
      const inboundDialog = page.getByRole("dialog", { name: "Edit Inbound" });
      await expect(inboundDialog).toBeVisible();
      await fillFormItemInput(inboundDialog, "Remark", vlessEditedRemark);
      await fillFormItemInput(inboundDialog, "Listen", "127.0.0.2");
      const updateInboundPromise = page.waitForResponse(
        (response) =>
          response.url().includes(`/panel/api/inbounds/update/${vlessId}`) &&
          response.request().method() === "POST",
      );
      await inboundDialog.getByRole("button", { name: "OK" }).click();
      await expectJsonSuccess(await updateInboundPromise);
      await expect(
        page.getByText(vlessEditedRemark, { exact: true }),
      ).toBeVisible();

      const vlessDrawer = await openNewUiInboundDetail(page, vlessEditedRemark);
      const clientRow = newUiDrawerClientRow(vlessDrawer, vlessOriginalEmail);
      await expect(clientRow).toBeVisible({ timeout: 15_000 });
      await clientRow.getByRole("button", { name: "Edit" }).click();
      const clientDialog = page.getByRole("dialog", { name: "Edit Client" });
      await expect(clientDialog).toBeVisible();
      await fillFormItemInput(clientDialog, "Email", vlessEditedEmail);
      await fillFormItemInput(clientDialog, "Comment", vlessEditedComment);
      await fillFormItemInput(clientDialog, "Sub ID", vlessEditedSubId);
      await setFormItemSwitch(clientDialog, "Enable", false);
      const updateClientPromise = page.waitForResponse(
        (response) =>
          response.url().includes("/panel/api/inbounds/updateClient/") &&
          response.request().method() === "POST",
      );
      await clientDialog.getByRole("button", { name: "OK" }).click();
      await expectJsonSuccess(await updateClientPromise);

      const wireguardDrawer = await openNewUiInboundDetail(
        page,
        wireguardRemark,
      );
      const peerRow = newUiDrawerClientRow(
        wireguardDrawer,
        wireguardOriginalEmail,
      );
      await expect(peerRow).toBeVisible({ timeout: 15_000 });
      await peerRow.getByRole("button", { name: "Edit" }).click();
      const peerDialog = page.getByRole("dialog", { name: "Edit Client" });
      await expect(peerDialog).toBeVisible();
      await fillFormItemInput(peerDialog, "Email", wireguardEditedEmail);
      await fillFormItemTextarea(peerDialog, "Allowed IPs", "10.0.0.42/32");
      await fillFormItemInput(peerDialog, "Keep Alive", "15");
      await fillFormItemInput(peerDialog, "Sub ID", wireguardEditedSubId);
      await setFormItemSwitch(peerDialog, "Enable", false);
      const updatePeerPromise = page.waitForResponse(
        (response) =>
          response
            .url()
            .includes(`/panel/api/inbounds/update/${wireguardId}`) &&
          response.request().method() === "POST",
      );
      await peerDialog.getByRole("button", { name: "OK" }).click();
      await expectJsonSuccess(await updatePeerPromise);

      await waitForLegacyInbounds(page, [
        {
          id: vlessId,
          protocol: "vless",
          remark: vlessEditedRemark,
        },
        {
          id: wireguardId,
          protocol: "wireguard",
          remark: wireguardRemark,
        },
      ]);
      await expectLegacyExpandedClientText(
        page,
        vlessEditedRemark,
        vlessEditedEmail,
      );

      const vlessLegacy = await getLegacyInbound(page, vlessId);
      expect(vlessLegacy.remark).toBe(vlessEditedRemark);
      expect(vlessLegacy.listen).toBe("127.0.0.2");
      const vlessClients = legacySettingsArray(vlessLegacy, "clients");
      const vlessClient = vlessClients.find(
        (client) => client.email === vlessEditedEmail,
      );
      expect(vlessClient?.id).toBe(vlessOriginalClientId);
      expect(vlessClient?.comment).toBe(vlessEditedComment);
      expect(vlessClient?.subId).toBe(vlessEditedSubId);
      expect(vlessClient?.enable).toBe(false);

      const wireguardLegacy = await getLegacyInbound(page, wireguardId);
      expect(wireguardLegacy.remark).toBe(wireguardRemark);
      const peers = legacySettingsArray(wireguardLegacy, "peers");
      const peer = peers.find((item) => item.email === wireguardEditedEmail);
      expect(peer?.publicKey).toBe(wireguardTestPeerPublic);
      expect(peer?.allowedIPs).toEqual(["10.0.0.42/32"]);
      expect(peer?.keepAlive).toBe(15);
      expect(peer?.subId).toBe(wireguardEditedSubId);
      expect(peer?.enable).toBe(false);
    } finally {
      for (const inboundId of [wireguardInboundId, vlessInboundId]) {
        if (inboundId) {
          await postForm(page, `panel/api/inbounds/del/${inboundId}`, {});
        }
      }
    }
  });

  test("legacy UI can read Trojan, Shadowsocks, and Hysteria2 clients edited from the new Vue UI", async ({
    page,
  }) => {
    test.skip(
      process.env.SUPERXRAY_E2E_MUTATION !== "1",
      "Set SUPERXRAY_E2E_MUTATION=1 to run extended client edit compatibility matrix",
    );

    await login(page);

    const stamp = Date.now();
    const tcpStreamSettings = {
      network: "tcp",
      security: "none",
      tcpSettings: { acceptProxyProtocol: false, header: { type: "none" } },
    };
    const protocolCases = [
      {
        protocol: "trojan",
        remark: `e2e-edit-trojan-${stamp}`,
        originalEmail: `e2e-old-trojan-${stamp}@example.invalid`,
        editedEmail: `e2e-edited-trojan-${stamp}@example.invalid`,
        credentialLabel: "Password",
        credentialKey: "password",
        editedCredential: `trojan-edited-${stamp}`,
        editedComment: `edited-trojan-${stamp}`,
        editedSubId: `edittrojan${stamp}`,
        settings: {
          clients: [
            {
              password: `trojan-old-${stamp}`,
              email: `e2e-old-trojan-${stamp}@example.invalid`,
              limitIp: 0,
              totalGB: 0,
              expiryTime: 0,
              enable: true,
              tgId: 0,
              subId: `oldtrojan${stamp}`,
              comment: "before trojan edit",
              reset: 0,
            },
          ],
          fallbacks: [],
        },
        streamSettings: tcpStreamSettings,
      },
      {
        protocol: "shadowsocks",
        remark: `e2e-edit-ss-${stamp}`,
        originalEmail: `e2e-old-ss-${stamp}@example.invalid`,
        editedEmail: `e2e-edited-ss-${stamp}@example.invalid`,
        credentialLabel: "Password",
        credentialKey: "password",
        editedCredential: `ss-edited-${stamp}`,
        editedComment: `edited-ss-${stamp}`,
        editedSubId: `editss${stamp}`,
        settings: {
          method: "chacha20-ietf-poly1305",
          network: "tcp,udp",
          clients: [
            {
              method: "chacha20-ietf-poly1305",
              password: `ss-old-${stamp}`,
              email: `e2e-old-ss-${stamp}@example.invalid`,
              limitIp: 0,
              totalGB: 0,
              expiryTime: 0,
              enable: true,
              tgId: 0,
              subId: `oldss${stamp}`,
              comment: "before shadowsocks edit",
              reset: 0,
            },
          ],
          ivCheck: false,
        },
        streamSettings: tcpStreamSettings,
      },
      {
        protocol: "hysteria",
        remark: `e2e-edit-hy2-${stamp}`,
        originalEmail: `e2e-old-hy2-${stamp}@example.invalid`,
        editedEmail: `e2e-edited-hy2-${stamp}@example.invalid`,
        credentialLabel: "Auth",
        credentialKey: "auth",
        editedCredential: `hy2-edited-${stamp}`,
        editedComment: `edited-hy2-${stamp}`,
        editedSubId: `edithy2${stamp}`,
        settings: {
          version: 2,
          clients: [
            {
              auth: `hy2-old-${stamp}`,
              email: `e2e-old-hy2-${stamp}@example.invalid`,
              limitIp: 0,
              totalGB: 0,
              expiryTime: 0,
              enable: true,
              tgId: 0,
              subId: `oldhy2${stamp}`,
              comment: "before hysteria2 edit",
              reset: 0,
            },
          ],
        },
        streamSettings: {
          network: "hysteria",
          security: "tls",
          tlsSettings: {
            certificates: [
              {
                certificateFile: "/tmp/e2e-superxray.crt",
                keyFile: "/tmp/e2e-superxray.key",
              },
            ],
            alpn: ["h3"],
            settings: { fingerprint: "chrome" },
          },
          hysteriaSettings: {
            protocol: "hysteria",
            version: 2,
            auth: "",
            udpIdleTimeout: 60,
          },
        },
      },
    ] as const;
    const inboundIds = new Map<string, number>();

    try {
      for (const [index, item] of protocolCases.entries()) {
        const added = await postForm(page, "panel/api/inbounds/add", {
          up: 0,
          down: 0,
          total: 0,
          remark: item.remark,
          enable: false,
          expiryTime: 0,
          trafficReset: "never",
          lastTrafficResetTime: 0,
          listen: "127.0.0.1",
          port: 49_000 + index + Math.floor(Math.random() * 500),
          protocol: item.protocol,
          settings: JSON.stringify(item.settings),
          streamSettings: JSON.stringify(item.streamSettings),
          sniffing: JSON.stringify({ enabled: false, destOverride: [] }),
        });
        inboundIds.set(item.protocol, added.obj.id);
      }

      for (const item of protocolCases) {
        const inboundId = inboundIds.get(item.protocol);
        expect(inboundId).toBeTruthy();

        const drawer = await openNewUiInboundDetail(page, item.remark);
        const clientRow = newUiDrawerClientRow(drawer, item.originalEmail);
        await expect(clientRow).toBeVisible({ timeout: 15_000 });
        await clientRow.getByRole("button", { name: "Edit" }).click();

        const clientDialog = page.getByRole("dialog", { name: "Edit Client" });
        await expect(clientDialog).toBeVisible();
        await fillFormItemInput(clientDialog, "Email", item.editedEmail);
        await fillFormItemInput(
          clientDialog,
          item.credentialLabel,
          item.editedCredential,
        );
        await fillFormItemInput(clientDialog, "Comment", item.editedComment);
        await fillFormItemInput(clientDialog, "Sub ID", item.editedSubId);
        await setFormItemSwitch(clientDialog, "Enable", false);

        const updateClientPromise = page.waitForResponse(
          (response) =>
            response.url().includes("/panel/api/inbounds/updateClient/") &&
            response.request().method() === "POST",
        );
        await clientDialog.getByRole("button", { name: "OK" }).click();
        await expectJsonSuccess(await updateClientPromise);

        await expectLegacyExpandedClientText(
          page,
          item.remark,
          item.editedEmail,
        );

        const legacyInbound = await getLegacyInbound(page, inboundId as number);
        const clients = legacySettingsArray(legacyInbound, "clients");
        const client = clients.find(
          (record) => record.email === item.editedEmail,
        );
        expect(client?.[item.credentialKey]).toBe(item.editedCredential);
        expect(client?.comment).toBe(item.editedComment);
        expect(client?.subId).toBe(item.editedSubId);
        expect(client?.enable).toBe(false);
        if (item.protocol === "shadowsocks") {
          expect(client?.method).toBe("chacha20-ietf-poly1305");
        }
      }
    } finally {
      for (const inboundId of Array.from(inboundIds.values()).reverse()) {
        await postForm(page, `panel/api/inbounds/del/${inboundId}`, {});
      }
    }
  });

  test("can create a disabled VLESS inbound and disabled client through legacy APIs", async ({
    page,
  }) => {
    test.skip(
      process.env.SUPERXRAY_E2E_MUTATION !== "1",
      "Set SUPERXRAY_E2E_MUTATION=1 to run legacy write-flow baseline",
    );

    await login(page);

    const stamp = Date.now();
    const remark = `e2e-disabled-vless-${stamp}`;
    const port = 40_000 + Math.floor(Math.random() * 10_000);
    let inboundId: number | undefined;

    try {
      const added = await postForm(page, "panel/api/inbounds/add", {
        up: 0,
        down: 0,
        total: 0,
        remark,
        enable: false,
        expiryTime: 0,
        trafficReset: "never",
        lastTrafficResetTime: 0,
        listen: "127.0.0.1",
        port,
        protocol: "vless",
        settings: JSON.stringify({
          clients: [],
          decryption: "none",
          encryption: "none",
        }),
        streamSettings: JSON.stringify({
          network: "tcp",
          security: "none",
          tcpSettings: { acceptProxyProtocol: false, header: { type: "none" } },
        }),
        sniffing: JSON.stringify({ enabled: false, destOverride: [] }),
      });
      inboundId = added.obj.id;
      expect(inboundId).toBeGreaterThan(0);

      const clientId = crypto.randomUUID();
      const email = `e2e-client-${stamp}@example.invalid`;
      await postForm(page, "panel/api/inbounds/addClient", {
        id: inboundId,
        settings: JSON.stringify({
          clients: [
            {
              id: clientId,
              flow: "",
              email,
              limitIp: 0,
              totalGB: 0,
              expiryTime: 0,
              enable: false,
              tgId: 0,
              subId: `e2e${stamp}`,
              comment: "phase0 baseline",
              reset: 0,
            },
          ],
        }),
      });

      const list = await expectJsonSuccess(
        await page.request.get(appUrl("panel/api/inbounds/list")),
      );
      const created = list.obj.find(
        (item: { id: number }) => item.id === inboundId,
      );
      expect(created?.remark).toBe(remark);
    } finally {
      if (inboundId) {
        await postForm(page, `panel/api/inbounds/del/${inboundId}`, {});
      }
    }
  });

  test("can create disabled Phase 6 protocol inbounds through legacy APIs", async ({
    page,
  }) => {
    test.skip(
      process.env.SUPERXRAY_E2E_MUTATION !== "1",
      "Set SUPERXRAY_E2E_MUTATION=1 to run Phase 6 protocol write-flow baseline",
    );

    await login(page);

    const stamp = Date.now();
    const createdInboundIds: number[] = [];
    const protocols = [
      {
        protocol: "trojan",
        remark: `e2e-disabled-trojan-${stamp}`,
        settings: {
          clients: [
            {
              password: `trojan-${stamp}`,
              email: `e2e-trojan-${stamp}@example.invalid`,
              limitIp: 0,
              totalGB: 0,
              expiryTime: 0,
              enable: false,
              tgId: 0,
              subId: `trojan${stamp}`,
              comment: "phase6 protocol baseline",
              reset: 0,
            },
          ],
          fallbacks: [],
        },
        streamSettings: {
          network: "tcp",
          security: "none",
          tcpSettings: {
            acceptProxyProtocol: false,
            header: { type: "none" },
          },
        },
      },
      {
        protocol: "shadowsocks",
        remark: `e2e-disabled-shadowsocks-${stamp}`,
        settings: {
          method: "chacha20-ietf-poly1305",
          network: "tcp,udp",
          clients: [
            {
              method: "chacha20-ietf-poly1305",
              password: `ss-${stamp}`,
              email: `e2e-ss-${stamp}@example.invalid`,
              limitIp: 0,
              totalGB: 0,
              expiryTime: 0,
              enable: false,
              tgId: 0,
              subId: `ss${stamp}`,
              comment: "phase6 protocol baseline",
              reset: 0,
            },
          ],
          ivCheck: false,
        },
        streamSettings: {
          network: "tcp",
          security: "none",
          tcpSettings: {
            acceptProxyProtocol: false,
            header: { type: "none" },
          },
        },
      },
      {
        protocol: "hysteria",
        remark: `e2e-disabled-hysteria2-${stamp}`,
        settings: {
          version: 2,
          clients: [
            {
              auth: `hy2-${stamp}`,
              email: `e2e-hy2-${stamp}@example.invalid`,
              limitIp: 0,
              totalGB: 0,
              expiryTime: 0,
              enable: false,
              tgId: 0,
              subId: `hy2${stamp}`,
              comment: "phase6 protocol baseline",
              reset: 0,
            },
          ],
        },
        streamSettings: {
          network: "hysteria",
          security: "tls",
          tlsSettings: {
            certificates: [
              {
                certificateFile: "/tmp/e2e-superxray.crt",
                keyFile: "/tmp/e2e-superxray.key",
              },
            ],
            alpn: ["h3"],
            settings: { fingerprint: "chrome" },
          },
          hysteriaSettings: {
            protocol: "hysteria",
            version: 2,
            auth: "",
            udpIdleTimeout: 60,
          },
        },
      },
      {
        protocol: "wireguard",
        remark: `e2e-disabled-wireguard-${stamp}`,
        settings: {
          mtu: 1420,
          secretKey: wireguardTestServerPrivate,
          pubKey: wireguardTestServerPublic,
          peers: [
            {
              privateKey: wireguardTestPeerPrivate,
              publicKey: wireguardTestPeerPublic,
              allowedIPs: ["10.0.0.2/32"],
              keepAlive: 0,
              email: `e2e-wg-${stamp}@example.invalid`,
              enable: false,
              subId: `wg${stamp}`,
            },
          ],
          noKernelTun: false,
        },
        streamSettings: {},
      },
    ];

    try {
      for (const [index, item] of protocols.entries()) {
        const added = await postForm(page, "panel/api/inbounds/add", {
          up: 0,
          down: 0,
          total: 0,
          remark: item.remark,
          enable: false,
          expiryTime: 0,
          trafficReset: "never",
          lastTrafficResetTime: 0,
          listen: "127.0.0.1",
          port: 45_000 + index + Math.floor(Math.random() * 1000),
          protocol: item.protocol,
          settings: JSON.stringify(item.settings),
          streamSettings: JSON.stringify(item.streamSettings),
          sniffing: JSON.stringify({ enabled: false, destOverride: [] }),
        });
        createdInboundIds.push(added.obj.id);
        expect(added.obj.id).toBeGreaterThan(0);
      }

      const list = await expectJsonSuccess(
        await page.request.get(appUrl("panel/api/inbounds/list")),
      );
      for (const item of protocols) {
        expect(
          list.obj.some(
            (created: { remark?: string; protocol?: string }) =>
              created.remark === item.remark &&
              created.protocol === item.protocol,
          ),
        ).toBe(true);
      }
    } finally {
      for (const inboundId of createdInboundIds.reverse()) {
        await postForm(page, `panel/api/inbounds/del/${inboundId}`, {});
      }
    }
  });

  test("can open Phase 7 settings UI, save safe settings, and download database backup", async ({
    page,
  }) => {
    await login(page);

    await page.goto(appUrl("panel/settings"));
    await expect(
      page.getByRole("heading", { name: "Settings", exact: true }),
    ).toBeVisible();
    for (const tab of [
      "Panel",
      "Security",
      "Subscription",
      "Formats",
      "Telegram",
      "LDAP",
      "Backup",
    ]) {
      await expect(page.getByRole("tab", { name: tab })).toBeVisible();
    }

    const settingsResponse = await expectJsonSuccess(
      await page.request.post(appUrl("panel/setting/all"), {
        headers: originHeader(page),
      }),
    );
    const settings = settingsResponse.obj as Record<
      string,
      string | number | boolean
    >;
    const stamp = Date.now();
    await postForm(page, "panel/setting/update", {
      ...settings,
      subTitle: `SuperXray E2E ${stamp}`,
      subAnnounce: `phase7a-${stamp}`,
      subJsonFragment: JSON.stringify({ remarks: [`phase7a-${stamp}`] }),
      subJsonNoises: JSON.stringify([]),
      subJsonMux: JSON.stringify({ enabled: false }),
      subJsonRules: JSON.stringify([]),
      tgCpu: 77,
      ldapUserFilter: "(objectClass=person)",
    });

    const savedResponse = await expectJsonSuccess(
      await page.request.post(appUrl("panel/setting/all"), {
        headers: originHeader(page),
      }),
    );
    expect(savedResponse.obj.subTitle).toBe(`SuperXray E2E ${stamp}`);
    expect(savedResponse.obj.subAnnounce).toBe(`phase7a-${stamp}`);
    expect(savedResponse.obj.tgCpu).toBe(77);
    expect(savedResponse.obj.ldapUserFilter).toBe("(objectClass=person)");

    const defaultSettings = await expectJsonSuccess(
      await page.request.post(appUrl("panel/setting/defaultSettings"), {
        headers: originHeader(page),
      }),
    );
    expect(defaultSettings.obj).toHaveProperty("subURI");

    const backup = await page.request.get(appUrl("panel/api/server/getDb"));
    expect(backup.status()).toBe(200);
    expect(backup.headers()["content-disposition"]).toContain("x-ui.db");
    expectSQLiteBackup(await backup.body());

    const originalUsername = process.env.SUPERXRAY_E2E_USERNAME as string;
    const originalPassword = process.env.SUPERXRAY_E2E_PASSWORD as string;
    const temporaryUsername = `phase7a-user-${stamp}`;
    const temporaryPassword = `phase7a-pass-${stamp}`;
    let credentialsChanged = false;

    try {
      await postForm(page, "panel/setting/updateUser", {
        oldUsername: originalUsername,
        oldPassword: originalPassword,
        newUsername: temporaryUsername,
        newPassword: temporaryPassword,
      });
      credentialsChanged = true;
    } finally {
      if (credentialsChanged) {
        await page.context().clearCookies();
        await loginWith(page, temporaryUsername, temporaryPassword);
        await postForm(page, "panel/setting/updateUser", {
          oldUsername: temporaryUsername,
          oldPassword: temporaryPassword,
          newUsername: originalUsername,
          newPassword: originalPassword,
        });
        await page.context().clearCookies();
        await login(page);
      }
    }
  });

  test("rejects Phase 9 CSRF and unsafe import/download paths", async ({
    page,
    request,
  }) => {
    await login(page);

    const settingWithoutOrigin = await page.request.post(
      appUrl("panel/setting/update"),
      {
        form: { subTitle: "csrf-without-origin" },
      },
    );
    expect(settingWithoutOrigin.status()).toBe(403);

    const xrayWithoutOrigin = await page.request.post(appUrl("panel/xray/"));
    expect(xrayWithoutOrigin.status()).toBe(403);

    const settingWrongToken = await page.request.post(
      appUrl("panel/setting/update"),
      {
        form: { subTitle: "csrf-wrong-token" },
        headers: {
          Origin: new URL(page.url()).origin,
          "X-CSRF-Token": "wrong-token",
        },
      },
    );
    expect(settingWrongToken.status()).toBe(403);

    const unauthenticatedBackup = await request.get(
      appUrl("panel/api/server/getDb"),
    );
    expect(unauthenticatedBackup.status()).toBe(404);

    for (const protectedEndpoint of [
      { method: "GET", path: "panel/api/server/getConfigJson" },
      { method: "POST", path: "panel/api/server/logs/20" },
      { method: "POST", path: "panel/api/server/xraylogs/20" },
      { method: "POST", path: "panel/api/custom-geo/download/1" },
    ] as const) {
      const response =
        protectedEndpoint.method === "POST"
          ? await request.post(appUrl(protectedEndpoint.path))
          : await request.get(appUrl(protectedEndpoint.path));
      expect(response.status(), protectedEndpoint.path).toBe(404);
    }

    await expectJsonFailure(
      await page.request.post(appUrl("panel/api/server/importDB"), {
        headers: originHeader(page),
        multipart: {},
      }),
    );

    await expectJsonFailure(
      await page.request.post(appUrl("panel/api/server/importDB"), {
        headers: originHeader(page),
        multipart: {
          db: {
            name: "not-a-database.txt",
            mimeType: "text/plain",
            buffer: Buffer.from("not sqlite"),
          },
        },
      }),
    );

    await expectJsonFailure(
      await page.request.post(appUrl("panel/api/server/importDB"), {
        headers: originHeader(page),
        multipart: {
          db: {
            name: "tiny.db",
            mimeType: "application/octet-stream",
            buffer: Buffer.from("too small"),
          },
        },
      }),
    );

    await expectJsonFailure(
      await page.request.post(appUrl("panel/api/server/importDB"), {
        headers: originHeader(page),
        multipart: {
          db: {
            name: "x-ui.db.exe",
            mimeType: "application/octet-stream",
            buffer: pseudoSQLiteBuffer(),
          },
        },
      }),
    );

    await expectJsonFailure(
      await page.request.post(appUrl("panel/api/server/importDB"), {
        headers: originHeader(page),
        multipart: {
          db: {
            name: "corrupt.db",
            mimeType: "application/octet-stream",
            buffer: pseudoSQLiteBuffer(),
          },
        },
      }),
    );
  });

  test("can restart Xray through the legacy control path when explicitly enabled", async ({
    page,
  }) => {
    test.skip(
      process.env.SUPERXRAY_E2E_RESTART !== "1",
      "Set SUPERXRAY_E2E_RESTART=1 to run Xray restart baseline",
    );

    await login(page);
    await postForm(page, "panel/api/server/restartXrayService", {});
  });

  test("can round-trip database backup import when explicitly enabled", async ({
    page,
  }) => {
    test.skip(
      process.env.SUPERXRAY_E2E_IMPORT_DB !== "1",
      "Set SUPERXRAY_E2E_IMPORT_DB=1 on a real isolated Xray core environment to run DB import success baseline",
    );

    await login(page);

    const backup = await page.request.get(appUrl("panel/api/server/getDb"));
    expect(backup.status()).toBe(200);
    const dbBytes = await backup.body();
    expectSQLiteBackup(dbBytes);

    await expectJsonSuccess(
      await page.request.post(appUrl("panel/api/server/importDB"), {
        headers: originHeader(page),
        multipart: {
          db: {
            name: "phase9-roundtrip.db",
            mimeType: "application/octet-stream",
            buffer: dbBytes,
          },
        },
      }),
    );
  });

  test("can access a known subscription URL when provided", async ({
    page,
  }) => {
    test.skip(
      !process.env.SUPERXRAY_E2E_SUB_URL,
      "Set SUPERXRAY_E2E_SUB_URL to run subscription baseline",
    );

    await login(page);
    const response = await page.request.get(
      process.env.SUPERXRAY_E2E_SUB_URL as string,
    );
    expect([200, 204, 404]).toContain(response.status());
  });
});
