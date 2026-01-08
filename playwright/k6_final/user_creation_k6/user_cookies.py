import requests
import random
import argparse
import time
import urllib3
from datetime import datetime
from playwright.sync_api import sync_playwright
import os
from pathlib import Path
from playwright._impl._driver import compute_driver_executable
import sys

# === Fix for PyInstaller + Playwright ===
if getattr(sys, 'frozen', False):
    # running in PyInstaller bundle
    os.environ["PLAYWRIGHT_BROWSERS_PATH"] = str(Path(sys._MEIPASS) / "playwright-browsers")
else:
    # running normally
    os.environ["PLAYWRIGHT_BROWSERS_PATH"] = str(Path.home() / ".cache/ms-playwright")

# ===== CONFIGURATION =====
BASE_URL = "https://216.48.191.10"
KEYCLOAK_BASE = "https://216.48.191.10"
REALM = "vunet"

# Disable all insecure request warnings globally
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)
requests.packages.urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

TOKEN_URL = f"{KEYCLOAK_BASE}/realms/{REALM}/protocol/openid-connect/token"
ADMIN_URL = f"{KEYCLOAK_BASE}/admin/realms/{REALM}/users"

CLIENT_ID = "nairobi"
CLIENT_SECRET = "95z5sjMZLE6qQjRrVrVGtOge3r1k8p4a"
ADMIN_CLIENT_ID = "admin-cli"
ADMIN_USERNAME = "vunetadmin"
ADMIN_PASSWORD = "Qwerty@123"
COMMON_PASSWORD = "Password123!"

# ===== FUNCTIONS =====
def get_admin_token():
    """Get Keycloak admin access token."""
    data = {
        "client_id": ADMIN_CLIENT_ID,
        "username": ADMIN_USERNAME,
        "password": ADMIN_PASSWORD,
        "grant_type": "password"
    }
    headers = {"Content-Type": "application/x-www-form-urlencoded"}
    response = requests.post(TOKEN_URL, data=data, headers=headers, verify=False)
    response.raise_for_status()
    return response.json()["access_token"]


def create_user(admin_token, username):
    """Create a new user in Keycloak."""
    user_data = {
        "username": username,
        "email": f"{username}@vunetsystems.com",
        "enabled": True,
        "firstName": "Test",
        "lastName": "User",
        "credentials": [{
            "type": "password",
            "value": COMMON_PASSWORD,
            "temporary": False
        }]
    }
    headers = {
        "Authorization": f"Bearer {admin_token}",
        "Content-Type": "application/json"
    }
    response = requests.post(ADMIN_URL, json=user_data, headers=headers, verify=False)
    if response.status_code == 201:
        print(f"✅ User {username} created successfully.")
        return True
    else:
        print(f"❌ Failed to create user {username}: {response.status_code} - {response.text}")
        return False


def generate_username(prefix="load_user_"):
    """Generate a random username."""
    return f"{prefix}{random.randint(1, 10000)}"


def login_vusmartmaps_get_cookies(username):
    """Use Playwright to login to vuSmartMaps and get cookies."""
    try:
        with sync_playwright() as p:
            browser = p.chromium.launch(
                headless=True,
                args=["--no-sandbox", "--disable-dev-shm-usage"]
            )
            context = browser.new_context(ignore_https_errors=True)
            page = context.new_page()

            print(f"🔑 Logging in as {username}...")
            page.goto(f"{BASE_URL}/vui/login", wait_until="networkidle")

            page.fill("input[name=username]", username)
            page.fill("input[name=password]", COMMON_PASSWORD)
            page.click("button[type=submit]")

            try:
                page.wait_for_url(f"{BASE_URL}/vui/*", timeout=20000)
            except Exception as e:
                print(f"❌ Failed to login {username} to vuSmartMaps: {e}")
                browser.close()
                return None

            cookies = context.cookies()
            browser.close()
            return {c["name"]: c["value"] for c in cookies}
    except Exception as e:
        print(f"⚠️ Playwright error for {username}: {e}")
        return None


# ===== MAIN FUNCTION =====
def main(num_users):
    # Log start time
    current_time = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    with open("timeout.txt", "w") as timeout_file:
        timeout_file.write(f"Script started at: {current_time}\n")
        timeout_file.write(f"Target number of users: {num_users}\n\n")

    admin_token = get_admin_token()
    created_count = 0

    with open("user_cookies.txt", "w") as file:
        file.write("username,password,vunet_session,X-VuNet-HTTP-Info,grafana_session_expiry\n")

        for _ in range(num_users):
            username = generate_username()
            if not create_user(admin_token, username):
                continue

            time.sleep(2)  # allow user creation to propagate

            cookies = login_vusmartmaps_get_cookies(username)
            if not cookies:
                print(f"⚠️ Skipping {username}, failed to get cookies.")
                continue

            vunet_session = cookies.get("vunet_session", "")
            X_VuNet_HTTP_Info = cookies.get("X-VuNet-HTTP-Info", "")
            grafana_session_expiry = cookies.get("grafana_session_expiry", "")

            file.write(f"{username},{COMMON_PASSWORD},{vunet_session},{X_VuNet_HTTP_Info},{grafana_session_expiry}\n")
            print(f"✅ Cookies saved for {username}")
            created_count += 1

    final_message = (
        f"\n🎉 All {created_count}/{num_users} users created successfully "
        f"at {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}!\n"
    )
    print(final_message)

    with open("timeout.txt", "a") as timeout_file:
        timeout_file.write(final_message)


# ===== ENTRY POINT =====
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Create users and get vuSmartMaps cookies")
    parser.add_argument("num_users", type=int, help="Number of users to create")
    args = parser.parse_args()
    main(args.num_users)
