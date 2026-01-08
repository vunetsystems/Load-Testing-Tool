// utils/global_setup.js
const { chromium } = require('@playwright/test');
const fs = require('fs');
const path = require('path');

module.exports = async function globalSetup(config) {
  console.log('\n===== GLOBAL SETUP: Pre-fetching Dashboard Metadata =====');

  const CONFIG_FILE = path.join(__dirname, '../config/test_config.json');
  if (!fs.existsSync(CONFIG_FILE)) {
    throw new Error('Config file not found!');
  }
  const configData = JSON.parse(fs.readFileSync(CONFIG_FILE, 'utf8'));
  
  const USERS_FILE = path.join(__dirname, '../config/users.json');
  const users = JSON.parse(fs.readFileSync(USERS_FILE, 'utf8'));
  const adminUser = users[0]; // Use first user for fetching metadata

  const enabledDashboards = configData.dashboards.filter(d => d.enabled);
  const metadataMap = {};

  const browser = await chromium.launch({ headless: true, ignoreHTTPSErrors: true });
  
  try {
    const context = await browser.newContext({ ignoreHTTPSErrors: true });
    // Add cookies if they exist to skip login, otherwise login
    if (adminUser.cookies) {
        await context.addCookies(adminUser.cookies);
    }

    const page = await context.newPage();
    const dashboardUiUrl = configData.base_urls.dashboard_ui; 

    // 1. Ensure we are logged in (UI Login fallback)
    console.log(`[Setup] Verifying authentication as ${adminUser.username}...`);
    try {
        await page.goto(dashboardUiUrl, { waitUntil: 'domcontentloaded', timeout: 15000 });
        const isLoginPage = await page.locator('#username').isVisible().catch(() => false);
        
        if (isLoginPage || page.url().includes('login') || page.url().includes('openid-connect')) {
            console.log('[Setup] Logging in...');
            await page.fill('#username', adminUser.username);
            await page.fill('#password', adminUser.password);
            await page.click('button[type="submit"]');
            await page.waitForURL(url => !url.toString().includes('login'), { timeout: 15000 });
        }
    } catch (e) {
        console.log('[Setup] Login check skipped or failed (might already be authenticated via cookies). Continuing...');
    }

    // 2. Fetch Metadata for each dashboard via API
    const request = context.request;

    for (const dashboard of enabledDashboards) {
      console.log(`[Setup] Fetching metadata for: ${dashboard.name} (${dashboard.id})`);
      const apiUrl = `${configData.base_urls.dashboard_api}${dashboard.id}`;
      
      const response = await request.get(apiUrl);
      if (response.ok()) {
        const json = await response.json();

        // Extract Panel IDs that are actual data queries (ignore text/rows)
        const panelIds = json.dashboard.panels
          .filter(p => p.type !== 'text' && p.type !== 'row' && p.targets && p.targets.length > 0)
          .map(p => p.id.toString());

        // Extract panel ID → title mapping
        const panels = json?.dashboard?.panels || [];
        const panelMap = {};

        const walkPanels = panelList => {
          for (const panel of panelList) {
            if (panel.id && panel.title) {
              panelMap[String(panel.id)] = panel.title;
            }
            if (Array.isArray(panel.panels)) {
              walkPanels(panel.panels);
            }
          }
        };
        walkPanels(panels);

        metadataMap[dashboard.id] = { panelIds, panelMap };
        console.log(`   -> Found ${panelIds.length} data panels.`);
      } else {
        console.warn(`   -> Failed to fetch (Status: ${response.status()}). Workers will rely on auto-discovery.`);
        metadataMap[dashboard.id] = { panelIds: [], panelMap: {} };
      }
    }

    // 3. Save metadata to temp file for workers to read
    const META_DIR = path.join(__dirname, '../results/temp_partials');
    if (!fs.existsSync(META_DIR)) fs.mkdirSync(META_DIR, { recursive: true });
    
    fs.writeFileSync(
      path.join(META_DIR, 'dashboard_meta.json'), 
      JSON.stringify(metadataMap, null, 2)
    );
    console.log('===== GLOBAL SETUP COMPLETE =====\n');

  } catch (e) {
    console.error('Global Setup failed:', e);
    // Don't kill process, let tests try to run even if setup failed
  } finally {
    await browser.close();
  }
};