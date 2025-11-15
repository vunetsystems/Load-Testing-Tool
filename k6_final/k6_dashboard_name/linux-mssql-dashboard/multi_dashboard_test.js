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

// ------------------ New metrics ------------------

// Throughput per dashboard/panel (requests per second)
const dashboardThroughput = new Trend('dashboard_throughput');
const panelThroughput = new Trend('panel_throughput');

// Error breakdown
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
  const base = 'https://216.48.191.10';
  jar.set(base, 'vunet_session', user.vunetSession);
  jar.set(base, 'X-VuNet-HTTP-Info', user.xVuNetHTTPInfo);
  jar.set(base, 'grafana_session_expiry', user.grafanaSessionExpiry.toString());

  const baseDashboardAPI = config.base_urls.dashboard_api;
  const basePanelURL = config.base_urls.panel;

  // Iterate through dashboards continuously
  for (const d of config.dashboards) {
    if (String(d.enabled).toLowerCase() !== 'true') {
      console.log(`🚫 Skipping disabled dashboard: ${d.name}`);
      continue;
    }

    const dashboardUrl = `${baseDashboardAPI}${d.id}`;

    group(`Dashboard: ${d.name}`, function () {
      const startTime = Date.now();
      const { res, json } = fetchJSON(dashboardUrl, {}, headers);
      const endTime = Date.now();

      const responseTime = res.status === 200 ? (res.timings?.duration || (endTime - startTime)) : 0;
      dashboardResponseTime.add(responseTime, { dashboard: d.name });

      // Throughput (req/sec) = 1 request per duration in seconds (only for successful requests)
      let throughput = 0;
      if (responseTime > 0) {
        throughput = 1000 / responseTime;
        dashboardThroughput.add(throughput, { dashboard: d.name });
      }

      // Error breakdown
      if (res.status >= 400 && res.status < 500) dashboardError4xx.add(1, { dashboard: d.name });
      else if (res.status >= 500) dashboardError5xx.add(1, { dashboard: d.name });

      const ok = check(res, { 'Dashboard loaded successfully': (r) => r.status === 200 });
      if (ok) dashboardSuccessCount.add(1);
      else dashboardFailureCount.add(1);

      console.log(`[DASHBOARD_DATA] name=${d.name} | status=${res.status} | response_time=${responseTime.toFixed(2)}ms | throughput=${throughput.toFixed(2)}req/sec | error_4xx=${res.status >= 400 && res.status < 500 ? 1 : 0} | error_5xx=${res.status >= 500 ? 1 : 0}`);

      if (!json?.dashboard?.panels) {
        console.log(`⚠️ No panels found for ${d.name}`);
        return;
      }

      const panels = extractPanels(json.dashboard.panels);
      let panelData = [];

      for (const p of panels) {
        const panelUrl = `${basePanelURL}${d.id}/${d.slug}?orgId=1&from=${TIME_FROM}&to=${TIME_TO}&panelId=${p.id}`;
        const panelStart = Date.now();
        let panelRes;
        try {
          panelRes = http.get(panelUrl, { headers });
        } catch (err) {
          panelConnectionError.add(1, { dashboard: d.name, panel: p.title });
          console.error(`Connection error loading panel ${p.title}: ${err}`);
          continue;
        }
        const panelEnd = Date.now();

        const panelResponseTimeValue = panelRes.status === 200 ? (panelRes.timings?.duration || (panelEnd - panelStart)) : 0;
        panelResponseTime.add(panelResponseTimeValue, { dashboard: d.name, panel: p.title });

        // Throughput (only for successful requests)
        let panelThroughputValue = 0;
        if (panelResponseTimeValue > 0) {
          panelThroughputValue = 1000 / panelResponseTimeValue;
          panelThroughput.add(panelThroughputValue, { dashboard: d.name, panel: p.title });
        }

        // Error breakdown
        if (panelRes.status >= 400 && panelRes.status < 500) panelError4xx.add(1, { dashboard: d.name, panel: p.title });
        else if (panelRes.status >= 500) panelError5xx.add(1, { dashboard: d.name, panel: p.title });

        const success = panelRes.status === 200;
        if (success) panelSuccessCount.add(1);
        else panelFailureCount.add(1);

        panelData.push({
          id: p.id,
          title: p.title,
          status: panelRes.status,
          responseTime: panelResponseTimeValue,
          throughput: panelThroughputValue,
          error4xx: panelRes.status >= 400 && panelRes.status < 500 ? 1 : 0,
          error5xx: panelRes.status >= 500 ? 1 : 0
        });
      }

      // Dashboard Panel Impact
      const totalPanelTime = panelData.reduce((sum, pd) => sum + pd.responseTime, 0);
      if (totalPanelTime > 0) {
        panelData.forEach(pd => {
          const contribution = (pd.responseTime / totalPanelTime) * 100;
          const safeTitle = (pd.title || 'Untitled').replace(/\|/g, '-');
          console.log(
            `[PANEL_DATA] dashboard=${d.name} | panel_id=${pd.id} | panel_name=${safeTitle} | status=${pd.status} | response_time=${pd.responseTime.toFixed(2)}ms | throughput=${pd.throughput.toFixed(2)}req/sec | error_4xx=${pd.error4xx} | error_5xx=${pd.error5xx} | contribution=${contribution.toFixed(2)}%`
          );
        });
      }
    });

    sleep(1);
  }
}