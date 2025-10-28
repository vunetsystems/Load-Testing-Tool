import http from 'k6/http';
import { check, sleep } from 'k6';
import { SharedArray } from 'k6/data';
import { Trend } from 'k6/metrics';

let responseTimeTrend = new Trend('response_time', true);

// Load user credentials
const users = new SharedArray('users', () =>
  open('/home/vunet/user_creation_k6/user_cookies_module.txt')
    .split('\n')
    .filter(line => line.trim() !== '')
    .map(line => {
      const [username, password, token] = line.split(',');
      return { username, password, token };
    })
);

// Get env vars
const filter = __ENV.FILTER;
const startTime = __ENV.START_TIME;
const endTime = __ENV.END_TIME;
const vus = parseInt(__ENV.VUS);
const iterations = parseInt(__ENV.ITERATIONS);

// Validate
if (!filter || !startTime || !endTime || isNaN(vus) || isNaN(iterations)) {
  throw new Error('Missing or invalid required environment variables!');
}

export let options = {
  vus,
  iterations,
};

export default function () {
  const user = users[__VU - 1];

  if (!user || !user.token || user.token.trim() === '') {
    console.error(`❌ Invalid or missing token for VU ${__VU}`);
    return;
  }

  const vql = {
    table: [
      {
        label: 'Au Logs Rep',
        table_name: 'vlogs_au_logs_rep'
      }
    ],
    size: 100,
    offset: 1,
    required_cols: ['timestamp', 'message', 'log_level', 'log_uuid'],
    timestamp_column: 'timestamp',
    query_filters: [
      {
        query_format: 'VQL',
        filter_list: filter,
        apply_filter: true
      }
    ]
  };

  const payload = {
    queries: [
      {
        receipt_timezone: 'Asia/Kolkata',
        query_name: 'Query1',
        source_id: 1,
        timezone: 'UTC',
        source_name: 'Hyperscale',
        query: {
          query_type: 'time-span',
          vunet_lquery: vql
        }
      }
    ],
    time_selection: {
      start_time: startTime,
      end_time: endTime
    }
  };

  const headers = {
    'Authorization': `Bearer ${user.token}`,
    'Content-Type': 'application/json',
    'Accept': 'application/json, text/plain, */*',
  };

  try {
    const res = http.post('https://164.52.213.158/api/vuaccel/datamodel/log_query/', JSON.stringify(payload), { headers });

    if (res && res.timings && !isNaN(res.timings.duration)) {
      responseTimeTrend.add(res.timings.duration);
    } else {
      console.warn(`⚠️ No valid timing data for user: ${user.username}`);
    }

    check(res, {
      'status is 200': (r) => r.status === 200,
      'response time < 2s': (r) => r.timings.duration < 2000,
    });

    console.log(`User: ${user.username} | Filter: ${filter} | Status: ${res.status} | Time: ${res.timings.duration}ms`);

  } catch (err) {
    console.error(`❌ Request failed for user: ${user.username} | Error: ${err.message}`);
  }

  sleep(1);
}

