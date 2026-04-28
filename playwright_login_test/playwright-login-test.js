const { chromium } = require('playwright');
const fs = require('fs');
const path = require('path');

// ============================================================================
// CONFIGURATION - Edit these values as needed
// ============================================================================

// User credentials file path (can be relative or absolute)
// Format: username,password (one per line)
const USER_FILE = process.env.USER_FILE || path.join(__dirname, 'users.txt');

// Base URL and login endpoint
const BASE_URL = process.env.BASE_URL || 'https://216.48.191.10';
const LOGIN_URL = `${BASE_URL}/vui/a/vusmartmaps-app?redirect=dashboard&lte=now&gte=now-15m`;

// Browser settings
const HEADLESS = process.env.HEADLESS !== 'false'; // Set to 'false' to see browser
const TIMEOUT = parseInt(process.env.TIMEOUT || '30000'); // 30 seconds default

// ClickHouse integration
const SEND_TO_CLICKHOUSE = process.env.SEND_TO_CLICKHOUSE === 'true'; // Set to 'true' to auto-send to ClickHouse

// ============================================================================
// SELECTORS (using Playwright codegen role-based selectors)
// ============================================================================
const SELECTORS = {
    username: { role: 'textbox', name: 'Username or email' },
    password: { role: 'textbox', name: 'Password' },
    signInButton: { role: 'button', name: 'Sign In' },
    dashboardUrl: '**/vusmartmaps-app**',
    dashboardHeading: { role: 'heading', name: 'Welcome to vuSmartMaps' }
};

// ============================================================================
// METRICS STORAGE
// ============================================================================
const metrics = {
    totalLogins: 0,
    successfulLogins: 0,
    failedLogins: 0,
    responseTimes: [],
    results: []
};

// ============================================================================
// LOAD USERS FROM FILE
// ============================================================================
function loadUsers() {
    try {
        if (!fs.existsSync(USER_FILE)) {
            console.error(`❌ User file not found: ${USER_FILE}`);
            console.error(`\nPlease create a file with format:`);
            console.error(`username,password`);
            console.error(`username2,password2`);
            process.exit(1);
        }

        const fileContent = fs.readFileSync(USER_FILE, 'utf-8');
        const users = fileContent.split('\n')
            .filter(line => line.trim() && !line.startsWith('#')) // Skip empty lines and comments
            .map(line => {
                const [username, password] = line.split(',');
                return { username: username?.trim(), password: password?.trim() };
            })
            .filter(user => user.username && user.password);

        if (users.length === 0) {
            console.error(`❌ No valid users found in: ${USER_FILE}`);
            console.error(`\nFile format should be:`);
            console.error(`username,password`);
            process.exit(1);
        }

        return users;
    } catch (error) {
        console.error(`❌ Error loading user file: ${error.message}`);
        process.exit(1);
    }
}

