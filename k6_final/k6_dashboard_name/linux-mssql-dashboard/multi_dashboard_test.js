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
const users = usersRaw.map(line => {
    const [username, password, vunetSession, xVuNetHTTPInfo, grafanaSessionExpiry] = line.split(',');
    return {
        username,
        password,
        vunetSession,
        xVuNetHTTPInfo,
        grafanaSessionExpiry: parseInt(grafanaSessionExpiry, 10)
    };
}).filter(user => user.vunetSession && user.xVuNetHTTPInfo && user.grafanaSessionExpiry);

const TIME_FROM = __ENV.TIME_FROM || 'now-15m';
const TIME_TO = __ENV.TIME_TO || 'now';

// ================================
// Dashboard Mapping
// ================================
const dashboardMap = {};
config.dashboards.forEach((d, idx) => {
  dashboardMap[`dashboard_${d.slug}`] = idx;
});

// ================================
// Dynamic Options
// ================================
export const options = {
  insecureSkipTLSVerify: true,
};

// ================================
// Helper Functions
// ================================
function fetchJSON(url, params, headers, cookies) {
  const res = http.get(url, { params, headers, cookies });
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

  // Iterate through all dashboards in the config
  for (const d of config.dashboards) {
    const dashboardUrl = `${baseDashboardAPI}${d.id}`;

    group(`Dashboard: ${d.name}`, function () {
      const { res, json } = fetchJSON(dashboardUrl, {}, headers);
      const status = res.status;

      const ok = check(res, {
        'Dashboard loaded successfully': (r) => r.status === 200,
      });

      // Record metrics
      dashboardResponseTime.add(res.timings.duration, { dashboard: d.name });

      // Required for Bash parsing - log dashboard name first
      console.log(`Dashboard: ${d.name}`);
      console.log(`Dashboard is status ${status}`);

      if (ok) {
        dashboardSuccessCount.add(1);
      } else {
        dashboardFailureCount.add(1);
      }

      const dashboardSuccessRate =
        (dashboardSuccessCount.value / (dashboardSuccessCount.value + dashboardFailureCount.value)) * 100 || 0;
      console.log(`dashboard_success_rate: ${dashboardSuccessRate.toFixed(2)}%`);

      // Panels
      if (!json?.dashboard?.panels) {
        console.log(`⚠️ No panels found for ${d.name}`);
        return;
      }

      const panels = extractPanels(json.dashboard.panels);
      console.log(`🔹 Found ${panels.length} panels for ${d.name}`);

      for (const p of panels) {
        const panelUrl = `${basePanelURL}${d.id}/${d.slug}?orgId=1&from=${TIME_FROM}&to=${TIME_TO}&panelId=${p.id}`;
        const res = http.get(panelUrl, headers);
        const panelStatus = res.status;

        check(res, {
          'Panel loaded': (r) => r.status === 200,
        });

       // --- ADD THESE LINES ---
        const panelResp = res.timings.duration.toFixed(2);
        const panelSuccess = panelStatus === 200 ? 1 : 0;
        const panelSuccessRate = (panelSuccess * 100).toFixed(2);
        
        // Sanitize panel title to remove any pipe characters
        const panelTitle = p.title.replace(/\|/g, '-'); 

        // Log everything on one, atomic line
        console.log(`PANEL_DATA: dashboard_name=${d.name} | panel_id=${p.id} | panel_name=${panelTitle} | panel_status=${panelStatus} | panel_avg=${panelResp} | panel_success_rate=${panelSuccessRate}`);
      }
    });

    sleep(1); // Sleep between dashboards
  }
}
