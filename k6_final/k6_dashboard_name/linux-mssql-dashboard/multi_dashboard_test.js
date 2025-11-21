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

// NEW: Helper to get a ClickHouse-friendly timestamp
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

// EDITED: Now returns a common structure
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
  
  // Handle cases where res might be undefined due to connection error
  if (!res) {
    res = { status: 0, timings: { duration: 0 } }; // Mock response object for failures
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
  const base = 'https://216.48.191.10'; // TODO: Make this dynamic from config?
  jar.set(base, 'vunet_session', user.vunetSession);
  jar.set(base, 'X-VuNet-HTTP-Info', user.xVuNetHTTPInfo);
  jar.set(base, 'grafana_session_expiry', user.grafanaSessionExpiry.toString());

  const baseDashboardAPI = config.base_urls.dashboard_api;
  const basePanelURL = config.base_urls.panel;

  for (const d of config.dashboards) {
    if (String(d.enabled).toLowerCase() !== 'true') {
      console.log(`🚫 Skipping disabled dashboard: ${d.name}`);
      continue;
    }

    const dashboardUrl = `${baseDashboardAPI}${d.id}`;

    group(`Dashboard: ${d.name}`, function () {
      const startTime = Date.now();
      // EDITED: Use new fetchJSON and error handling
      const { res, json, error } = fetchJSON(dashboardUrl, {}, headers);
      const endTime = Date.now();
      
      const isoTime = getISOTime();
      const responseTime = (res.status > 0) ? (res.timings?.duration || (endTime - startTime)) : 0;
      
      let throughput = 0;
      if (responseTime > 0) {
        throughput = 1000 / responseTime;
      }
      
      const err4xx = (res.status >= 400 && res.status < 500) ? 1 : 0;
      const err5xx = (res.status >= 500) ? 1 : 0;
      const connErr = (res.status === 0 || error) ? 1 : 0; // Connection Error
      
      // Add metrics
      dashboardResponseTime.add(responseTime, { dashboard: d.name });
      dashboardThroughput.add(throughput, { dashboard: d.name });
      if (err4xx) dashboardError4xx.add(1, { dashboard: d.name });
      if (err5xx) dashboardError5xx.add(1, { dashboard: d.name });
      if (connErr) dashboardConnectionError.add(1, { dashboard: d.name });

      const ok = check(res, { 'Dashboard loaded successfully': (r) => r.status === 200 });
      if (ok) {
        dashboardSuccessCount.add(1);
      } else {
        dashboardFailureCount.add(1);
      }

      // ----------------------------------------------------------------------
      // NEW ROBUST LOGGING FORMAT (Pipe-delimited)
      // [TYPE] | Timestamp | Dashboard Name | Status | Resp Time | Throughput | 4xx | 5xx | Conn Err
      // ----------------------------------------------------------------------
      console.log(
        `[DASHBOARD_DATA] | ${isoTime} | ${d.name} | ${res.status} | ${responseTime.toFixed(2)} | ${throughput.toFixed(2)} | ${err4xx} | ${err5xx} | ${connErr}`
      );

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
        let panelConnErr = 0;
        
        try {
          panelRes = http.get(panelUrl, { headers });
        } catch (err) {
          panelConnErr = 1;
          console.error(`Connection error loading panel ${p.title}: ${err}`);
        }
        
        // Handle no-response case
        if (!panelRes) {
            panelRes = { status: 0, timings: { duration: 0 } }; // Mock response
        }
        
        const panelEnd = Date.now();
        const panelIsoTime = getISOTime(); // Timestamp for the panel event

        const panelResponseTimeValue = (panelRes.status === 200) ? (panelRes.timings?.duration || (panelEnd - panelStart)) : 0;
        
        let panelThroughputValue = 0;
        if (panelResponseTimeValue > 0) {
          panelThroughputValue = 1000 / panelResponseTimeValue;
        }

        const panelErr4xx = (panelRes.status >= 400 && panelRes.status < 500) ? 1 : 0;
        const panelErr5xx = (panelRes.status >= 500) ? 1 : 0;
        if (panelConnErr === 0 && panelRes.status === 0) panelConnErr = 1; // Catch k6 internal 0 status

        // Add metrics
        panelResponseTime.add(panelResponseTimeValue, { dashboard: d.name, panel: p.title });
        panelThroughput.add(panelThroughputValue, { dashboard: d.name, panel: p.title });
        if (panelErr4xx) panelError4xx.add(1, { dashboard: d.name, panel: p.title });
        if (panelErr5xx) panelError5xx.add(1, { dashboard: d.name, panel: p.title });
        if (panelConnErr) panelConnectionError.add(1, { dashboard: d.name, panel: p.title });

        const success = panelRes.status === 200;
        if (success) panelSuccessCount.add(1);
        else panelFailureCount.add(1);

        panelData.push({
          id: p.id,
          title: p.title,
          status: panelRes.status,
          responseTime: panelResponseTimeValue,
          throughput: panelThroughputValue,
          error4xx: panelErr4xx,
          error5xx: panelErr5xx,
          connErr: panelConnErr,
          timestamp: panelIsoTime // Store timestamp
        });
      }

      // Dashboard Panel Impact
      const totalPanelTime = panelData.reduce((sum, pd) => sum + pd.responseTime, 0);
      
      panelData.forEach(pd => {
          let contribution = 0;
          if (totalPanelTime > 0 && pd.responseTime > 0) {
            contribution = (pd.responseTime / totalPanelTime) * 100;
          }
          const safeTitle = (pd.title || 'Untitled').replace(/\|/g, '-'); // Remove pipes from name

          // ----------------------------------------------------------------------
          // NEW ROBUST LOGGING FORMAT (Pipe-delimited)
          // [TYPE] | Timestamp | Dashboard Name | Panel ID | Panel Name | Status | Resp Time | Throughput | 4xx | 5xx | Conn Err | Contribution %
          // ----------------------------------------------------------------------
          console.log(
            `[PANEL_DATA] | ${pd.timestamp} | ${d.name} | ${pd.id} | ${safeTitle} | ${pd.status} | ${pd.responseTime.toFixed(2)} | ${pd.throughput.toFixed(2)} | ${pd.error4xx} | ${pd.error5xx} | ${pd.connErr} | ${contribution.toFixed(2)}`
          );
      });
    });

    sleep(1);
  }
}