// ============================================================================
// PERFORM LOGIN FOR A SINGLE USER
// ============================================================================
async function loginUser(user, userIndex, totalUsers) {
    const browser = await chromium.launch({
        headless: HEADLESS,
        args: ['--ignore-certificate-errors']
    });

    const context = await browser.newContext({
        ignoreHTTPSErrors: true,
        viewport: { width: 1280, height: 720 }
    });

    const page = await context.newPage();
    const timestamp = new Date().toISOString();
    const startTime = Date.now();

    // Timing metrics
    let navigationStartTime = 0;
    let apiResponseTime = 0;
    let uiRenderTime = 0;
    let loginApiTime = 0;

    try {
        console.log(`[${timestamp}] 🔹 User ${userIndex + 1}/${totalUsers} | Logging in as: ${user.username}`);

        // Track navigation start
        navigationStartTime = Date.now();

        // Navigate to login page
        await page.goto(LOGIN_URL, {
            waitUntil: 'networkidle',
            timeout: TIMEOUT
        });

        const pageLoadTime = Date.now() - navigationStartTime;

        // Fill in login form using role-based selectors (from Playwright codegen)
        await page.getByRole(SELECTORS.username.role, { name: SELECTORS.username.name }).fill(user.username);
        await page.getByRole(SELECTORS.password.role, { name: SELECTORS.password.name }).fill(user.password);

        // Track login API call
        const loginApiStartTime = Date.now();

        // Listen for the authentication API response
        const authResponsePromise = page.waitForResponse(
            response => response.url().includes('/protocol/openid-connect/') ||
                response.url().includes('/login') ||
                response.url().includes('/authenticate'),
            { timeout: TIMEOUT }
        ).catch(() => null);

        // Click Sign In button
        await page.getByRole(SELECTORS.signInButton.role, { name: SELECTORS.signInButton.name }).click();

        // Wait for authentication API response
        const authResponse = await authResponsePromise;
        if (authResponse) {
            loginApiTime = Date.now() - loginApiStartTime;
        }

        // Wait for navigation to dashboard
        try {
            const uiStartTime = Date.now();
            await page.waitForURL(SELECTORS.dashboardUrl, { timeout: TIMEOUT });

            // Give the dashboard a moment to start loading
            await page.waitForTimeout(2000);

            uiRenderTime = Date.now() - uiStartTime;
            apiResponseTime = loginApiTime;

            const totalResponseTime = Date.now() - startTime;
            const currentUrl = page.url();

            console.log(`[${timestamp}] ✅ User ${userIndex + 1}/${totalUsers} | Login successful | ${user.username} | Total: ${totalResponseTime}ms | API: ${apiResponseTime}ms | UI: ${uiRenderTime}ms`);

            metrics.successfulLogins++;
            metrics.totalLogins++;
            metrics.responseTimes.push(totalResponseTime);
            metrics.results.push({
                userIndex: userIndex + 1,
                username: user.username,
                success: true,
                totalResponseTime,
                apiResponseTime,
                uiRenderTime,
                pageLoadTime,
                timestamp,
                finalUrl: currentUrl
            });

        } catch (waitError) {
            const responseTime = Date.now() - startTime;
            const currentUrl = page.url();

            console.error(`[${timestamp}] ❌ User ${userIndex + 1}/${totalUsers} | Login failed | ${user.username} | ${responseTime}ms`);

            metrics.failedLogins++;
            metrics.totalLogins++;
            metrics.responseTimes.push(responseTime);
            metrics.results.push({
                userIndex: userIndex + 1,
                username: user.username,
                success: false,
                responseTime,
                timestamp,
                finalUrl: currentUrl,
                error: 'Login timeout or redirect failed'
            });
        }

    } catch (error) {
        const responseTime = Date.now() - startTime;
        console.error(`[${timestamp}] ❌ User ${userIndex + 1}/${totalUsers} | Error | ${user.username} | ${error.message}`);

        metrics.failedLogins++;
        metrics.totalLogins++;
        metrics.responseTimes.push(responseTime);
        metrics.results.push({
            userIndex: userIndex + 1,
            username: user.username,
            success: false,
            responseTime,
            timestamp,
            error: error.message
        });
    } finally {
        await browser.close();
    }
}

