const { chromium } = require('playwright');
const fs = require('fs');
const path = require('path');

// ============================================================================
// CONFIGURATION
// ============================================================================

const USER_FILE = process.env.USER_FILE || path.join(__dirname, 'users.txt');
const FILTER_FILE = process.env.FILTER_FILE || path.join(__dirname, 'filters.txt');
const BASE_URL = process.env.BASE_URL || 'https://216.48.191.10';
const LOG_SOURCE = process.env.LOG_SOURCE || 'Apachelogs';
const TIME_RANGE = process.env.TIME_RANGE || 'Last 12 hours';

// Browser settings
const HEADLESS = process.env.HEADLESS !== 'false';
const TIMEOUT = parseInt(process.env.TIMEOUT || '60000'); // 60 seconds for log queries

// ClickHouse integration
const SEND_TO_CLICKHOUSE = process.env.SEND_TO_CLICKHOUSE === 'true';

// ============================================================================
// SELECTORS
// ============================================================================
const SELECTORS = {
    username: { role: 'textbox', name: 'Username or email' },
    password: { role: 'textbox', name: 'Password' },
    signInButton: { role: 'button', name: 'Sign In' },
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
// LOAD USERS FROM FILE
// ============================================================================
function loadUsers() {
    try {
        if (!fs.existsSync(USER_FILE)) {
            console.error(`❌ User file not found: ${USER_FILE}`);
            process.exit(1);
        }

        const fileContent = fs.readFileSync(USER_FILE, 'utf-8');
        const users = fileContent.split('\n')
            .filter(line => line.trim() && !line.startsWith('#'))
            .map(line => {
                const [username, password] = line.split(',');
                return { username: username?.trim(), password: password?.trim() };
            })
            .filter(user => user.username && user.password);

        if (users.length === 0) {
            console.error(`❌ No valid users found in: ${USER_FILE}`);
            process.exit(1);
        }

        return users;
    } catch (error) {
        console.error(`❌ Error loading user file: ${error.message}`);
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
// PERFORM LOG ANALYTICS TEST FOR A USER
// ============================================================================
async function testLogAnalytics(user, filters, userIndex, totalUsers, testId) {
    const browser = await chromium.launch({
        headless: HEADLESS,
        args: ['--ignore-certificate-errors']
    });

    const context = await browser.newContext({
        ignoreHTTPSErrors: true,
        viewport: { width: 1920, height: 1080 }
    });

    const page = await context.newPage();
    const testStartTime = Date.now();

    try {
        console.log(`\n[User ${userIndex + 1}/${totalUsers}] 🔹 Testing as: ${user.username}`);
        console.log(`[User ${userIndex + 1}/${totalUsers}] 📊 Filters to test: ${filters.length}`);

        // ========== LOGIN ==========
        console.log(`[User ${userIndex + 1}/${totalUsers}] 🔑 Logging in...`);
        const loginStartTime = Date.now();

        await page.goto(`${BASE_URL}/vui/a/vusmartmaps-app?redirect=dashboard&lte=now&gte=now-15m`, {
            waitUntil: 'networkidle',
            timeout: TIMEOUT
        });

        await page.getByRole(SELECTORS.username.role, { name: SELECTORS.username.name }).fill(user.username);
        await page.getByRole(SELECTORS.password.role, { name: SELECTORS.password.name }).fill(user.password);
        await page.getByRole(SELECTORS.signInButton.role, { name: SELECTORS.signInButton.name }).click();

        // Wait for dashboard to load
        await page.waitForURL('**/vusmartmaps-app**', { timeout: TIMEOUT });
        await page.waitForTimeout(2000);

        const loginTime = Date.now() - loginStartTime;
        console.log(`[User ${userIndex + 1}/${totalUsers}] ✅ Login successful (${loginTime}ms)`);

        // ========== NAVIGATE TO LOG ANALYTICS ==========
        console.log(`[User ${userIndex + 1}/${totalUsers}] 🗂️  Navigating to Log Analytics...`);
        const navStartTime = Date.now();

        await page.getByTestId(SELECTORS.toggleMenu.testId).click();
        await page.waitForTimeout(500);

        await page.getByRole(SELECTORS.observabilitySection.role, { name: SELECTORS.observabilitySection.name }).click();
        await page.waitForTimeout(500);

        await page.getByRole(SELECTORS.logAnalyticsLink.role, { name: SELECTORS.logAnalyticsLink.name }).click();
        await page.waitForTimeout(2000);

        // Select log source (Apachelogs)
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
                await page.waitForTimeout(3000); // Wait for results to fully render
                const resultsRenderTime = Date.now() - renderStartTime;

                const totalFilterTime = Date.now() - filterStartTime;
                const pageLoadTime = navTime; // Using navigation time as page load

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
                    test_id: testId
                });

                // Delete query to prepare for next filter
                await page.locator('.css-mwxb35-button.pillIconWrapper').first().click();
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
                    test_id: testId,
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
    const csvFile = path.join(__dirname, 'results', `log_analytics_results_${Date.now()}.csv`);
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
    console.log('🚀 Playwright Log Analytics Load Test\n');
    console.log('='.repeat(70));
    console.log(`📍 Base URL:      ${BASE_URL}`);
    console.log(`📁 User File:     ${USER_FILE}`);
    console.log(`📁 Filter File:   ${FILTER_FILE}`);
    console.log(`🗂️  Log Source:    ${LOG_SOURCE}`);
    console.log(`⏰ Time Range:    ${TIME_RANGE}`);
    console.log(`🖥️  Headless:      ${HEADLESS}`);
    console.log('='.repeat(70) + '\n');

    const users = loadUsers();
    const filters = loadFilters();

    console.log(`📋 Loaded ${users.length} user(s)`);
    console.log(`🔍 Loaded ${filters.length} filter(s)\n`);

    const startTime = Date.now();
    const testId = `${Date.now()}_${filters.length}filters`;

    // Run tests for all users sequentially (to avoid overwhelming the system)
    for (let i = 0; i < users.length; i++) {
        await testLogAnalytics(users[i], filters, i, users.length, testId);
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
