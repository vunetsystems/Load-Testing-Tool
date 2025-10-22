import http from 'k6/http';
import { check } from 'k6';
import { Trend, Rate, Counter } from 'k6/metrics';

// Dashboard configuration
const DASHBOARD_CONFIG = {
  id: 'b1ec2dfd-6a8e-456b-869e-1f8df644e614',
  name: 'MSSQL Overview Dashboard'
};

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
        test_type: 'dashboard_performance'
    }
};

// Custom metrics with tag support
const dashboardResponseTime = new Trend('dashboard_response_time', true);
const dashboardSuccessRate = new Rate('dashboard_success_rate', true);
const httpReqDuration = new Trend('http_req_duration_custom', true); // Custom metric to track all requests

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
                    title: title || `Panel ${panel.id}`
                };
            });
    } catch (e) {
        console.error('Error extracting panel info:', e);
        return [];
    }
}

export default function () {
    let user = users[__VU - 1];
    
    const baseTags = {
        dashboardName: DASHBOARD_CONFIG.name,
        dashboardId: DASHBOARD_CONFIG.id,
        userId: user.username
    };

    // 1. Get dashboard JSON
    const dashboardUrl = `https://164.52.213.158/vui/api/dashboards/uid/${DASHBOARD_CONFIG.id}`;
    
    // Make request with full params to ensure built-in metrics are captured
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
    
    // Add to custom metrics
    const dashboardTags = {
        ...baseTags,
        endpoint: 'dashboard',
        status: dashboardRes.status.toString()
    };
    
    dashboardResponseTime.add(dashboardRes.timings.duration, dashboardTags);
    dashboardSuccessRate.add(dashboardRes.status === 200, dashboardTags);
    httpReqDuration.add(dashboardRes.timings.duration, dashboardTags);

    if (!check(dashboardRes, {
        'Dashboard is status 200': (r) => r.status === 200
    })) {
        console.error(`Failed to fetch dashboard: ${dashboardRes.status}`);
        return;
    }

    let dashboardJson;
    try {
        dashboardJson = dashboardRes.json();
    } catch (e) {
        console.error('Failed to parse dashboard JSON:', e);
        return;
    }

    // 2. Extract panel info
    const panelInfo = getPanelInfo(dashboardJson);
    if (panelInfo.length === 0) {
        console.error('No panels found in dashboard');
        return;
    }

    // 3. Test each panel
    panelInfo.forEach(({id: panelId, title: panelTitle}) => {
        if (panelId < 1 || panelId > MAX_PANEL_ID) {
            console.error(`Panel ID ${panelId} is out of range (1-${MAX_PANEL_ID})`);
            return;
        }

        const panelUrl = `https://164.52.213.158/vui/d/${DASHBOARD_CONFIG.id}/mssql-overview-dashboard?orgId=1&viewPanel=${panelId}`;
        
        // Make panel request with complete params
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

        // Record metrics
        panelMetrics[panelId].responseTime.add(panelRes.timings.duration, panelTags);
        panelMetrics[panelId].successRate.add(panelRes.status === 200, panelTags);
        panelMetrics[panelId].failureRate.add(panelRes.status !== 200, panelTags);
        httpReqDuration.add(panelRes.timings.duration, panelTags);
        
        // User metrics
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
            status: panelRes.status
        }));
    });
}
