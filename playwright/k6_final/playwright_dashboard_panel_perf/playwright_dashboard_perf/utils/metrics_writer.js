const fs = require('fs');
const path = require('path');

function writeResults(result) {
  const date = new Date().toISOString().split('T')[0];
  const timestamp = new Date().toISOString().replace(/[:.]/g, '-');

  const jsonPath = path.join(__dirname, `../results/run_${date}_attempt_${result.attempt}_${timestamp}.json`);
  const csvPath = path.join(__dirname, `../results/run_${date}_attempt_${result.attempt}_${timestamp}.csv`);

  fs.writeFileSync(jsonPath, JSON.stringify(result, null, 2));

  const csvRows = [
    'panel_id,panel_title,queryCount,responseTimeMs'
  ];

  result.panels.forEach((data) => {
    csvRows.push(
      `${data.id},"${data.title}",${data.queryCount},${data.responseTimeMs}`
    );
  });

  fs.writeFileSync(csvPath, csvRows.join('\n'));
}

module.exports = { writeResults };
