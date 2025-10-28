import http from 'k6/http';
import { check } from 'k6';
import { Trend, Rate, Counter } from 'k6/metrics';

// Dashboard configuration
const DASHBOARD_CONFIG = {
  id: 'fa44e3f5-e677-4d1e-837d-16c7d15b59c5',
  name: 'Payment Journey Observability'
};

// Time range configuration - can be set via environment variables
const TIME_RANGE = {
  from: __ENV.TIME_FROM || 'now-15m',  // Default to last 15 minutes
  to: __ENV.TIME_TO || 'now'           // Default to now
};

// Validate time range format
function validateTimeRange(from, to) {
    try {
        if (typeof from !== 'string' || typeof to !== 'string') {
            throw new Error('Time range must be strings');
        }
        if (from.length === 0 || to.length === 0) {
            throw new Error('Time range cannot be empty');
        }
        return true;
    } catch (e) {
        console.error(`Invalid time range: ${from} to ${to}`);
        return false;
    }
}

if (!validateTimeRange(TIME_RANGE.from, TIME_RANGE.to)) {
    throw new Error('Invalid time range parameters');
}

const usersRaw = open('/home/vunet/user_creation_k6/user_cookies.txt').split('\n');
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

export let options = {
    vus: users.length,
    iterations: users.length,
    tags: {
        dashboardName: DASHBOARD_CONFIG.name,
        dashboardId: DASHBOARD_CONFIG.id,
        test_type: 'dashboard_performance',
        timeFrom: TIME_RANGE.from,
        timeTo: TIME_RANGE.to
    }
};

// Custom metrics with tag support
const dashboardResponseTime = new Trend('dashboard_response_time', true);
const dashboardSuccessRate = new Rate('dashboard_success_rate', true);
const httpReqDuration = new Trend('http_req_duration_custom', true);

// User-specific metrics
const userSuccessCount = new Counter('user_success_count', true);
const userFailureCount = new Counter('user_failure_count', true);

// Panel metrics
const MAX_PANEL_ID = 150;
const panelMetrics = {};

for (let panelId = 1; panelId <= MAX_PANEL_ID; panelId++) {
    panelMetrics[panelId] = {
        responseTime: new Trend(`panel_response_time_${panelId}`, true),
        successRate: new Rate(`panel_success_rate_${panelId}`, true),
        failureRate: new Rate(`panel_failure_rate_${panelId}`, true)
    };
}

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
                    title: title || `Panel ${panel.id}`,
                    drilldownUrl: panel.links?.find(link => link.type === 'drilldown')?.url || null
                };
            });
    } catch (e) {
        console.error('Error extracting panel info:', e);
        return [];
    }
}

function fetchDashboardJson(dashboardId, user) {
    const dashboardUrl = `https://164.52.213.158/vui/api/dashboards/uid/${dashboardId}`;
    const res = http.get(dashboardUrl, {
        headers: {
            'User-Agent': 'Mozilla/5.0 (X11; Linux x86_64)',
            'Accept': 'application/json',
            'Content-Type': 'application/json'
        },
        cookies: {
            'vunet_session': user.vunetSession,
            'X-VuNet-HTTP-Info': user.xVuNetHTTPInfo,
            'grafana_session_expiry': user.grafanaSessionExpiry.toString()
        }
    });

    if (res.status !== 200) {
        console.error(`Failed to fetch dashboard ${dashboardId}: ${res.status}`);
        return null;
    }

    try {
        return res.json();
    } catch (e) {
        console.error(`Failed to parse dashboard JSON for ${dashboardId}:`, e);
        return null;
    }
}

function testDashboard(dashboardId, user, baseTags) {
    const dashboardJson = fetchDashboardJson(dashboardId, user);
    if (!dashboardJson) return;

    const panelInfo = getPanelInfo(dashboardJson);
    if (panelInfo.length === 0) {
        console.error(`No panels found in dashboard ${dashboardId}`);
        return;
    }

    panelInfo.forEach(({id: panelId, title: panelTitle, drilldownUrl}) => {
        if (panelId < 1 || panelId > MAX_PANEL_ID) {
            console.error(`Panel ID ${panelId} is out of range (1-${MAX_PANEL_ID})`);
            return;
        }

        const panelUrl = `https://164.52.213.158/vui/d/${dashboardId}/payment-journey-observability?orgId=1&viewPanel=${panelId}&from=${encodeURIComponent(TIME_RANGE.from)}&to=${encodeURIComponent(TIME_RANGE.to)}`;
        
        const panelRes = http.get(panelUrl, {
            headers: {
                'User-Agent': 'Mozilla/5.0 (X11; Linux x86_64)',
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

        panelMetrics[panelId].responseTime.add(panelRes.timings.duration, panelTags);
        panelMetrics[panelId].successRate.add(panelRes.status === 200, panelTags);
        panelMetrics[panelId].failureRate.add(panelRes.status !== 200, panelTags);
        httpReqDuration.add(panelRes.timings.duration, panelTags);

        if (panelRes.status === 200) {
            userSuccessCount.add(1, panelTags);
        } else {
            userFailureCount.add(1, panelTags);
        }

        check(panelRes, {
            [`Panel ${panelId} (${panelTitle}) is status 200`]: (r) => r.status === 200
        });

        console.log(JSON.stringify({
            timestamp: new Date().toISOString(),
            ...panelTags,
            method: 'GET',
            url: panelUrl,
            responseTime: panelRes.timings.duration,
            status: panelRes.status,
            timeRange: {
                from: TIME_RANGE.from,
                to: TIME_RANGE.to
            }
        }));

        // If the panel has a drilldown URL, test the drilldown dashboard
        if (drilldownUrl) {
            const drilldownDashboardIdMatch = drilldownUrl.match(/\/d\/([^\/]+)\//);
            if (drilldownDashboardIdMatch && drilldownDashboardIdMatch[1]) {
                const drilldownDashboardId = drilldownDashboardIdMatch[1];
                const drilldownTags = {
                    ...baseTags,
                    dashboardId: drilldownDashboardId,
                    dashboardName: `Drilldown of ${panelTitle}`
                };
                testDashboard(drilldownDashboardId, user, drilldownTags);
            }
        }
    });
}

export default function () {
    let user = users[__VU - 1];
    
    const baseTags = {
        dashboardName: DASHBOARD_CONFIG.name,
        dashboardId: DASHBOARD_CONFIG.id,
        userId: user.username,
        timeFrom: TIME_RANGE.from,
        timeTo: TIME_RANGE.to
    };

    testDashboard(DASHBOARD_CONFIG.id, user, baseTags);
}

