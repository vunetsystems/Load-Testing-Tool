import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend, Rate, Counter } from 'k6/metrics';

// ===== DASHBOARD CONFIGURATIONS =====
const DASHBOARD_CONFIGS = [
  {
    id: 'b95b768c-bb56-4faf-af9e-2db19d50c149',
    name: 'Linux Server Insights',
    slug: 'linux-server-insights',
    maxPanels: 400,
    userCount: 25  // Dedicated users for this dashboard
  },
  {
    id: 'b1ec2dfd-6a8e-456b-869e-1f8df644e614',
    name: 'MSSQL Overview Dashboard',
    slug: 'mssql-overview-dashboard',
    maxPanels: 300,
    userCount: 25  // Dedicated users for this dashboard
  }
];

// ===== TIME RANGE CONFIGURATION =====
const TIME_RANGE = {
  from: __ENV.TIME_FROM || 'now-15m',
  to: __ENV.TIME_TO || 'now'
};

// ===== USER MANAGEMENT =====
const usersRaw = open('/home/vunet/k6_final/user_creation/user_cookies.txt').split('\n');
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

if (users.length === 0) {
  console.error('🚨 No valid users found in user_cookies.txt!');
  __ENV.K6_ABORT_ON_FAIL = 'true';
}

// Distribute users across dashboards
const totalUsersNeeded = DASHBOARD_CONFIGS.reduce((sum, config) => sum + config.userCount, 0);
if (users.length < totalUsersNeeded) {
  console.error(`🚨 Need ${totalUsersNeeded} users but only found ${users.length} in user_cookies.txt`);
  __ENV.K6_ABORT_ON_FAIL = 'true';
}

// ===== K6 SCENARIOS - SIMULTANEOUS EXECUTION =====
export let options = {
  scenarios: {
    'linux_dashboard_scenario': {
      executor: 'per-vu-iterations',
      vus: DASHBOARD_CONFIGS[0].userCount,
      iterations: 1,
      tags: {
        dashboard: DASHBOARD_CONFIGS[0].name,
        test_type: 'multi_dashboard_performance'
      }
    },
    'mssql_dashboard_scenario': {
      executor: 'per-vu-iterations',
      vus: DASHBOARD_CONFIGS[1].userCount,
      iterations: 1,
      tags: {
        dashboard: DASHBOARD_CONFIGS[1].name,
        test_type: 'multi_dashboard_performance'
      }
    }
  },
  insecureSkipTLSVerify: true
};

// ===== METRICS SETUP =====
const multiDashboardMetrics = {};

// Initialize metrics for each dashboard
DASHBOARD_CONFIGS.forEach(config => {
  const dashboardKey = config.name.toLowerCase().replace(/\s+/g, '_');

  multiDashboardMetrics[dashboardKey] = {
    dashboardResponseTime: new Trend(`dashboard_response_time_${dashboardKey}`, true),
    dashboardSuccessRate: new Rate(`dashboard_success_rate_${dashboardKey}`, true),
    userSuccessCount: new Counter(`user_success_count_${dashboardKey}`, true),
    userFailureCount: new Counter(`user_failure_count_${dashboardKey}`, true),

    // Panel metrics for this dashboard
    panelMetrics: {}
  };

  // Initialize panel metrics for this dashboard
  for (let panelId = 1; panelId <= config.maxPanels; panelId++) {
    multiDashboardMetrics[dashboardKey].panelMetrics[panelId] = {
      responseTime: new Trend(`panel_response_time_${dashboardKey}_${panelId}`, true),
      successRate: new Rate(`panel_success_rate_${dashboardKey}_${panelId}`, true),
      failureRate: new Rate(`panel_failure_rate_${dashboardKey}_${panelId}`, true)
    };
  }
});

