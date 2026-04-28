import http from 'k6/http';
import { check, sleep } from 'k6';
import { SharedArray } from 'k6/data';
import { Trend } from 'k6/metrics';

let responseTimeTrend = new Trend('response_time', true);

// Load users with cookies
const users = new SharedArray('users', () =>
  open('/home/vunet/Load-Testing-Tool/k6_final/k6_dashboard_name/log_analytics/user_cookies_module.txt')
    .split('\n')
    .filter(l => l.trim())
    .map(line => {
      const [
        username,
        password,
        token,
        vunet_session,
        X_VuNet_HTTP_Info,
        grafana_session_expiry,
        ajs_anonymous_id
      ] = line.split(',');
      return { username, token, vunet_session, X_VuNet_HTTP_Info, grafana_session_expiry, ajs_anonymous_id };
    })
);

// Env vars
const filter = __ENV.FILTER;
const startTime = __ENV.START_TIME;
const endTime = __ENV.END_TIME;
const vus = Number(__ENV.VUS);
const iterations = Number(__ENV.ITERATIONS);

if (!filter || !startTime || !endTime || !vus || !iterations) {
  throw new Error('❌ Missing env vars');
}

export const options = { vus, iterations };

export default function () {
  if (__VU > users.length) {
    console.error(`❌ VU ${__VU} > users ${users.length}`);
    return;
  }

  const user = users[__VU - 1];

  if (!user.token || user.token.length < 100) {
    console.error(`❌ Invalid token for VU ${__VU}`);
    return;
  }

  // Parse filter: e.g., "message:10"
  const [column, value] = filter.includes(':')
    ? filter.split(':', 2)
    : ['message', filter];

  const payload = {
    queries: [
      {
        query_name: 'Query1',
        source_id: 1,
        source_name: 'Hyperscale',
        receipt_timezone: 'UTC',
        timezone: 'UTC',
        query: {
          query_type: 'time-span',
          vunet_lquery: {
            table: [
              {
                label: 'Apachelogs',
                table_name: 'vlogs_apachelogs'
              }
            ],
            size: 100,
            offset: 0,
            timestamp_column: 'timestamp',
            required_cols: [
              'timestamp',
              'message',
              'log_level',
              'log_uuid'
            ],
            query_filters: {
              filters: [
                {
                  column: column,
                  operator: 'CONTAINS',
                  value: value
                }
              ]
            }
          }
        }
      }
    ],
    time_selection: {
      start_time: startTime,
      end_time: endTime
    }
  };

  // Send request with all necessary headers + cookies
  const res = http.post(
    'https://216.48.191.10/api/vuaccel/datamodel/log_query/',
    JSON.stringify(payload),
    {
      headers: {
        'Authorization': `Bearer ${user.token}`,
        'Content-Type': 'application/json',
        'Accept': 'application/json, text/plain, */*',
        'Origin': 'https://216.48.191.10',
        'Referer': 'https://216.48.191.10/vui/a/vusmartmaps-app/observability/log_analytics/source_selection/log_analytics',
        'Cookie': `vunet_session=${user.vunet_session}; X-VuNet-HTTP-Info=${user.X_VuNet_HTTP_Info}; grafana_session_expiry=${user.grafana_session_expiry}; ajs_anonymous_id=${user.ajs_anonymous_id}`
      }
    }
  );

  if (res.status !== 200) {
    console.error(`❌ ${res.status} for ${user.username}`);
    console.error(res.body);
  }

  responseTimeTrend.add(res.timings.duration);

  check(res, {
    'status 200': r => r.status === 200
  });

  console.log(`User ${user.username} | Status ${res.status} | ${res.timings.duration}ms`);

  sleep(1);
}
