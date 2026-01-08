import http from "k6/http";
import { check } from "k6";
import { Trend, Rate, Counter } from "k6/metrics";

/* =========================================================
   USER DATA
========================================================= */
const RAW = open("/home/vunet/Load-Testing-Tool/k6_final/user_creation_k6/user_cookies.txt", "utf-8");

const users = RAW
  .split("\n")
  .map(l => l.trim())
  .filter(l => l && !l.startsWith("#"))
  .map(line => {
    const [username, password] = line.split(",");
    return { username: username?.trim(), password: password?.trim() };
  });

/* =========================================================
   ENV VARIABLES (FROM SHELL)
========================================================= */
const TEST_NAME = __ENV.TEST_NAME || "login_batch_test";
const TEST_ID = __ENV.TEST_ID || "";
const VUS = parseInt(__ENV.VUS || "1", 10);
const BATCH_ID = parseInt(__ENV.BATCH_ID || "1", 10);

/* =========================================================
   K6 OPTIONS (NO DURATION)
========================================================= */
export let options = {
  vus: VUS,
  iterations: VUS,            // EXACTLY one login per VU
  insecureSkipTLSVerify: true,
};

/* =========================================================
   ENDPOINT
========================================================= */
const BASE_URL = "https://216.48.191.10";
const LOGIN_ENDPOINT = `${BASE_URL}/vui/a/vusmartmaps-app?redirect=dashboard&lte=now&gte=now-15m`;

/* =========================================================
   METRICS
========================================================= */
const loginRT = new Trend("login_response_time", true);
const successRate = new Rate("login_success_rate");
const error4xx = new Counter("error_4xx");
const error5xx = new Counter("error_5xx");
const errorConn = new Counter("error_connection");
const bytesReceived = new Trend("response_size_bytes");

/* =========================================================
   HELPERS
========================================================= */
function getUser(vu) {
  return users[Math.min(vu - 1, users.length - 1)];
}

/* =========================================================
   DEFAULT FUNCTION
========================================================= */
export default function () {
  const user = getUser(__VU);
  const ts = new Date().toISOString();

  if (!user) {
    console.error(`[${ts}] ❌ No user mapped to VU ${__VU}`);
    return;
  }

  const payload = JSON.stringify({
    username: user.username,
    password: user.password,
  });

  const params = {
    headers: { "Content-Type": "application/json" },
    timeout: "60s",
    tags: {
      test_name: TEST_NAME,
      username: user.username,
      batch_id: BATCH_ID,
    },
  };

  let res;
  try {
    res = http.post(LOGIN_ENDPOINT, payload, params);
  } catch (e) {
    errorConn.add(1);
    successRate.add(0);

    console.error(`[${ts}] ❌ CONNECTION ERROR | User=${user.username}`);
    console.log(
      `[K6-ROW] timestamp=${ts} test_name=${TEST_NAME} username=${user.username} `
      + `avg_response_time=0 status_code=0 success_rate=0 `
      + `vus=${VUS} vus_max=${VUS} iterations=1 segment_number=${BATCH_ID} `
      + `duration=0s throughput_rps=0 error_4xx=0 error_5xx=0 `
      + `error_connection=1 response_size_bytes=0 concurrent_users=${VUS} `
      + `error_rate=1 request_rate=0 p95_response_time=0 p99_response_time=0 `
      + `test_id=${TEST_ID}`
    );
    return;
  }

  const rt = res.timings.duration;
  const size = res.body ? res.body.length : 0;

  loginRT.add(rt);
  bytesReceived.add(size);

  const ok = check(res, { "status is 200": r => r.status === 200 });
  successRate.add(ok ? 1 : 0);

  if (res.status >= 400 && res.status < 500) error4xx.add(1);
  if (res.status >= 500) error5xx.add(1);

  /* =========================================================
     STRUCTURED ROW LOG (FOR SHELL → CSV → CLICKHOUSE)
  ========================================================= */
  console.log(
    `[K6-ROW] `
    + `timestamp=${ts} `
    + `test_name=${TEST_NAME} `
    + `username=${user.username} `
    + `avg_response_time=${rt} `
    + `status_code=${res.status} `
    + `success_rate=${ok ? 100 : 0} `
    + `vus=${VUS} `
    + `vus_max=${VUS} `
    + `iterations=1 `
    + `segment_number=${BATCH_ID} `
    + `duration=0s `
    + `throughput_rps=${VUS} `
    + `error_4xx=${res.status >= 400 && res.status < 500 ? 1 : 0} `
    + `error_5xx=${res.status >= 500 ? 1 : 0} `
    + `error_connection=0 `
    + `response_size_bytes=${size} `
    + `concurrent_users=${VUS} `
    + `error_rate=${ok ? 0 : 1} `
    + `request_rate=${VUS} `
    + `p95_response_time=${rt} `
    + `p99_response_time=${rt} `
    + `test_id=${TEST_ID}`
  );

  /* =========================================================
     HUMAN LOG
  ========================================================= */
  if (ok) {
    console.log(`[${ts}] ✅ Login successful | ${user.username} | ${rt} ms`);
  } else {
    console.error(`[${ts}] ❌ Login failed | ${user.username} | ${res.status} | ${rt} ms`);
  }
}

