import http from 'k6/http';
import { check, group, sleep } from 'k6';
import { SharedArray } from 'k6/data';
import { Counter, Trend } from 'k6/metrics';

// ================================
// Custom Metrics
// ================================
const dashboardResponseTime = new Trend('dashboard_response_time');
const dashboardSuccessCount = new Counter('dashboard_success_count');
const dashboardFailureCount = new Counter('dashboard_failure_count');

const panelResponseTime = new Trend('panel_response_time');
const panelSuccessCount = new Counter('panel_success_count');
const panelFailureCount = new Counter('panel_failure_count');

const dashboardThroughput = new Trend('dashboard_throughput');
const panelThroughput = new Trend('panel_throughput');

const dashboardError4xx = new Counter('dashboard_error_4xx');
const dashboardError5xx = new Counter('dashboard_error_5xx');
const panelError4xx = new Counter('panel_error_4xx');
const panelError5xx = new Counter('panel_error_5xx');
const dashboardConnectionError = new Counter('dashboard_connection_error');
const panelConnectionError = new Counter('panel_connection_error');

// ================================
// Config Paths
// ================================
const CONFIG_PATH = '../k6_config.yaml';
const COOKIE_PATH = '../../user_creation_k6/user_cookies.txt';

// ================================
// Utility Functions
// ================================
function getISOTime() {
  return new Date().toISOString().replace('T', ' ').substring(0, 19);
}

function parseYAML(yamlText) {
  const result = { base_urls: {}, dashboards: [] };
  const lines = yamlText.split('\n');
  let section = null;
  let currentDashboard = null;

  for (let raw of lines) {
    const line = raw.trim();
    if (!line || line.startsWith('#')) continue;

    if (line === 'base_urls:') section = 'base_urls';
    else if (line === 'dashboards:') section = 'dashboards';
    else if (section === 'base_urls' && line.includes(':')) {
      const [key, ...rest] = line.split(':');
      result.base_urls[key.trim()] = rest.join(':').trim().replace(/^['"]|['"]$/g, '');
    } else if (section === 'dashboards') {
      if (line.startsWith('-')) {
        currentDashboard = {};
        result.dashboards.push(currentDashboard);
        const afterDash = line.substring(1).trim();
        if (afterDash.includes(':')) {
          const [key, ...rest] = afterDash.split(':');
          currentDashboard[key.trim()] = rest.join(':').trim().replace(/^['"]|['"]$/g, '');
        }
      } else if (currentDashboard && line.includes(':')) {
        const [key, ...rest] = line.split(':');
        currentDashboard[key.trim()] = rest.join(':').trim().replace(/^['"]|['"]$/g, '');
      }
    }
  }

  return result;
}

// ================================
// Load Config & Cookies
// ================================
const config = new SharedArray('k6_config', () => {
  const raw = open(CONFIG_PATH, 'utf-8');
  return [parseYAML(raw)];
})[0];

const usersRaw = open(COOKIE_PATH, 'utf-8').split('\n');
const users = usersRaw
  .map(line => {
    const [username, password, vunetSession, xVuNetHTTPInfo, grafanaSessionExpiry] = line.split(',');
    return {
      username,
      password,
      vunetSession,
      xVuNetHTTPInfo,
      grafanaSessionExpiry: parseInt(grafanaSessionExpiry, 10),
    };
  })
  .filter(user => user.vunetSession && user.xVuNetHTTPInfo && user.grafanaSessionExpiry);

const TIME_FROM = __ENV.TIME_FROM || 'now-15m';
const TIME_TO = __ENV.TIME_TO || 'now';

// ================================
// Dynamic Options
// ================================
export const options = {
  insecureSkipTLSVerify: true,
};

// ================================
// Helper Functions
// ================================
function fetchJSON(url, params, headers) {
  let res;
  let json = {};
  let error = null;

  try {
    res = http.get(url, { params, headers });
    json = res.json();
  } catch (e) {
    error = e;
  }

  if (!res) {
    res = { status: 0, timings: { duration: 0 }, body: '' };
  }

  return { res, json, error };
}

function extractPanels(panels) {
  let list = [];
  if (!panels) return list;
  for (const p of panels) {
    if (p.id) list.push({ id: p.id, title: p.title || 'Untitled' });
    if (p.panels) list = list.concat(extractPanels(p.panels));
  }
  return list;
}

// ================================
// Main Test
// ================================
export default function () {
  const user = users[(__VU - 1) % users.length];

  const headers = {
    'User-Agent': 'k6-load-test',
    'Content-Type': 'application/json',
  };

  const jar = http.cookieJar();
  const base = 'https://216.48.191.10';
  jar.set(base, 'vunet_session', user.vunetSession);
  jar.set(base, 'X-VuNet-HTTP-Info', user.xVuNetHTTPInfo);
  jar.set(base, 'grafana_session_expiry', user.grafanaSessionExpiry.toString());

  const baseDashboardAPI = config.base_urls.dashboard_api;
  const basePanelURL = config.base_urls.panel;

  for (const d of config.dashboards) {
    if (String(d.enabled).toLowerCase() !== 'true') continue;

    const dashboardUrl = `${baseDashboardAPI}${d.id}`;

    group(`Dashboard: ${d.name}`, function () {
      const { res: dashRes, json: dashJson } = fetchJSON(dashboardUrl, {}, headers);

      if (!dashJson?.dashboard?.panels) return;

      const panels = extractPanels(dashJson.dashboard.panels);

      for (const p of panels) {
        const panelUrl = `${basePanelURL}${d.id}/${d.slug}?orgId=1&from=${TIME_FROM}&to=${TIME_TO}&panelId=${p.id}`;
        const panelStart = Date.now();
        let panelRes;
        let panelConnErr = 0;

        try {
          panelRes = http.get(panelUrl, { headers });
        } catch (err) {
          panelConnErr = 1;
          console.error(`Connection error loading panel ${p.title}: ${err}`);
        }

        if (!panelRes) panelRes = { status: 0, body: '', timings: { duration: 0 } };

        const panelEnd = Date.now();
        const panelResponseTimeValue = (panelRes.status === 200) ? (panelRes.timings?.duration || (panelEnd - panelStart)) : 0;

        // ================================
        // LOG RAW PANEL RESPONSE
        // ================================
        const bodyPreview = panelRes.body.length > 1000 ? panelRes.body.substring(0, 1000) + '...' : panelRes.body;
        console.log('----------------------------------------');
        console.log(`Panel: ${p.title} | ID: ${p.id}`);
        console.log(`HTTP Status: ${panelRes.status}`);
        console.log(`Response Time: ${panelResponseTimeValue} ms`);

        if (panelRes.body.startsWith('<!DOCTYPE html>') || panelRes.body.startsWith('<html')) {
          console.log('⚠️ Response looks like HTML (probably login redirect)');
        } else {
          console.log('✅ Panel query returned JSON/data');
        }

        console.log('Raw Response (truncated 1000 chars max):\n', bodyPreview);
        console.log('----------------------------------------');

        // Add metrics
        panelResponseTime.add(panelResponseTimeValue, { dashboard: d.name, panel: p.title });
      }
    });

    sleep(1);
  }
}
