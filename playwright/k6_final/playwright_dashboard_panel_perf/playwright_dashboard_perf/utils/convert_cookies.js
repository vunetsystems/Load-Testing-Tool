const fs = require('fs');
const path = require('path');

const inputFile = path.join(__dirname, '../config/cookies.txt');
const outputFile = path.join(__dirname, '../config/users.json');

function convertCookiesToJson() {
  const content = fs.readFileSync(inputFile, 'utf8');
  const lines = content.trim().split('\n');
  const headers = lines[0].split(',');

  const users = [];

  for (let i = 1; i < lines.length; i++) {
    const values = lines[i].split(',').map(v => v.trim());
    const expiry = parseInt(values[4]);
    const user = {
      username: values[0],
      password: values[1],
      cookies: [
        {
          name: 'vunet_session',
          value: values[2],
          domain: '216.48.191.10',
          path: '/',
          httpOnly: true,
          secure: true
        },
        {
          name: 'X-VuNet-HTTP-Info',
          value: values[3],
          domain: '216.48.191.10',
          path: '/',
          httpOnly: true,
          secure: true
        },
        {
          name: 'grafana_session_expiry',
          value: values[4],
          domain: '216.48.191.10',
          path: '/',
          httpOnly: true,
          secure: true,
          expires: expiry
        }
      ]
    };
    users.push(user);
  }

  fs.writeFileSync(outputFile, JSON.stringify(users, null, 2));
  console.log(`Converted ${users.length} users to ${outputFile}`);
}

convertCookiesToJson();