// ===== UTILITY FUNCTIONS =====
function getPanelInfo(dashboardJson) {
  try {
    const findAllPanels = (item) => {
      let panels = [];
      if (item.id !== undefined) panels.push(item);
      if (item.panels) panels = panels.concat(item.panels.flatMap(findAllPanels));
      if (item.collapsed && item.panels) panels = panels.concat(item.panels.flatMap(findAllPanels));
      if (item.rows) panels = panels.concat(item.rows.flatMap(row => row.panels?.flatMap(findAllPanels) || []));
      return panels;
    };

    const allPanels = findAllPanels(dashboardJson.dashboard);

    return allPanels
      .filter(panel => panel.id !== undefined)
      .map(panel => {
        let title = panel.title;
        if (!title && panel.options) title = panel.options.title;
        if (!title && panel.targets && panel.targets[0]) {
          title = panel.targets[0].title || panel.targets[0].expr;
        }
        return {
          id: panel.id,
          title: title || `Panel ${panel.id}`
        };
      });
  } catch (e) {
    console.error('Error extracting panel info:', e);
    return [];
  }
}

function getUserForDashboard(dashboardIndex, vuId) {
  // Distribute users across dashboards
  // Dashboard 0 (Linux): users 0-24
  // Dashboard 1 (MSSQL): users 25-49
  const userIndex = (dashboardIndex * DASHBOARD_CONFIGS[0].userCount) + (vuId - 1);
  return users[userIndex];
}

function getMetricsForDashboard(dashboardName) {
  const dashboardKey = dashboardName.toLowerCase().replace(/\s+/g, '_');
  return multiDashboardMetrics[dashboardKey];
}

