# Playwright Login Test - Usage Guide

## 🚀 Quick Start

### Run with 1 user:
```bash
# Edit users.txt to have 1 line
node playwright-login-test.js
```

### Run with 10 users:
```bash
# Edit users.txt to have 10 lines
node playwright-login-test.js
```

### Run with 100 users:
```bash
# Edit users.txt to have 100 lines
node playwright-login-test.js
```

## 📁 User Credentials File

**Default file:** `users.txt`

**Format:**
```
username,password
username2,password2
username3,password3
```

**Example:**
```
vunetadmin,Qwerty@123
load_user_1,Password123!
load_user_2,Password123!
testuser@example.com,SecurePass456
```

## ⚙️ Configuration Options

### Use a different credentials file:
```bash
USER_FILE=my-users.txt node playwright-login-test.js
```

### Use absolute path:
```bash
USER_FILE=/path/to/credentials.txt node playwright-login-test.js
```

### See browser in action (non-headless):
```bash
HEADLESS=false node playwright-login-test.js
```

### Change timeout (default 30 seconds):
```bash
TIMEOUT=60000 node playwright-login-test.js
```

### Combine multiple options:
```bash
USER_FILE=test-users.txt HEADLESS=false TIMEOUT=45000 node playwright-login-test.js
```

## 📊 Examples

### Example 1: Test with 1 user
```bash
# Create users.txt with 1 line
echo "vunetadmin,Qwerty@123" > users.txt

# Run test
node playwright-login-test.js
```

### Example 2: Test with 10 users
```bash
# Create users.txt with 10 different users
cat > users.txt << EOF
load_user_1,Password123!
load_user_2,Password123!
load_user_3,Password123!
load_user_4,Password123!
load_user_5,Password123!
load_user_6,Password123!
load_user_7,Password123!
load_user_8,Password123!
load_user_9,Password123!
load_user_10,Password123!
EOF

# Run test
node playwright-login-test.js
```

### Example 3: Use existing k6 user file
```bash
USER_FILE=../k6_final/user_creation_k6/user_cookies.txt node playwright-login-test.js
```

**Note:** The k6 file has extra columns (session tokens) which will be ignored. Only username and password are used.

### Example 4: Test with custom file and see browser
```bash
USER_FILE=my-test-users.txt HEADLESS=false node playwright-login-test.js
```

## 📈 Scaling Recommendations

| Users | Recommended | Notes |
|-------|-------------|-------|
| 1-10  | ✅ Perfect | Fast, reliable |
| 11-50 | ✅ Good | May be slower |
| 51-100| ⚠️ Caution | Resource intensive, use powerful machine |
| 100+  | ❌ Not recommended | Use k6 instead for large-scale load testing |

**Why?** Each Playwright browser uses ~100-200MB RAM. 100 browsers = ~20GB RAM needed.

## 📄 Output Files

After each test run, a JSON file is created:
```
results-{number}users-{timestamp}.json
```

Example:
```
results-10users-1768241232852.json
```

This file contains:
- Summary statistics (success rate, response times)
- Detailed results for each user
- Timestamps and URLs

## 🆚 Playwright vs K6

**Use Playwright when:**
- Testing 1-50 users
- Need to verify UI renders correctly
- Testing JavaScript-heavy applications
- Want screenshots/videos of login flow

**Use K6 when:**
- Testing 100+ concurrent users
- Pure API/backend load testing
- Need maximum performance
- Testing at scale

## 🔧 Troubleshooting

### Error: User file not found
```bash
# Make sure users.txt exists in the same directory
ls -la users.txt

# Or specify full path
USER_FILE=/full/path/to/users.txt node playwright-login-test.js
```

### Error: No valid users found
```bash
# Check file format (should be: username,password)
cat users.txt

# Make sure no extra spaces
# Correct: vunetadmin,Qwerty@123
# Wrong:   vunetadmin , Qwerty@123
```

### All logins failing
- Check if credentials are correct
- Try with 1 user first
- Check if server is accessible
- Look at the generated JSON file for error details

## 💡 Tips

1. **Start small:** Test with 1 user first to verify credentials work
2. **Use different usernames:** Concurrent logins with same username may fail
3. **Monitor resources:** Watch CPU/RAM usage during large tests
4. **Check results:** Always review the generated JSON file
5. **Comments in file:** Lines starting with `#` are ignored in users.txt

## 📝 Summary

**ONE script, ANY number of users!**

Just edit `users.txt` with as many users as you need, then run:
```bash
node playwright-login-test.js
```

That's it! 🎉
