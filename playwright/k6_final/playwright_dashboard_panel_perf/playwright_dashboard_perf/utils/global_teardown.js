// utils/global_teardown.js
const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

module.exports = async function globalTeardown() {
  console.log('\n\n===== GLOBAL TEARDOWN STARTED =====');

  const RESULTS_DIR = path.join(__dirname, '../results');
  const TEMP_DIR = path.join(RESULTS_DIR, 'temp_partials');
  
  // 1. Check if temp dir exists and has files
  if (!fs.existsSync(TEMP_DIR)) {
    console.log('No temporary results found. Skipping aggregation.');
    return;
  }

  const files = fs.readdirSync(TEMP_DIR).filter(f => f.endsWith('.json'));
  if (files.length === 0) {
    console.log('No JSON files found in temp directory.');
    return;
  }

  console.log(`Aggregating ${files.length} partial result files...`);

  const aggregatedResults = [];

  // 2. Read and merge all partial JSONs
// Inside your global_teardown loop where you process files:
for (const file of files) {
    const filePath = path.join(TEMP_DIR, file);
    // Ignore the dashboard_meta.json file used for setup!
    if (file === 'dashboard_meta.json') continue; 

    try {
        const json = JSON.parse(fs.readFileSync(filePath, 'utf8'));
        // VALIDATION: Ensure the worker actually returned a valid panels array
        if (json && Array.isArray(json.panels)) {
            aggregatedResults.push(json);
        } else {
            console.warn(`Skipping malformed result file: ${file}`);
        }
    } catch (err) {
        console.error(`Failed to parse ${file}:`, err.message);
    }
}

  // 3. Generate final filename (using the dashboard slug from the first result if available)
  const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
  let slug = 'combined';
  if (aggregatedResults.length > 0 && aggregatedResults[0].dashboardName) {
    // Simple slugify of dashboard name just in case
    slug = aggregatedResults[0].dashboardName.replace(/[^a-zA-Z0-9]/g, '_').toLowerCase();
  }

  const finalLogPath = path.join(RESULTS_DIR, `concurrent_dashboard_run_${slug}_${timestamp}.json`);
  const finalCsvPath = finalLogPath.replace('.json', '.csv');

  // 4. Write the Combined JSON
  fs.writeFileSync(finalLogPath, JSON.stringify(aggregatedResults, null, 2));
  console.log(`\nFinal JSON Saved: ${finalLogPath}`);

  // 5. Run Post-Processing (CSV Conversion for logging)
  try {
    const converterScript = path.join(__dirname, 'convert_json_to_csv.js');

    console.log('Running CSV conversion for logging...');
    execSync(`node "${converterScript}" "${finalLogPath}"`, { stdio: 'inherit' });

    console.log('Post-processing complete.');

  } catch (e) {
    console.error('Post-processing failed during global teardown:', e.message);
  }

  // 6. Cleanup Temp Files
  try {
    files.forEach(f => fs.unlinkSync(path.join(TEMP_DIR, f)));
    fs.rmdirSync(TEMP_DIR);
    console.log('Temporary files cleaned up.');
  } catch (cleanupErr) {
    console.error('Warning: Failed to clean up temp files:', cleanupErr.message);
  }
  
  console.log('===== GLOBAL TEARDOWN COMPLETE =====\n');
};