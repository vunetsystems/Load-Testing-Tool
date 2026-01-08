const { test } = require('@playwright/test');
const { chromium } = require('playwright');
const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const CONFIG_FILE = path.join(__dirname, '../config/test_config.json');
const config = JSON.parse(fs.readFileSync(CONFIG_FILE, 'utf8'));

const USERS_FILE = path.join(__dirname, '../config/users.json');
const users = JSON.parse(fs.readFileSync(USERS_FILE, 'utf8'));

const META_FILE = path.join(__dirname, '../results/temp_partials/dashboard_meta.json');
let dashboardMeta = {};
if (fs.existsSync(META_FILE)) {
    try {
        dashboardMeta = JSON.parse(fs.readFileSync(META_FILE, 'utf8'));
    } catch (e) {
        console.error("Could not parse dashboard metadata file.");
    }
}

const enabledDashboards = config.dashboards.filter(d => d.enabled);
const TEMP_RESULTS_DIR = path.join(__dirname, '../results/temp_partials');
if (!fs.existsSync(TEMP_RESULTS_DIR)) {
    fs.mkdirSync(TEMP_RESULTS_DIR, { recursive: true });
}

async function runDashboardSession(attempt, user, dashboardUrl, dashboardUid, dashboardName, targetPanelIds, panelMap) {
    const browser = await chromium.launch({ headless: true, ignoreHTTPSErrors: true });
    const context = await browser.newContext({ ignoreHTTPSErrors: true });
    if (user.cookies) await context.addCookies(user.cookies);
    const page = await context.newPage();

    // Standardize result structure to prevent CSV converter crashes
    const timestamp = new Date().toLocaleString('en-GB', {
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit'
}).replace(',', '');
    
    const result = {
        attempt,
        user: user.username,
        totalDashboardLoadTimeMs: 0,
        dashboardName,
        dashboardId: dashboardUid,
        dashboardStatus: 'FAILED',
        panels: [],
testName: `Grafana Performance - ${dashboardName} - ${config.timeRange} - ${config.concurrent_attempts} - ${timestamp}`    };

    try {
        console.log(`[Worker ${attempt}] [User: ${user.username}] Navigating to Dashboard...`);
        await page.goto(dashboardUrl, { waitUntil: 'load' });

        // Login check
        const isLoginPage = await page.locator('#username').isVisible().catch(() => false);
        if (isLoginPage || page.url().includes('login')) {
            await page.fill('#username', user.username);
            await page.fill('#password', user.password);
            await Promise.all([
                page.click('button[type="submit"]'),
                page.waitForURL(url => url.toString().includes(dashboardUid))
            ]);
        }

        // Select time range
        console.log(`[Worker ${attempt}] Setting time range to ${config.timeRange}...`);
        await page.locator('[data-testid="data-testid TimePicker Open Button"]').click();
        await page.waitForTimeout(500);
        await page.locator('label').filter({ hasText: config.timeRange }).click();
        // Click elsewhere to close dropdown
        await page.mouse.click(100, 100);
        await page.waitForTimeout(500);

        // Metrics Tracking
        const triggeredPanels = new Set();
        const panelMetrics = new Map();
        let reqCounter = 0;
        const allKnownIds = new Set(targetPanelIds || []);

        page.on('request', req => {
            if (req.url().includes('/vui/api/ds/query')) {
                const panelId = req.headers()['x-panel-id'] || 'unknown';
                triggeredPanels.add(panelId);
                
                if (!panelMetrics.has(panelId)) panelMetrics.set(panelId, { requests: [] });
                
                const requestId = `${attempt}-${++reqCounter}`;
                panelMetrics.get(panelId).requests.push({
                    requestId,
                    startTime: Date.now(),
                    endTime: null,
                    durationMs: null,
                    status: 'PENDING'
                });
                req._perfRequestId = requestId;
                req._perfPanelId = panelId;
            }
        });

        page.on('response', async res => {
            if (res.url().includes('/vui/api/ds/query')) {
                const req = res.request();
                const panelId = req._perfPanelId;
                const entry = panelMetrics.get(panelId)?.requests.find(r => r.requestId === req._perfRequestId);
                if (entry) {
                    entry.endTime = Date.now();
                    entry.durationMs = entry.endTime - entry.startTime;
                    entry.status = res.status();
                }
            }
        });

        // --- Improved Scroll Trigger Logic ---
        const dashboardStart = Date.now();
        console.log(`[Worker ${attempt}] Triggering ${allKnownIds.size} panels...`);

        // Focus
        await page.mouse.click(100, 100);
        await page.waitForTimeout(300);

        const viewportHeight = await page.evaluate(() => window.innerHeight);
        let totalHeight = await page.evaluate(() => document.body.scrollHeight);

        const maxScrollCycles = 10; // Increased cycles for thorough triggering
        let scrollCycle = 0;

        while (scrollCycle < maxScrollCycles) {
            console.log(`[Worker ${attempt}] Scroll cycle ${scrollCycle + 1}/${maxScrollCycles} | Triggered: ${triggeredPanels.size}/${allKnownIds.size} panels`);

            // Click to focus before scrolling
            await page.mouse.click(viewportHeight / 2, viewportHeight / 2);
            await page.waitForTimeout(200);

            // Scroll down using PageDown
            for (let i = 0; i < 25; i++) {
                await page.keyboard.press('PageDown');
                await page.waitForTimeout(400); // Wait for lazy loading
            }

            // Click again
            await page.mouse.click(viewportHeight / 2, viewportHeight / 2);
            await page.waitForTimeout(200);

            // Scroll up using PageUp
            for (let i = 0; i < 25; i++) {
                await page.keyboard.press('PageUp');
                await page.waitForTimeout(400); // Wait for lazy loading
            }

            // Update total height in case content expanded
            totalHeight = await page.evaluate(() => document.body.scrollHeight);

            // Check if all panels triggered
            if (allKnownIds.size > 0 && [...allKnownIds].every(id => triggeredPanels.has(id))) {
                console.log(`[Worker ${attempt}] SUCCESS: All ${allKnownIds.size} target panels triggered after ${scrollCycle + 1} cycles.`);
                break;
            }

            scrollCycle++;
        }

        if (triggeredPanels.size < allKnownIds.size) {
            console.log(`[Worker ${attempt}] Still missing ${allKnownIds.size - triggeredPanels.size} panels after ${maxScrollCycles} cycles.`);
        }

        result.totalDashboardLoadTimeMs = Date.now() - dashboardStart;
        result.dashboardStatus = 'SUCCESS';

        // Map Panel Titles
        panelMetrics.forEach((data, panelId) => {
            result.panels.push({
                panelId,
                title: panelMap[panelId] || (panelId === 'unknown' ? 'Global / Vars' : panelId),
                requestCount: data.requests.length,
                requests: data.requests
            });
        });

        // Log Missing IDs specifically
        const missing = [...allKnownIds].filter(id => !triggeredPanels.has(id));
        if (missing.length > 0) {
            console.log(`[Worker ${attempt}] Still missing panels: ${missing.join(', ')}`);
        }

    } catch (e) {
        console.error(`[Worker ${attempt}] Critical Error: ${e.message}`);
        result.dashboardStatus = 'ERROR';
    } finally {
        // Save result even on partial failure so CSV converter has data to work with
        const filename = `result_${dashboardName.replace(/\s+/g, '_')}_${attempt}_${Date.now()}.json`;
        fs.writeFileSync(path.join(TEMP_RESULTS_DIR, filename), JSON.stringify(result, null, 2));

        // Insert partial results to ClickHouse immediately
        try {
            const tempJson = path.join(TEMP_RESULTS_DIR, `partial_${dashboardName.replace(/\s+/g, '_')}_${attempt}_${Date.now()}.json`);
            fs.writeFileSync(tempJson, JSON.stringify([result], null, 2));
            const converterScript = path.join(__dirname, '../utils/convert_json_to_csv.js');
            execSync(`node "${converterScript}" "${tempJson}"`, { stdio: 'inherit' });
            const csvFile = tempJson.replace('.json', '.csv');
            const insertScript = path.join(__dirname, '../utils/insert_to_clickhouse.sh');
            execSync(`"${insertScript}" "${csvFile}"`, { stdio: 'inherit' });
            // Cleanup temp files
            fs.unlinkSync(tempJson);
            fs.unlinkSync(csvFile);
        } catch (insertErr) {
            console.error(`[Worker ${attempt}] Failed to insert partial results: ${insertErr.message}`);
        }

        await context.close();
        await browser.close();
    }
}

enabledDashboards.forEach(dashboard => {
    const dashboardUrl = `${config.base_urls.dashboard_ui}${dashboard.id}/${dashboard.slug}?orgId=1&refresh=30s`;
    const meta = dashboardMeta[dashboard.id] || { panelIds: [], panelMap: {} };
    const targetIds = meta.panelIds || [];

    for (let i = 0; i < config.concurrent_attempts; i++) {
        const attemptNum = i + 1;
        const currentUser = users[i % users.length];

        test(`Perf - ${dashboard.name} - Worker ${attemptNum}`, async () => {
            await runDashboardSession(
                attemptNum,
                currentUser,
                dashboardUrl,
                dashboard.id,
                dashboard.name,
                targetIds,
                meta.panelMap || {}
            );
        });
    }
});