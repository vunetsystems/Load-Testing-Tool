const BASE_URL = 'https://216.48.191.10'; // VuNet/Grafana base URL

/**
 * Fetch panelId → panelTitle mapping safely
 */
async function getPanelMap(page, dashboardUid) {
  const apiUrl = `${BASE_URL}/vui/api/dashboards/uid/${dashboardUid}`;

  // Load cookies from browser context
  const cookies = await page.context().cookies();
  const cookieHeader = cookies.map(c => `${c.name}=${c.value}`).join('; ');

  const headers = {
    'Accept': 'application/json',
    'Cookie': cookieHeader,
    'x-grafana-org-id': '1',
    'x-grafana-device-id': 'e95af4dda8279089e04e0d949be9de10', // optional but mirrors dashboard requests
    'User-Agent': 'Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:146.0) Gecko/20100101 Firefox/146.0',
    'Referer': `${BASE_URL}/vui/d/${dashboardUid}/mssql-overview-dashboard?orgId=1&refresh=30s`,
    'Origin': BASE_URL
  };

  let response;
  try {
    response = await page.request.get(apiUrl, { headers });
  } catch (err) {
    console.error('[panel_mapper] Failed request:', err.message);
    return {};
  }

  if (!response.ok()) {
    console.warn('[panel_mapper] Failed to fetch dashboard JSON, status:', response.status());
    return {};
  }

  const json = await response.json();

  // Handle multiple Grafana/VuNet response shapes
  const panels =
    json?.dashboard?.panels ||
    json?.dashboard?.dashboard?.panels ||
    json?.dashboard?.data?.panels ||
    json?.panels ||
    [];

  if (!Array.isArray(panels) || panels.length === 0) {
    console.warn(
      '[panel_mapper] No panels found in dashboard JSON. Falling back to Unknown titles.'
    );
    return {};
  }

  const map = {};

  // Recursively handle nested panels (rows, nested dashboards)
  const walkPanels = panelList => {
    for (const panel of panelList) {
      if (panel.id && panel.title) {
        map[String(panel.id)] = panel.title;
      }

      if (Array.isArray(panel.panels)) {
        walkPanels(panel.panels);
      }
    }
  };

  walkPanels(panels);

  return map;
}

module.exports = { getPanelMap };
