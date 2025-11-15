import http from "k6/http";
import { check } from "k6";
import { Trend, Rate } from "k6/metrics";

// ------------------------------
// Load users file
// ------------------------------
const RAW = open("/home/vunet/Load-Testing-Tool/k6_final/user_creation_k6/user_cookies.txt", "utf-8");
const usersRaw = RAW
  .split("\n")
  .map((l) => l.trim())
  .filter((l) => l && !l.startsWith("#"));

const users = usersRaw.map((line) => {
  // support both "username,password" and "username,password,..." lines
  const parts = line.split(",");
  return { username: parts[0] ? parts[0].trim() : "", password: parts[1] ? parts[1].trim() : "" };
});

// simple guard
if (users.length === 0) {
  console.error("🚨 No valid users found in user_cookies.txt — aborting test.");
  // k6 won't gracefully stop here, but this prevents later null access
}

// ------------------------------
// Options: duration-based login
// ------------------------------
export let options = {
  scenarios: {
    default: {
      executor: 'constant-vus',
      vus: __ENV.VUS ? parseInt(__ENV.VUS) : 1,       // Controlled by shell
      duration: __ENV.DURATION || '1m',           // Controlled by shell
      gracefulStop: '30s',
    },
  },
  insecureSkipTLSVerify: true,
};

// ------------------------------
// Endpoints & metrics
// ------------------------------
const BASE_URL = "https://216.48.191.10";
const LOGIN_ENDPOINT = `${BASE_URL}/vui/a/vusmartmaps-app?redirect=dashboard&lte=now&gte=now-15m`;

const loginSuccessRate = new Rate("login_success_rate");
const loginResponseTime = new Trend("login_response_time");

// ------------------------------
// Helper to safely get user for this VU
// ------------------------------
function getUserForVU(vuIndex) {
  if (!users || users.length === 0) return null;
  const idx = Math.max(0, Math.min(vuIndex - 1, users.length - 1));
  return users[idx];
}

// ------------------------------
// Default: attempt login once per VU
// ------------------------------
export default function () {
  const user = getUserForVU(__VU);
  const timestamp = new Date().toISOString();

  if (!user) {
    console.error(`[${timestamp}] ❌ No user mapped to VU ${__VU}. Skipping login.`);
    return;
  }

  console.log(`[${timestamp}] 🔹 VU ${__VU} attempting login for: ${user.username}`);

  const payload = JSON.stringify({ username: user.username, password: user.password });
  const params = {
    headers: { "Content-Type": "application/json" },
    tags: { name: "LoginRequest", username: user.username },
    timeout: "60s",
  };

  let loginRes;
  try {
    loginRes = http.post(LOGIN_ENDPOINT, payload, params);
  } catch (err) {
    console.error(`[${timestamp}] ❌ VU ${__VU} network/error: ${err}`);
    loginSuccessRate.add(false, { username: user.username });
    return;
  }

  const responseTime = loginRes.timings ? loginRes.timings.duration : -1;
  const ok = check(loginRes, {
    "status is 200": (r) => r.status === 200,
  });

  // metrics
  loginSuccessRate.add(ok ? 1 : 0, { username: user.username });
  loginResponseTime.add(responseTime, { username: user.username });

  // Structured log lines for downstream parsing
  console.log(`[K6-METRIC] status_code=${loginRes.status}`);
  console.log(`[K6-METRIC] response_time=${responseTime}`);

  // Print cookies returned by server (if any) so you can capture them in pipeline
  try {
    const cookieNames = Object.keys(loginRes.cookies || {});
    if (cookieNames.length > 0) {
      cookieNames.forEach((name) => {
        // log cookie name and first value; do not leak secrets if you don't want to
        const cookieVal = (loginRes.cookies[name] && loginRes.cookies[name][0] && loginRes.cookies[name][0].value) || "";
        // OPTIONAL: If you want to export cookies to external store, avoid printing raw cookie values in logs in prod.
        console.log(`[K6-COOKIE] user=${user.username} | cookie=${name} | value_snippet=${cookieVal.substring(0, 32)}`);
      });
    } else {
      console.log(`[K6-COOKIE] user=${user.username} | no-cookies-returned`);
    }
  } catch (e) {
    // non-fatal
  }

  // Human-friendly success / failure messages (kept for existing parser)
  if (!ok) {
    console.error(`[${timestamp}] ❌ Login failed | User: ${user.username} | Status: ${loginRes.status} | Response Time: ${responseTime} ms | Body: ${loginRes.body ? loginRes.body.substring(0, 300) : "<empty>"}`);
  } else {
    console.log(`[${timestamp}] ✅ Login successful | User: ${user.username} | Response Time: ${responseTime} ms`);
  }
}
