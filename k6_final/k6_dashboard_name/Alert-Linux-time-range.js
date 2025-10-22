import http from 'k6/http';
import { check } from 'k6';
import { Trend, Rate, Counter } from 'k6/metrics';

// Dynamic configuration via environment variables
const CONFIG = {
  timeRange: {
    from: __ENV.TIME_FROM || 'now-15m',
    to: __ENV.TIME_TO || 'now'
  },
  userCount: parseInt(__ENV.USER_COUNT) || 5,          // Default to 5 users
  iterationsPerUser: parseInt(__ENV.ITERATIONS) || 10  // Default to 10 iterations per user
};

// Validate configuration
function validateConfig(config) {
  try {
    // Validate time range
    if (typeof config.timeRange.from !== 'string' || typeof config.timeRange.to !== 'string') {
      throw new Error('Time range must be strings');
    }
    
    // Validate user count and iterations
    if (isNaN(config.userCount) || config.userCount < 1) {
      throw new Error('USER_COUNT must be a positive integer');
    }
    if (isNaN(config.iterationsPerUser) || config.iterationsPerUser < 1) {
      throw new Error('ITERATIONS must be a positive integer');
    }
    
    return true;
  } catch (e) {
    console.error(`Configuration error: ${e.message}`);
    return false;
  }
}

if (!validateConfig(CONFIG)) {
  throw new Error('Invalid configuration parameters');
}

// Read all users from file
const allUsers = open('/home/vunet/user_creation_k6/user_cookies_module.txt')
  .split('\n')
  .map(line => {
    const [username, password, accessToken, vunetSession, xVuNetHTTPInfo, grafanaSessionExpiry] = line.split(',');
    return {
      username,
      password,
      accessToken,
      vunetSession,
      xVuNetHTTPInfo,
      grafanaSessionExpiry: parseInt(grafanaSessionExpiry, 10),
      dashboardName: "Linux%20-%20CPU%20Utilization"
    };
  })
  .filter(user => user.accessToken && user.vunetSession && user.xVuNetHTTPInfo && user.grafanaSessionExpiry);

// Select the requested number of users
const selectedUsers = allUsers.slice(0, CONFIG.userCount);

if (selectedUsers.length < CONFIG.userCount) {
  console.error(`🚨 Requested ${CONFIG.userCount} users but only found ${selectedUsers.length} valid users!`);
  __ENV.K6_ABORT_ON_FAIL = 'true';
}

export let options = {
  vus: selectedUsers.length,                          // Dynamic VU count
  iterations: CONFIG.iterationsPerUser * selectedUsers.length, // Total iterations
  tags: {
    test_type: 'alert_execution',
    timeFrom: CONFIG.timeRange.from,
    timeTo: CONFIG.timeRange.to,
    userCount: CONFIG.userCount.toString(),
    iterationsPerUser: CONFIG.iterationsPerUser.toString()
  }
};

// Metrics
const alertExecutionResponseTime = new Trend('alert_execution_response_time', true);
const alertExecutionSuccessRate = new Rate('alert_execution_success_rate', true);
const userExecutionCount = new Counter('user_execution_count', true);

export default function () {
  // Distribute iterations evenly among users
  const userIndex = (__VU - 1) % selectedUsers.length;
  const user = selectedUsers[userIndex];
  
  const baseTags = {
    dashboardName: user.dashboardName,
    username: user.username,
    timeFrom: CONFIG.timeRange.from,
    timeTo: CONFIG.timeRange.to,
    vu: __VU.toString(),
    iteration: __ITER.toString(),
    userIndex: userIndex.toString()
  };

  const url = `https://164.52.214.184/vuSmartMaps/api/1/bu/1/alertrule/${user.dashboardName}/?execute_now=true&from=${encodeURIComponent(CONFIG.timeRange.from)}&to=${encodeURIComponent(CONFIG.timeRange.to)}`;
  
  const params = {
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
    tags: baseTags
  };

  const res = http.get(url, params);
  
  const metricTags = {
    ...baseTags,
    status: res.status.toString()
  };
  
  alertExecutionResponseTime.add(res.timings.duration, metricTags);
  alertExecutionSuccessRate.add(res.status === 200, metricTags);
  userExecutionCount.add(1, { username: user.username });

  check(res, {
    'is status 200': (r) => r.status === 200,
    'response time acceptable': (r) => r.timings.duration < 2000
  });

  console.log(JSON.stringify({
    timestamp: new Date().toISOString(),
    config: {
      userCount: CONFIG.userCount,
      iterationsPerUser: CONFIG.iterationsPerUser,
      timeRange: CONFIG.timeRange
    },
    ...baseTags,
    method: 'GET',
    url: url,
    responseTime: res.timings.duration,
    status: res.status
  }));
}