// ===== MAIN TEST FUNCTION =====
export default function () {
  // Determine which dashboard this VU should test based on scenario
  const scenarioName = __ENV.K6_SCENARIO;
  let dashboardConfig, dashboardIndex;

  if (scenarioName === 'linux_dashboard_scenario') {
    dashboardConfig = DASHBOARD_CONFIGS[0];
    dashboardIndex = 0;
  } else if (scenarioName === 'mssql_dashboard_scenario') {
    dashboardConfig = DASHBOARD_CONFIGS[1];
    dashboardIndex = 1;
  } else {
    console.error(`Unknown scenario: ${scenarioName}`);
    return;
  }

  // Get user for this dashboard
  const user = getUserForDashboard(dashboardIndex, __VU);
  if (!user) {
    console.error(`No user available for dashboard ${dashboardConfig.name}, VU ${__VU}`);
    return;
  }

  // Get metrics for this dashboard
  const metrics = getMetricsForDashboard(dashboardConfig.name);

  const baseTags = {
    dashboardName: dashboardConfig.name,
    dashboardId: dashboardConfig.id,
    userId: user.username,
    timeFrom: TIME_RANGE.from,
    timeTo: TIME_RANGE.to,
    scenario: scenarioName
  };

  console.log(`[${new Date().toISOString()}] 🔹 Starting test for ${dashboardConfig.name} with user ${user.username}`);

  // 1. Get dashboard JSON
  const dashboardUrl = `https://164.52.213.158/vui/api/dashboards/uid/${dashboardConfig.id}`;

  const dashboardRes = http.get(dashboardUrl, {
    headers: {
      'User-Agent': 'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
      'Accept': 'application/json',
      'Content-Type': 'application/json'
    },
    cookies: {
      'vunet_session': user.vunetSession,
      'X-VuNet-HTTP-Info': user.xVuNetHTTPInfo,
      'grafana_session_expiry': user.grafanaSessionExpiry.toString()
    },
    tags: {
      ...baseTags,
      endpoint: 'dashboard',
      request_type: 'metadata'
    }
  });

  const dashboardTags = {
    ...baseTags,
    endpoint: 'dashboard',
    status: dashboardRes.status.toString()
  };

  // Record dashboard-level metrics
  metrics.dashboardResponseTime.add(dashboardRes.timings.duration, dashboardTags);
  metrics.dashboardSuccessRate.add(dashboardRes.status === 200, dashboardTags);

  if (!check(dashboardRes, {
    [`${dashboardConfig.name} Dashboard is status 200`]: (r) => r.status === 200
  })) {
    console.error(`❌ ${dashboardConfig.name} dashboard fetch failed: ${dashboardRes.status}`);
    metrics.userFailureCount.add(1, baseTags);
    return;
  }

  let dashboardJson;
  try {
    dashboardJson = dashboardRes.json();
  } catch (e) {
    console.error(`❌ Failed to parse ${dashboardConfig.name} dashboard JSON:`, e);
    metrics.userFailureCount.add(1, baseTags);
    return;
  }

  // 2. Extract panel info
  const panelInfo = getPanelInfo(dashboardJson);
  if (panelInfo.length === 0) {
    console.error(`❌ No panels found in ${dashboardConfig.name} dashboard`);
    metrics.userFailureCount.add(1, baseTags);
    return;
  }

  console.log(`[${new Date().toISOString()}] 📋 Testing ${panelInfo.length} panels in ${dashboardConfig.name}`);

  // 3. Test each panel with time range parameters
  let successCount = 0;
  let failureCount = 0;

  for (const {id: panelId, title: panelTitle} of panelInfo) {
    if (panelId < 1 || panelId > dashboardConfig.maxPanels) {
      console.error(`⚠️ Panel ID ${panelId} is out of range for ${dashboardConfig.name} (1-${dashboardConfig.maxPanels})`);
      continue;
    }

    // Add time range parameters to the URL
    const panelUrl = `https://164.52.213.158/vui/d/${dashboardConfig.id}/${dashboardConfig.slug}?orgId=1&viewPanel=${panelId}&from=${encodeURIComponent(TIME_RANGE.from)}&to=${encodeURIComponent(TIME_RANGE.to)}`;

    const panelRes = http.get(panelUrl, {
      headers: {
        'User-Agent': 'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
        'Accept': 'application/json',
        'Content-Type': 'application/json'
      },
      cookies: {
        'vunet_session': user.vunetSession,
        'X-VuNet-HTTP-Info': user.xVuNetHTTPInfo,
        'grafana_session_expiry': user.grafanaSessionExpiry.toString()
      },
      tags: {
        ...baseTags,
        endpoint: `panel_${panelId}`,
        panelId: panelId.toString(),
        panelTitle: panelTitle,
        request_type: 'panel_view'
      }
    });

    const panelTags = {
      ...baseTags,
      panelId: panelId.toString(),
      panelTitle: panelTitle,
      status: panelRes.status.toString()
    };

    // Record panel-level metrics
    metrics.panelMetrics[panelId].responseTime.add(panelRes.timings.duration, panelTags);
    metrics.panelMetrics[panelId].successRate.add(panelRes.status === 200, panelTags);
    metrics.panelMetrics[panelId].failureRate.add(panelRes.status !== 200, panelTags);

    if (panelRes.status === 200) {
      successCount++;
    } else {
      failureCount++;
    }

    // Log panel result
    console.log(JSON.stringify({
      timestamp: new Date().toISOString(),
      dashboard: dashboardConfig.name,
      user: user.username,
      panelId: panelId,
      panelTitle: panelTitle,
      responseTime: panelRes.timings.duration,
      status: panelRes.status,
      success: panelRes.status === 200
    }));

    // Small delay between panels to avoid overwhelming the system
    sleep(Math.random() * 0.5); // 0-500ms random delay
  }

  // Record user-level metrics
  metrics.userSuccessCount.add(successCount, baseTags);
  metrics.userFailureCount.add(failureCount, baseTags);

  console.log(`[${new Date().toISOString()}] ✅ ${dashboardConfig.name} completed for user ${user.username}: ${successCount}✓ ${failureCount}✗`);
}

// ===== SUMMARY FUNCTION =====
export function handleSummary(data) {
  const summary = {};

  DASHBOARD_CONFIGS.forEach(config => {
    const dashboardKey = config.name.toLowerCase().replace(/\s+/g, '_');

    summary[`${dashboardKey}_summary.json`] = JSON.stringify({
      dashboard: config.name,
      metrics: data.metrics,
      timestamp: new Date().toISOString(),
      userCount: config.userCount,
      timeRange: TIME_RANGE
    }, null, 2);
  });

  return summary;
}