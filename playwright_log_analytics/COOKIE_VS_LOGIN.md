# Cookie-Based vs Login-Based Playwright Tests

## Overview

The Playwright log analytics framework now has **two versions**:

1. **Login-Based** (`playwright-log-analytics-test.js`) - Original version
2. **Cookie-Based** (`playwright-log-analytics-test-cookies.js`) - New K6-style version

## Comparison

| Feature | Login-Based | Cookie-Based |
|---------|-------------|--------------|
| **Speed** | 🐌 Slower | ⚡ Faster |
| **Authentication** | Logs in via UI | Uses pre-captured cookies |
| **Cookie File** | Not needed | Uses `user_cookies_module.txt` (same as K6) |
| **Login Time** | ~3-5 seconds per user | ~0 seconds (skipped) |
| **User File** | `users.txt` (username,password) | `user_cookies_module.txt` (cookies) |
| **Tests Login Flow** | ✅ Yes | ❌ No |
| **Cookie Expiration** | N/A (always fresh) | ⚠️ Cookies expire (~7 days) |
| **Similarity to K6** | Low | High |
| **Use Case** | Full E2E testing | Performance testing |

## Cookie File Format

The cookie-based version uses the **same file as K6**:

```csv
username,password,auth_token,vunet_session,X-VuNet-HTTP-Info,grafana_session_expiry
load_user_6564,Password123!,eyJhbGc...,f1fd5965...,BNPOX2uH...,1767943206
```

**Location:** `/home/vunet/Load-Testing-Tool/k6_final/k6_dashboard_name/log_analytics/user_cookies_module.txt`

A symbolic link has been created in the Playwright folder for convenience.

## How to Run

### Cookie-Based (Recommended for Performance Testing)

```bash
cd /home/vunet/Load-Testing-Tool/playwright_log_analytics

# Run with cookies (fast)
make test-cookies

# Run with cookies + send to ClickHouse
make test-cookies-send

# With visible browser
HEADLESS=false make test-cookies
```

### Login-Based (For Full E2E Testing)

```bash
# Run with login
make test

# Run with login + send to ClickHouse
make test-send
```

## What Cookies Are Used

The cookie-based version injects these cookies into the browser:

1. **`vunet_session`** - Main session cookie
2. **`X-VuNet-HTTP-Info`** - Custom authentication info
3. **`grafana_session_expiry`** - Session expiration timestamp

Plus the **`auth_token`** is available but not injected as a cookie (used for API calls if needed).

## Performance Comparison

**Example with 5 users, 8 filters:**

| Method | Login Time | Test Time | Total Time |
|--------|------------|-----------|------------|
| Login-Based | ~15-25s | ~40s | ~55-65s |
| Cookie-Based | ~0s | ~40s | ~40s |

**Savings:** ~15-25 seconds per test run (25-40% faster)

## When to Use Each

### Use Cookie-Based When:
- ✅ Testing performance/load
- ✅ Running frequent tests
- ✅ Testing with many users
- ✅ Comparing with K6 results
- ✅ Speed is important

### Use Login-Based When:
- ✅ Testing the complete user journey
- ✅ Validating login functionality
- ✅ Cookies are expired
- ✅ Need fresh authentication
- ✅ Debugging authentication issues

## Cookie Expiration

**Important:** Cookies expire after ~7 days (based on JWT expiration).

**When cookies expire, you'll see:**
```
❌ Test failed: Navigation timeout
❌ Redirected to login page
```

**Solution:** Regenerate cookies using the K6 user creation script:
```bash
cd /home/vunet/Load-Testing-Tool/k6_final/k6_dashboard_name/log_analytics
python user_creation.py 5  # Creates 5 users with fresh cookies
```

## Advantages of Cookie-Based Approach

1. **⚡ Faster** - No login overhead
2. **🔄 Consistent with K6** - Uses same cookie file
3. **📊 Better for load testing** - Can test more users quickly
4. **🎯 Focused** - Tests only log analytics, not login
5. **💰 Cost-effective** - Less browser time = lower resource usage

## Files

- **Cookie-based test:** `playwright-log-analytics-test-cookies.js`
- **Login-based test:** `playwright-log-analytics-test.js`
- **Cookie file:** `user_cookies_module.txt` (symlink to K6 file)
- **Filter file:** `filters.txt` (shared by both)
- **Results:** Both save to `results/` directory

## Example Output

```bash
$ make test-cookies

🚀 Playwright Log Analytics Load Test (Cookie-Based)

======================================================================
📍 Base URL:      https://216.48.191.10
📁 Cookie File:   /path/to/user_cookies_module.txt
📁 Filter File:   /path/to/filters.txt
🗂️  Log Source:    Apachelogs
⏰ Time Range:    Last 12 hours
🖥️  Headless:      true
🍪 Auth Method:   Pre-captured cookies (like K6)
======================================================================

📋 Loaded 5 user(s) with cookies
🔍 Loaded 8 filter(s)

[User 1/5] 🔹 Testing as: load_user_6564 (using cookies)
[User 1/5] 📊 Filters to test: 8
[User 1/5] 🗂️  Navigating to Log Analytics...
[User 1/5] ✅ Navigation complete (4521ms)
[User 1/5] 🔍 Testing filter 1/8: "500"
[User 1/5] ✅ Filter "500" | Total: 5234ms | API: 3421ms | Render: 1813ms
...
```

## Recommendation

**For most use cases, use the cookie-based version** (`make test-cookies`) as it's:
- Faster
- More aligned with K6 testing
- Better for performance testing
- Uses the same cookie infrastructure

**Use the login-based version** only when you need to test the complete authentication flow.
