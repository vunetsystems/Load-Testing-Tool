import http from 'k6/http';
import { check, sleep } from 'k6';
import { SharedArray } from 'k6/data';

// Load users from CSV
const users = new SharedArray('user data', function () {
  return open('/home/vunet/user_creation_k6/user_cookies_module.txt')
    .split('\n')
    .filter(line => line.trim() !== '')
    .map(line => {
      const [username, password, token] = line.split(',');
      return { username, password, token };
    });
});

export let options = {
  vus: users.length,
  iterations: users.length,
};

export default function () {
  const user = users[__VU - 1]; // each VU gets a different user
  const url = 'https://164.52.213.158/api/vuaccel/datamodel/log_query/';

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
        filter_list: 'ApplicationID:2025051285451911119287',
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
      start_time: '2024-05-08T06:38:42.522Z',
      end_time:'2025-05-08T06:38:42.522Z'
    }
  };

  const headers = {
    'Authorization': `Bearer ${user.token}`,
    'Content-Type': 'application/json',
    'Accept': 'application/json, text/plain, */*',
  };

  const res = http.post(url, JSON.stringify(payload), { headers });

  check(res, {
    'status is 200': (r) => r.status === 200,
    'response time < 2s': (r) => r.timings.duration < 2000,
  });

  console.log(`User ${user.username} | Status: ${res.status} | Response time: ${res.timings.duration}ms | VQL: ${JSON.stringify(vql)}`);


  sleep(1);
}

