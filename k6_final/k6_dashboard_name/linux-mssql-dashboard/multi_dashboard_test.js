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

// ================================
// Config Paths
// ================================
const CONFIG_PATH = '../k6_config.yaml';
const COOKIE_PATH = '../../user_creation_k6/user_cookies.txt';

// ================================
// Utility Functions
// ================================
function parseYAML(yamlText) {
  const result = { base_urls: {}, dashboards: [] };
  const lines = yamlText.split('\n');
  let section = null;
  let currentDashboard = null;

  for (let raw of lines) {
    const line = raw.trim();
    if (!line || line.startsWith('#')) continue;

    if (line === 'base_urls:') {
      section = 'base_urls';
      continue;
    }
    if (line === 'dashboards:') {
      section = 'dashboards';
      continue;
    }

    if (section === 'base_urls' && line.includes(':')) {
      const [key, ...rest] = line.split(':');
      const val = rest.join(':').trim().replace(/^['"]|['"]$/g, '');
      result.base_urls[key.trim()] = val;
      continue;
    }

    if (section === 'dashboards') {
      if (line.startsWith('-')) {
        currentDashboard = {};
        result.dashboards.push(currentDashboard);
        const afterDash = line.substring(1).trim();
        if (afterDash.includes(':')) {
          const [key, ...rest] = afterDash.split(':');
          const val = rest.join(':').trim().replace(/^['"]|['"]$/g, '');
          currentDashboard[key.trim()] = val;
        }
        continue;
      }

      if (currentDashboard && line.includes(':')) {
        const [key, ...rest] = line.split(':');
        const val = rest.join(':').trim().replace(/^['"]|['"]$/g, '');
        currentDashboard[key.trim()] = val;
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
  const res = http.get(url, { params, headers });
  let json = {};
  try {
    json = res.json();
  } catch {
    json = {};
  }
  return { res, json };
}

function extractPanels(panels) {
  let list = [];
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
  jar.set('https://qa.vunetsystems.com', 'vunet_session', user.vunetSession);
  jar.set('https://qa.vunetsystems.com', 'X-VuNet-HTTP-Info', user.xVuNetHTTPInfo);
  jar.set('https://qa.vunetsystems.com', 'grafana_session_expiry', user.grafanaSessionExpiry.toString());

  const baseDashboardAPI = config.base_urls.dashboard_api;
  const basePanelURL = config.base_urls.panel;

  // Iterate through dashboards
  for (const d of config.dashboards) {
    const dashboardUrl = `${baseDashboardAPI}${d.id}`;

    group(`Dashboard: ${d.name}`, function () {
      const startTime = Date.now();
      const { res, json } = fetchJSON(dashboardUrl, {}, headers);
      const endTime = Date.now();

      const responseTime = res.timings?.duration || (endTime - startTime);
      dashboardResponseTime.add(responseTime, { dashboard: d.name });

      const ok = check(res, { 'Dashboard loaded successfully': (r) => r.status === 200 });
      if (ok) dashboardSuccessCount.add(1);
      else dashboardFailureCount.add(1);

      console.log(`DASHBOARD_DATA: name=${d.name} | status=${res.status} | response_time=${responseTime.toFixed(2)}ms`);

      if (!json?.dashboard?.panels) {
        console.log(`⚠️ No panels found for ${d.name}`);
        return;
      }

      const panels = extractPanels(json.dashboard.panels);
      console.log(`🔹 Found ${panels.length} panels for ${d.name}`);

      for (const p of panels) {
        const panelUrl = `${basePanelURL}${d.id}/${d.slug}?orgId=1&from=${TIME_FROM}&to=${TIME_TO}&panelId=${p.id}`;

        const panelStart = Date.now();
        const res = http.get(panelUrl, { headers });
        const panelEnd = Date.now();

        const responseTime = res.timings?.duration || (panelEnd - panelStart);
        panelResponseTime.add(responseTime, { dashboard: d.name, panel: p.title });

        const success = res.status === 200;
        if (success) panelSuccessCount.add(1);
        else panelFailureCount.add(1);

        const safeTitle = (p.title || 'Untitled').replace(/\|/g, '-');

        console.log(
          `[PANEL_DATA] dashboard=${d.name} | panel_id=${p.id} | panel_name=${safeTitle} | status=${res.status} | response_time=${responseTime.toFixed(2)}ms`
        );
      }
    });

    sleep(1);
  }
}
