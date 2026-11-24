// Run from the repo with Node 22.12+, curl, and a local Xray binary:
// node --experimental-strip-types tests/xray-ai-egress.mjs path/to/xray
import assert from 'node:assert/strict';
import { execFile, spawn } from 'node:child_process';
import { once } from 'node:events';
import { createServer as httpServer } from 'node:http';
import { connect, createServer } from 'node:net';
import { resolve } from 'node:path';
import { test } from 'node:test';
import { setTimeout as delay } from 'node:timers/promises';
import { promisify } from 'node:util';
import { mergeGatewayEgressMvpConfig } from '../frontend/src/utils/gatewayEgressMvp.ts';
import { applyAiResidentialRouting } from '../frontend/src/utils/xrayCompat.ts';

const run = promisify(execFile);
const host = '127.0.0.1';
assert.ok(process.argv[2], 'Pass the path to the Xray binary as the first argument.');
const binary = resolve(process.argv[2]);

async function listen(server) {
  server.listen(0, host);
  await once(server, 'listening');
  return server.address().port;
}

async function freePort() {
  const reservation = createServer();
  const port = await listen(reservation);
  await new Promise((done) => reservation.close(done));
  return port;
}

async function startCore(t, config) {
  const child = spawn(binary, ['run', '-config', 'stdin:'], { windowsHide: true });
  const exited = once(child, 'close');
  let output = '';
  child.stdout.on('data', (data) => {
    output += data;
  });
  child.stderr.on('data', (data) => {
    output += data;
  });
  t.after(async () => {
    if (child.exitCode === null && child.signalCode === null) child.kill();
    await exited;
  });
  child.stdin.end(JSON.stringify(config));
  const port = config.inbounds[0].port;
  for (let attempt = 0; attempt < 200; attempt++) {
    if (child.exitCode !== null) throw new Error(output);
    const socket = connect(port, host);
    try {
      await once(socket, 'connect');
      return;
    } catch (err) {
      if (err.code !== 'ECONNREFUSED') throw err;
    } finally {
      socket.destroy();
    }
    await delay(25);
  }
  throw new Error('Loopback Xray startup timed out. ' + output);
}

function socksInbound(tag, port) {
  return { tag, port, listen: host, protocol: 'socks', settings: { auth: 'noauth', udp: false } };
}

// Each outbound returns a different marker even when the HTTP status is identical.
async function markedOutbound(t, tag) {
  const server = httpServer((_req, res) => res.end(tag));
  const port = await listen(server);
  t.after(() => new Promise((done) => server.close(done)));
  return { tag, protocol: 'freedom', settings: { redirect: host + ':' + port } };
}

async function request(port, domain) {
  const args = [
    '-q',
    '--silent',
    '--show-error',
    '--noproxy',
    '',
    '--socks5-hostname',
    host + ':' + port,
    '--max-time',
    '5',
    '--fail',
    'http://' + domain + '.example.invalid/',
  ];
  try {
    const { stdout } = await run('curl', args);
    return stdout;
  } catch (err) {
    // A closed tunnel may return an empty response or reset the local socket.
    if ([52, 56].includes(err.code) && !err.stdout) return 'rejected';
    throw err;
  }
}

test('generated Gateway and residential routes on a loopback Xray core', async (t) => {
  const { stdout } = await run(binary, ['version']);
  t.diagnostic(stdout.split(/\r?\n/)[0]);
  const defaultOutbound = await markedOutbound(t, 'default');
  const residentialOrigin = await markedOutbound(t, 'residential-origin');
  const platformOutbounds = [];
  for (const platform of ['anthropic', 'openai', 'gemini']) {
    platformOutbounds.push(await markedOutbound(t, platform + '-egress'));
  }
  const uplinkPort = await freePort();
  await startCore(t, {
    inbounds: [socksInbound('uplink', uplinkPort)],
    outbounds: [residentialOrigin],
  });

  for (const scenario of ['ai-first', 'gateway-first', 'empty-selector', 'unavailable-socks']) {
    await t.test(scenario, async (s) => {
      const ordinaryPort = await freePort();
      const socksPort = scenario === 'unavailable-socks' ? await freePort() : uplinkPort;
      const base = {
        inbounds: [socksInbound('ordinary', ordinaryPort)],
        outbounds: [
          defaultOutbound,
          ...platformOutbounds,
          {
            tag: 'us-residential-socks',
            protocol: 'socks',
            settings: { servers: [{ address: host, port: socksPort }] },
          },
        ],
      };
      const config =
        scenario === 'ai-first'
          ? mergeGatewayEgressMvpConfig(applyAiResidentialRouting(base))
          : applyAiResidentialRouting(mergeGatewayEgressMvpConfig(base));
      if (scenario === 'empty-selector') config.routing.balancers[0].selector = ['no-such-node'];
      for (const inbound of config.inbounds) {
        if (inbound.tag !== 'ordinary') inbound.port = await freePort();
      }
      // Isolate names while retaining the overlap of the actual generated rules.
      for (const rule of config.routing.rules) {
        if (rule.domain) rule.domain = rule.domain.map((domain) => domain + '.example.invalid');
      }
      await startCore(s, config);
      assert.equal(await request(ordinaryPort, 'control'), 'default');
      const poolFailed = scenario === 'unavailable-socks' || scenario === 'empty-selector';
      assert.equal(
        await request(ordinaryPort, 'api.anthropic.com'),
        poolFailed ? 'rejected' : 'residential-origin',
      );
      for (const [platform, domain] of [
        ['anthropic', 'api.anthropic.com'],
        ['openai', 'api.openai.com'],
        ['gemini', 'generativelanguage.googleapis.com'],
      ]) {
        const tag = 'gateway-' + platform + '-us-primary';
        const { port } = config.inbounds.find((inbound) => inbound.tag === tag);
        assert.equal(await request(port, domain), platform + '-egress');
        assert.equal(await request(port, 'unknown'), 'rejected');
        if (platform === 'anthropic')
          assert.equal(await request(port, 'api.openai.com'), 'rejected');
      }
    });
  }
});
