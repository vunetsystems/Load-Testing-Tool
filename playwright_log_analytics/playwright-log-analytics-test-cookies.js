const { chromium } = require('playwright');
const fs = require('fs');
const path = require('path');

// ============================================================================
// CONFIGURATION
// ============================================================================

// Cookie file (same format as K6)
const COOKIE_FILE = process.env.COOKIE_FILE || path.join(__dirname, 'user_cookies_module.txt');
const FILTER_FILE = process.env.FILTER_FILE || path.join(__dirname, 'filters.txt');
const BASE_URL = process.env.BASE_URL || 'https://216.48.191.10';
const LOG_SOURCE = process.env.LOG_SOURCE || 'Apachelogs';
const TIME_RANGE = process.env.TIME_RANGE || 'Last 12 hours';

// Browser settings
const HEADLESS = process.env.HEADLESS !== 'false';
const TIMEOUT = parseInt(process.env.TIMEOUT || '60000');

// ClickHouse integration
const SEND_TO_CLICKHOUSE = process.env.SEND_TO_CLICKHOUSE === 'true';

// ============================================================================
// SELECTORS
// ============================================================================
const SELECTORS = {
    toggleMenu: { testId: 'data-testid Toggle menu' },
    observabilitySection: { role: 'button', name: 'Expand section Observability' },
    logAnalyticsLink: { role: 'link', name: 'Log Analytics' },
    startNewAnalysis: { role: 'button', name: 'Start a New Analysis' },
    timePicker: { testId: 'data-testid TimePicker Open Button' },
    searchBox: { role: 'textbox', name: 'Search Keyword | Type VQL' },
    submitButton: { role: 'button', name: 'Submit' },
    deleteQuery: { role: 'button', name: 'Delete Query' }
};

// ============================================================================
// METRICS STORAGE
// ============================================================================
const metrics = {
    totalTests: 0,
    successfulTests: 0,
    failedTests: 0,
    results: []
};

// ============================================================================
// LOAD USERS WITH COOKIES FROM FILE
// ============================================================================
function loadUsersWithCookies() {
    try {
        if (!fs.existsSync(COOKIE_FILE)) {
            console.error(`❌ Cookie file not found: ${COOKIE_FILE}`);
            console.error(`\nExpected format (same as K6):`);
            console.error(`username,password,auth_token,vunet_session,X-VuNet-HTTP-Info,grafana_session_expiry`);
            process.exit(1);
        }

        const fileContent = fs.readFileSync(COOKIE_FILE, 'utf-8');
        const users = fileContent.split('\n')
            .filter(line => line.trim() && !line.startsWith('#') && !line.startsWith('username'))
            .map(line => {
                const [username, password, auth_token, vunet_session, X_VuNet_HTTP_Info, grafana_session_expiry] = line.split(',');
                return {
                    username: username?.trim(),
                    password: password?.trim(),
                    auth_token: auth_token?.trim(),
                    vunet_session: vunet_session?.trim(),
                    X_VuNet_HTTP_Info: X_VuNet_HTTP_Info?.trim(),
                    grafana_session_expiry: grafana_session_expiry?.trim()
                };
            })
            .filter(user => user.username && user.auth_token && user.vunet_session);

        if (users.length === 0) {
            console.error(`❌ No valid users found in: ${COOKIE_FILE}`);
            process.exit(1);
        }

        return users;
    } catch (error) {
        console.error(`❌ Error loading cookie file: ${error.message}`);
        process.exit(1);
    }
}

// ============================================================================
// LOAD FILTERS FROM FILE
// ============================================================================
function loadFilters() {
    try {
        if (!fs.existsSync(FILTER_FILE)) {
            console.error(`❌ Filter file not found: ${FILTER_FILE}`);
            process.exit(1);
        }

        const fileContent = fs.readFileSync(FILTER_FILE, 'utf-8');
        const filters = fileContent.split('\n')
            .filter(line => line.trim() && !line.startsWith('#'))
            .map(line => line.trim());

        if (filters.length === 0) {
            console.error(`❌ No valid filters found in: ${FILTER_FILE}`);
            process.exit(1);
        }

        return filters;
    } catch (error) {
        console.error(`❌ Error loading filter file: ${error.message}`);
        process.exit(1);
    }
}

