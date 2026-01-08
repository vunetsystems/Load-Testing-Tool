const fs = require('fs');
const path = require('path');

const COOKIE_FILE = path.join(__dirname, '../config/cookies.json');

async function loadCookies(context) {
  if (fs.existsSync(COOKIE_FILE)) {
    const cookies = JSON.parse(fs.readFileSync(COOKIE_FILE, 'utf8'));
    await context.addCookies(cookies);
    console.log('[cookie_loader] Loaded cookies:', cookies.map(c => c.name));
  }
}

async function saveCookies(context) {
  const cookies = await context.cookies();
  fs.writeFileSync(COOKIE_FILE, JSON.stringify(cookies, null, 2));
  console.log('[cookie_loader] Saved cookies:', cookies.map(c => c.name));
}

module.exports = { loadCookies, saveCookies };
