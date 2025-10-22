import http from 'k6/http';
import { check } from 'k6';
import { Trend, Rate } from 'k6/metrics';

// Define test options
export let options = {
    vus: 1, // Ensures only one iteration
    iterations: 1,
};

// Read user details from file
const usersRaw = open('/home/vunet/k6_final/user_cookies_module.txt').split('\n');
const users = usersRaw.map(line => {
    const [username, password, accessToken, vunetSession, xVuNetHTTPInfo, grafanaSessionExpiry] = line.split(',');
    return {
        username,
        password,
        accessToken,
        vunetSession,
        xVuNetHTTPInfo,
        grafanaSessionExpiry: parseInt(grafanaSessionExpiry, 10),
        dashboardName: "MSSQL%20-%20Space%20Utilization"  // Set dashboard name here
    };
}).filter(user => user.accessToken && user.vunetSession && user.xVuNetHTTPInfo && user.grafanaSessionExpiry);

if (users.length === 0) {
    console.error('🚨 No valid users found in user_cookies.txt!');
    __ENV.K6_ABORT_ON_FAIL = 'true';
}

// Metrics
const alertExecutionResponseTime = new Trend('alert_execution_response_time');
const alertExecutionSuccessRate = new Rate('alert_execution_success_rate');

export default function () {
    users.forEach(user => {
        let url = `https://164.52.214.184/vuSmartMaps/api/1/bu/1/alertrule/${user.dashboardName}/?execute_now=true`;
        
        let params = {
            headers: {
                'User-Agent': 'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
                'Accept': 'application/json',
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${user.accessToken}`
            },
            cookies: {
                'vunet_session': user.vunetSession,
                'X-VuNet-HTTP-Info': user.xVuNetHTTPInfo,
                'grafana_session_expiry': user.grafanaSessionExpiry.toString()
            },
            tags: { 
                dashboardName: user.dashboardName  // Add dashboardName as a tag
            }
        };

        let res = http.get(url, params);
        
        // Associate response time with dashboardName
        alertExecutionResponseTime.add(res.timings.duration, { dashboardName: user.dashboardName });
        alertExecutionSuccessRate.add(res.status === 200, { dashboardName: user.dashboardName });

        check(res, {
            'is status 200': (r) => r.status === 200
        });

        console.log(`Dashboard: ${user.dashboardName}, User: ${user.username}, Response Time: ${res.timings.duration}ms, Status: ${res.status}`);
    });
}