// ============================================================================
// CREATE PLAYWRIGHT COOKIES FROM USER DATA
// ============================================================================
function createCookies(user) {
    const domain = new URL(BASE_URL).hostname;

    return [
        {
            name: 'vunet_session',
            value: user.vunet_session,
            domain: domain,
            path: '/',
            httpOnly: true,
            secure: true,
            sameSite: 'Lax'
        },
        {
            name: 'X-VuNet-HTTP-Info',
            value: user.X_VuNet_HTTP_Info,
            domain: domain,
            path: '/',
            httpOnly: false,
            secure: true,
            sameSite: 'Lax'
        },
        {
            name: 'grafana_session_expiry',
            value: user.grafana_session_expiry,
            domain: domain,
            path: '/',
            httpOnly: false,
            secure: true,
            sameSite: 'Lax'
        }
    ];
}

// ============================================================================
// PERFORM LOG ANALYTICS TEST WITH COOKIES
// ============================================================================
async function testLogAnalyticsWithCookies(user, filters, userIndex, totalUsers) {
    const browser = await chromium.launch({
        headless: HEADLESS,
        args: ['--ignore-certificate-errors']
    });

    const context = await browser.newContext({
        ignoreHTTPSErrors: true,
        viewport: { width: 1920, height: 1080 }
    });

    // Inject cookies into the browser context
    await context.addCookies(createCookies(user));

    const page = await context.newPage();
    const testStartTime = Date.now();

    try {
        console.log(`\n[User ${userIndex + 1}/${totalUsers}] 🔹 Testing as: ${user.username} (using cookies)`);
        console.log(`[User ${userIndex + 1}/${totalUsers}] 📊 Filters to test: ${filters.length}`);

        // ========== NAVIGATE DIRECTLY TO LOG ANALYTICS (NO LOGIN) ==========
        console.log(`[User ${userIndex + 1}/${totalUsers}] 🗂️  Navigating to Log Analytics...`);
        const navStartTime = Date.now();

        // Go directly to the app (already authenticated via cookies)
        await page.goto(`${BASE_URL}/vui/a/vusmartmaps-app`, {
            waitUntil: 'networkidle',
            timeout: TIMEOUT
        });

        await page.waitForTimeout(2000);

        // Navigate to Log Analytics
        await page.getByTestId(SELECTORS.toggleMenu.testId).click();
        await page.waitForTimeout(500);

        await page.getByRole(SELECTORS.observabilitySection.role, { name: SELECTORS.observabilitySection.name }).click();
        await page.waitForTimeout(500);

        await page.getByRole(SELECTORS.logAnalyticsLink.role, { name: SELECTORS.logAnalyticsLink.name }).click();
        await page.waitForTimeout(2000);

        // Select log source
        await page.locator('div:nth-child(2) > div > .css-1mp3i2x-input-wrapper > .css-1i88p6p > .css-hd7v9r-input-suffix > .css-1d3xu67-Icon').click();
        await page.waitForTimeout(500);
        await page.getByText(LOG_SOURCE).click();
        await page.waitForTimeout(500);

        // Start new analysis
        await page.getByRole(SELECTORS.startNewAnalysis.role, { name: SELECTORS.startNewAnalysis.name }).click();
        await page.waitForTimeout(2000);

        // Set time range
        await page.getByTestId(SELECTORS.timePicker.testId).click();
        await page.waitForTimeout(500);
        await page.getByText(TIME_RANGE).click();
        await page.waitForTimeout(1000);

        const navTime = Date.now() - navStartTime;
        console.log(`[User ${userIndex + 1}/${totalUsers}] ✅ Navigation complete (${navTime}ms)`);

        // ========== TEST EACH FILTER ==========
        for (let i = 0; i < filters.length; i++) {
            const filter = filters[i];
            const filterStartTime = Date.now();
            const timestamp = new Date().toISOString();

            console.log(`[User ${userIndex + 1}/${totalUsers}] 🔍 Testing filter ${i + 1}/${filters.length}: "${filter}"`);

            try {
                // Enter search filter
                const searchBox = page.getByRole(SELECTORS.searchBox.role, { name: SELECTORS.searchBox.name });
                await searchBox.click();
                await searchBox.fill(filter);

                // Track API call timing
                const apiStartTime = Date.now();

                // Listen for log query API response
                const apiResponsePromise = page.waitForResponse(
                    response => response.url().includes('/api/vuaccel/datamodel/log_query'),
                    { timeout: TIMEOUT }
                ).catch(() => null);

                // Submit search
                await searchBox.press('Enter');

                // Wait for API response
                const apiResponse = await apiResponsePromise;
                const searchApiTime = apiResponse ? Date.now() - apiStartTime : 0;

                // Wait for results to render
                const renderStartTime = Date.now();
                await page.waitForTimeout(3000);
                const resultsRenderTime = Date.now() - renderStartTime;

                const totalFilterTime = Date.now() - filterStartTime;
                const pageLoadTime = navTime;

                console.log(`[User ${userIndex + 1}/${totalUsers}] ✅ Filter "${filter}" | Total: ${totalFilterTime}ms | API: ${searchApiTime}ms | Render: ${resultsRenderTime}ms`);

                metrics.successfulTests++;
                metrics.totalTests++;
                metrics.results.push({
                    timestamp,
                    username: user.username,
                    filter,
                    success: 1,
                    total_time_ms: totalFilterTime,
                    page_load_time_ms: pageLoadTime,
                    search_api_time_ms: searchApiTime,
                    results_render_time_ms: resultsRenderTime,
                    test_id: `${Date.now()}_${filters.length}filters`
                });

                // Delete query to prepare for next filter
                await page.locator('.css-mwxb35-button.pillIconWrapper').click();
                await page.waitForTimeout(500);
                await page.getByRole(SELECTORS.deleteQuery.role, { name: SELECTORS.deleteQuery.name }).click();
                await page.waitForTimeout(1000);

            } catch (error) {
                const filterTime = Date.now() - filterStartTime;
                console.error(`[User ${userIndex + 1}/${totalUsers}] ❌ Filter "${filter}" failed: ${error.message}`);

                metrics.failedTests++;
                metrics.totalTests++;
                metrics.results.push({
                    timestamp,
                    username: user.username,
                    filter,
                    success: 0,
                    total_time_ms: filterTime,
                    page_load_time_ms: 0,
                    search_api_time_ms: 0,
                    results_render_time_ms: 0,
                    test_id: `${Date.now()}_${filters.length}filters`,
                    error: error.message
                });
            }
        }

        const totalTestTime = Date.now() - testStartTime;
        console.log(`[User ${userIndex + 1}/${totalUsers}] 🎉 All filters tested in ${(totalTestTime / 1000).toFixed(2)}s`);

    } catch (error) {
        console.error(`[User ${userIndex + 1}/${totalUsers}] ❌ Test failed: ${error.message}`);
    } finally {
        await browser.close();
    }
}

