import http from "k6/http";
import { check } from "k6";
import { Trend, Rate } from "k6/metrics";

// Load all users from a file
const usersRaw = open("/home/vunet/k6_final/user_creation/user_cookies.txt").split("\n");
const users = usersRaw.map((line) => {
  const [username, password] = line.split(",");
  return { username, password };
});

if (users.length === 0) {
  console.error("🚨 No valid users found in user_passwords_fixed.txt!");
  __ENV.K6_ABORT_ON_FAIL = "true";
}

console.log(users);

// Test Configuration: each VU runs one iteration
export let options = {
  scenarios: {
    default: {
      executor: "per-vu-iterations",
      vus: users.length,  // Use one VU per user
      iterations: 1,      // Run one iteration per VU
    },
  },
  insecureSkipTLSVerify: true,
};

// Base URL and login endpoint
const BASE_URL = "https://164.52.214.184";
const LOGIN_ENDPOINT = `${BASE_URL}/vui/a/vusmartmaps-app?redirect=dashboard&lte=now&gte=now-15m`;

// Metrics
const loginSuccessRate = new Rate("login_success_rate");
const loginResponseTime = new Trend("login_response_time");

export default function () {
  let user = users[__VU - 1];
  let timestamp = new Date().toISOString();

  console.log(`[${timestamp}] 🔹 VU ${__VU} Logging in as: ${user.username}`);

  // Login Request
  let loginRes = http.post(
    LOGIN_ENDPOINT,
    JSON.stringify({ username: user.username, password: user.password }),
    { headers: { "Content-Type": "application/json" }, tags: { name: "LoginRequest" } }
  );

  console.log(loginRes.headers)

  let responseTime = loginRes.timings.duration;
  let loginSuccess = check(loginRes, {
    [`Login Success (${user.username})`]: (r) => r.status === 200,
  });

  loginSuccessRate.add(loginSuccess, { username: user.username });
  loginResponseTime.add(responseTime, { username: user.username });

  let result;
  // Log the status code in a consistent way for shell parsing
  // Unique tag for shell grep
  console.log(`[K6-METRIC] status_code=${loginRes.status}`);


  if (!loginSuccess) {
  const failMsg = `[${timestamp}] ❌ Login failed | User: ${user.username} | Status: ${loginRes.status} | Response Time: ${responseTime} ms | Response: ${loginRes.body}`;
  console.error(failMsg);
} else {
  const successMsg = `[${timestamp}] ✅ Login successful | User: ${user.username} | Response Time: ${responseTime} ms`;
  console.log(successMsg);
}

}
