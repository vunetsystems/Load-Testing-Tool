const fs = require('fs');
const path = require('path');

// Function to flatten the JSON data
function flattenData(data) {
    const rows = [];
    const panelMap = new Map();

    data.forEach(run => {
        const { attempt, user, totalDashboardLoadTimeMs, testName, dashboardName = 'Unknown Dashboard', dashboardStatus = 'UNKNOWN', panels } = run;
        panels.forEach(panel => {
            const { panelId, title, requests } = panel;
            const key = `${attempt}-${user}-${totalDashboardLoadTimeMs}-${testName}-${dashboardName}-${dashboardStatus}-${panelId}-${title}`;
            if (!panelMap.has(key)) {
                panelMap.set(key, []);
            }
            requests.forEach(request => {
                let { requestId, startTime, endTime, durationMs, status } = request;
                // Include all requests, set defaults for incomplete
                if (endTime === null) {
                    endTime = startTime;
                    durationMs = 0;
                    status = 0;
                }
                panelMap.get(key).push({
                    attempt,
                    user,
                    totalDashboardLoadTimeMs,
                    testName,
                    dashboardName,
                    dashboardStatus,
                    panelId,
                    title,
                    requestId,
                    startTime,
                    endTime,
                    durationMs,
                    status
                });
            });
        });
    });

    // Now, for each panel, select one request: prefer status 200, else any
    panelMap.forEach(requests => {
        if (requests.length === 0) return;
        let selected = requests.find(r => r.status === 200);
        if (!selected) {
            selected = requests[0];
        }
        rows.push(selected);
    });

    return rows;
}

// Function to convert to CSV
function toCSV(rows) {
    if (rows.length === 0) return '';
    const headers = Object.keys(rows[0]);
    const csvRows = rows.map(row => headers.map(header => JSON.stringify(row[header])).join(','));
    return [headers.join(','), ...csvRows].join('\n');
}

// Main function
function main() {
    const inputFile = process.argv[2];
    if (!inputFile) {
        console.error('Usage: node convert_json_to_csv.js <input_json_file>');
        process.exit(1);
    }

    const outputFile = inputFile.replace('.json', '.csv');

    try {
        const jsonData = JSON.parse(fs.readFileSync(inputFile, 'utf8'));
        const flattened = flattenData(jsonData);
        const csv = toCSV(flattened);
        fs.writeFileSync(outputFile, csv);
        console.log(`CSV file created: ${outputFile}`);
    } catch (error) {
        console.error('Error:', error.message);
        process.exit(1);
    }
}

main();