// ============================================================================
// SAVE RESULTS TO CSV
// ============================================================================
function saveResultsToCSV() {
    const csvFile = path.join(__dirname, 'results', `log_analytics_results_cookies_${Date.now()}.csv`);
    const headers = 'timestamp,username,filter,success,total_time_ms,page_load_time_ms,search_api_time_ms,results_render_time_ms,test_id\n';

    const rows = metrics.results.map(r =>
        `${r.timestamp},${r.username},"${r.filter}",${r.success},${r.total_time_ms},${r.page_load_time_ms},${r.search_api_time_ms},${r.results_render_time_ms},${r.test_id}`
    ).join('\n');

    fs.writeFileSync(csvFile, headers + rows);
    console.log(`\n📄 Results saved to: ${csvFile}`);
    return csvFile;
}

// ============================================================================
// MAIN EXECUTION
// ============================================================================
async function runLoadTest() {
    console.log('🚀 Playwright Log Analytics Load Test (Cookie-Based)\n');
    console.log('='.repeat(70));
    console.log(`📍 Base URL:      ${BASE_URL}`);
    console.log(`📁 Cookie File:   ${COOKIE_FILE}`);
    console.log(`📁 Filter File:   ${FILTER_FILE}`);
    console.log(`🗂️  Log Source:    ${LOG_SOURCE}`);
    console.log(`⏰ Time Range:    ${TIME_RANGE}`);
    console.log(`🖥️  Headless:      ${HEADLESS}`);
    console.log(`🍪 Auth Method:   Pre-captured cookies (like K6)`);
    console.log('='.repeat(70) + '\n');

    const users = loadUsersWithCookies();
    const filters = loadFilters();

    console.log(`📋 Loaded ${users.length} user(s) with cookies`);
    console.log(`🔍 Loaded ${filters.length} filter(s)\n`);

    const startTime = Date.now();

    // Run tests for all users sequentially
    for (let i = 0; i < users.length; i++) {
        await testLogAnalyticsWithCookies(users[i], filters, i, users.length);
    }

    const totalTime = Date.now() - startTime;

    // Calculate statistics
    const successRate = metrics.totalTests > 0
        ? (metrics.successfulTests / metrics.totalTests * 100).toFixed(2)
        : 0;

    const successfulResults = metrics.results.filter(r => r.success === 1);
    const avgTotalTime = successfulResults.length > 0
        ? successfulResults.reduce((sum, r) => sum + r.total_time_ms, 0) / successfulResults.length
        : 0;
    const avgSearchApiTime = successfulResults.length > 0
        ? successfulResults.reduce((sum, r) => sum + r.search_api_time_ms, 0) / successfulResults.length
        : 0;
    const avgRenderTime = successfulResults.length > 0
        ? successfulResults.reduce((sum, r) => sum + r.results_render_time_ms, 0) / successfulResults.length
        : 0;

    // Print summary
    console.log('\n' + '='.repeat(70));
    console.log('📊 TEST SUMMARY');
    console.log('='.repeat(70));
    console.log(`Total Tests:          ${metrics.totalTests}`);
    console.log(`Successful Tests:     ${metrics.successfulTests} ✅ (${successRate}%)`);
    console.log(`Failed Tests:         ${metrics.failedTests} ❌`);
    console.log(`Total Duration:       ${(totalTime / 1000).toFixed(2)}s`);
    console.log('');
    console.log('⏱️  TIMING BREAKDOWN (Average):');
    console.log(`  Total Time:         ${avgTotalTime.toFixed(2)}ms`);
    console.log(`  ├─ Search API:      ${avgSearchApiTime.toFixed(2)}ms`);
    console.log(`  └─ Results Render:  ${avgRenderTime.toFixed(2)}ms`);
    console.log('='.repeat(70) + '\n');

    // Save results
    const csvFile = saveResultsToCSV();

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

            // Generate HTML report
            console.log('📊 Generating HTML report...\n');
            try {
                const reportOutput = execSync('./generate-report', {
                    cwd: __dirname,
                    encoding: 'utf-8'
                });
                console.log(reportOutput);
            } catch (reportError) {
                console.error('⚠️  Failed to generate HTML report:', reportError.message);
            }
        } catch (error) {
            console.error('❌ Failed to send to ClickHouse:', error.message);
        }
    }

    process.exit(metrics.failedTests > 0 ? 1 : 0);
}

// ============================================================================
// RUN THE TEST
// ============================================================================
runLoadTest().catch(error => {
    console.error('💥 Fatal error:', error);
    process.exit(1);
});