// ============================================================================
// MAIN EXECUTION
// ============================================================================
async function runLoadTest() {
    console.log('🚀 Playwright Login Load Test\n');
    console.log('='.repeat(60));
    console.log(`📍 Target URL:    ${LOGIN_URL}`);
    console.log(`📁 User File:     ${USER_FILE}`);
    console.log(`🖥️  Headless Mode: ${HEADLESS}`);
    console.log(`⏱️  Timeout:       ${TIMEOUT}ms`);
    console.log('='.repeat(60) + '\n');

    const users = loadUsers();
    console.log(`📋 Loaded ${users.length} user(s) from file\n`);

    const startTime = Date.now();

    // Run all logins in parallel
    console.log(`⚡ Starting ${users.length} concurrent login attempt(s)...\n`);
    await Promise.all(users.map((user, index) => loginUser(user, index, users.length)));

    const totalTime = Date.now() - startTime;

    // Calculate statistics
    const avgResponseTime = metrics.responseTimes.length > 0
        ? metrics.responseTimes.reduce((a, b) => a + b, 0) / metrics.responseTimes.length
        : 0;
    const minResponseTime = metrics.responseTimes.length > 0 ? Math.min(...metrics.responseTimes) : 0;
    const maxResponseTime = metrics.responseTimes.length > 0 ? Math.max(...metrics.responseTimes) : 0;
    const successRate = metrics.totalLogins > 0
        ? (metrics.successfulLogins / metrics.totalLogins * 100).toFixed(2)
        : 0;

    // Calculate API vs UI timing breakdown (only for successful logins)
    const successfulResults = metrics.results.filter(r => r.success);
    const avgApiTime = successfulResults.length > 0
        ? successfulResults.reduce((sum, r) => sum + (r.apiResponseTime || 0), 0) / successfulResults.length
        : 0;
    const avgUiTime = successfulResults.length > 0
        ? successfulResults.reduce((sum, r) => sum + (r.uiRenderTime || 0), 0) / successfulResults.length
        : 0;
    const avgPageLoadTime = successfulResults.length > 0
        ? successfulResults.reduce((sum, r) => sum + (r.pageLoadTime || 0), 0) / successfulResults.length
        : 0;

    // Print summary
    console.log('\n' + '='.repeat(60));
    console.log('📊 TEST SUMMARY');
    console.log('='.repeat(60));
    console.log(`Total Users:          ${metrics.totalLogins}`);
    console.log(`Successful Logins:    ${metrics.successfulLogins} ✅ (${successRate}%)`);
    console.log(`Failed Logins:        ${metrics.failedLogins} ❌`);
    console.log(`Total Test Duration:  ${(totalTime / 1000).toFixed(2)}s`);
    console.log('');
    console.log('⏱️  TIMING BREAKDOWN (Average):');
    console.log(`  Total Response:     ${avgResponseTime.toFixed(2)}ms`);
    console.log(`  ├─ Page Load:       ${avgPageLoadTime.toFixed(2)}ms`);
    console.log(`  ├─ API Response:    ${avgApiTime.toFixed(2)}ms ${avgApiTime > avgUiTime ? '🔴 SLOWER' : '🟢'}`);
    console.log(`  └─ UI Rendering:    ${avgUiTime.toFixed(2)}ms ${avgUiTime > avgApiTime ? '🔴 SLOWER' : '🟢'}`);
    console.log('');
    console.log(`Min Response Time:    ${minResponseTime}ms`);
    console.log(`Max Response Time:    ${maxResponseTime}ms`);
    console.log('='.repeat(60) + '\n');

    // Save detailed results to JSON
    const resultsFile = path.join(__dirname, 'results', `results-${users.length}users-${Date.now()}.json`);
    const summaryData = {
        testType: 'Playwright Login Load Test',
        timestamp: new Date().toISOString(),
        targetUrl: LOGIN_URL,
        userFile: USER_FILE,
        summary: {
            totalUsers: metrics.totalLogins,
            successfulLogins: metrics.successfulLogins,
            failedLogins: metrics.failedLogins,
            successRate: `${successRate}%`,
            totalDurationMs: totalTime,
            totalDurationSec: parseFloat((totalTime / 1000).toFixed(2)),
            avgTotalResponseTime: parseFloat(avgResponseTime.toFixed(2)),
            avgPageLoadTime: parseFloat(avgPageLoadTime.toFixed(2)),
            avgApiResponseTime: parseFloat(avgApiTime.toFixed(2)),
            avgUiRenderTime: parseFloat(avgUiTime.toFixed(2)),
            minResponseTime,
            maxResponseTime,
            bottleneck: avgApiTime > avgUiTime ? 'API' : 'UI'
        },
        details: metrics.results
    };

    fs.writeFileSync(resultsFile, JSON.stringify(summaryData, null, 2));

    console.log(`📄 Detailed results saved to: ${resultsFile}\n`);

    // Send to ClickHouse if enabled
    if (SEND_TO_CLICKHOUSE) {
        console.log('🚀 Sending results to ClickHouse...\n');
        try {
            const { execSync } = require('child_process');
            const output = execSync('bash send-to-clickhouse.sh', {
                cwd: __dirname,
                encoding: 'utf-8'
            });
            console.log(output);

            // Generate HTML report after ClickHouse insertion
            console.log('📊 Generating HTML report...\n');
            try {
                const reportOutput = execSync('./generate-report', {
                    cwd: __dirname,
                    encoding: 'utf-8'
                });
                console.log(reportOutput);
            } catch (reportError) {
                console.error('⚠️  Failed to generate HTML report:', reportError.message);
                console.error('   You can manually generate it by running: ./generate-report\n');
            }
        } catch (error) {
            console.error('❌ Failed to send to ClickHouse:', error.message);
            console.error('   You can manually send results by running: bash send-to-clickhouse.sh\n');
        }
    }

    // Exit with appropriate code
    process.exit(metrics.failedLogins > 0 ? 1 : 0);
}

// ============================================================================
// RUN THE TEST
// ============================================================================
runLoadTest().catch(error => {
    console.error('💥 Fatal error:', error);
    process.exit(1);
});
