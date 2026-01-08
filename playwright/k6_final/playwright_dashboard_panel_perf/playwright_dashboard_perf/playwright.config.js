// // const { defineConfig } = require('@playwright/test');

// // module.exports = defineConfig({
// //   use: {
// //     headless: true,
// //     ignoreHTTPSErrors: true,
// //   },
// //   testDir: './tests',
// //   workers: 1, // Single test that handles parallelism internally
// // });




// const { defineConfig, devices } = require('@playwright/test');
// const fs = require('fs');
// const path = require('path');

// // Read your config to determine worker count
// const CONFIG_FILE = path.join(__dirname, './config/test_config.json');
// let configData = { concurrent_attempts: 1 };

// try {
//   if (fs.existsSync(CONFIG_FILE)) {
//     configData = JSON.parse(fs.readFileSync(CONFIG_FILE, 'utf8'));
//   }
// } catch (e) {
//   console.error('Error reading config file in playwright.config.js', e);
// }

// module.exports = defineConfig({
//   testDir: './tests',
//   /* Maximum time one test can run for. */
//   timeout: 60 * 1000, 
//   expect: {
//     timeout: 5000
//   },
//   /* Run tests in files in parallel */
//   fullyParallel: true,
//   /* Fail the build on CI if you accidentally left test.only in the source code. */
//   forbidOnly: !!process.env.CI,
//   /* Retry on CI only */
//   retries: 0,
  
//   /* KEY CHANGE: 
//      Set workers equal to concurrent_attempts. 
//      This ensures 1 Process = 1 Worker = 1 User Session.
//   */
//   workers: configData.concurrent_attempts,

//   /* Reporter to use. */
//   reporter: 'list',

//   /* Shared settings for all the projects below. */
//   use: {
//     headless: true,
//     ignoreHTTPSErrors: true,
//     trace: 'on-first-retry',
//   },

//   /* GLOBAL TEARDOWN:
//      Runs once after all workers have finished. 
//      This is where we aggregate JSONs and run CSV/Clickhouse scripts.
//   */
//   globalTeardown: require.resolve('./utils/global_teardown.js'),
// });



const { defineConfig } = require('@playwright/test');
const fs = require('fs');
const path = require('path');

const CONFIG_FILE = path.join(__dirname, './config/test_config.json');
let configData = { concurrent_attempts: 1 };
try {
  if (fs.existsSync(CONFIG_FILE)) {
    configData = JSON.parse(fs.readFileSync(CONFIG_FILE, 'utf8'));
  }
} catch (e) {}

module.exports = defineConfig({
  testDir: './tests',
  timeout: 300 * 1000,
  expect: { timeout: 5000 },
  fullyParallel: true, // Run tests in parallel
  workers: configData.concurrent_attempts, // 1 worker = 1 attempt
  reporter: 'list',
  use: {
    headless: true,
    ignoreHTTPSErrors: true,
  },
  
  // Point to the new Setup file
  globalSetup: require.resolve('./utils/global_setup.js'),
  
  // Existing Teardown
  globalTeardown: require.resolve('./utils/global_teardown.js'),
});