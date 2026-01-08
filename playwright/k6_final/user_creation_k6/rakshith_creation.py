import requests
import random
import argparse
import time
import urllib3
from datetime import datetime
from playwright.sync_api import sync_playwright
import os
from pathlib import Path
import sys

# === Fix for PyInstaller + Playwright ===
if getattr(sys, 'frozen', False):
    os.environ["PLAYWRIGHT_BROWSERS_PATH"] = str(Path(sys._MEIPASS) / "playwright-browsers")
else:
    os.environ["PLAYWRIGHT_BROWSERS_PATH"] = str(Path.home() / ".cache/ms-playwright")

# ===== CONFIGURATION =====
BASE_URL = "https://216.48.191.10"
KEYCLOAK_BASE = "https://216.48.191.10"
REALM = "vunet"

GROUP_NAME = "load_test"

ADMIN_CLIENT_ID = "admin-cli"
ADMIN_USERNAME = "vunetadmin"
ADMIN_PASSWORD = "Qwerty@123"
COMMON_PASSWORD = "Password123!"

# Disable SSL warnings
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)
requests.packages.urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

TOKEN_URL = f"{KEYCLOAK_BASE}/realms/{REALM}/protocol/openid-connect/token"
USERS_URL = f"{KEYCLOAK_BASE}/admin/realms/{REALM}/users"
GROUPS_URL = f"{KEYCLOAK_BASE}/admin/realms/{REALM}/groups"

# ===== FUNCTIONS =====
def get_admin_token():
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


def get_group_id(admin_token, group_name):
    headers = {"Authorization": f"Bearer {admin_token}"}
    response = requests.get(GROUPS_URL, headers=headers, verify=False)
    response.raise_for_status()

    for group in response.json():
        if group["name"] == group_name:
            return group["id"]

    raise Exception(f"❌ Group '{group_name}' not found")


def create_user(admin_token, username):
    """Create user with required attributes"""
    user_data = {
        "username": username,
        "email": f"{username}@vunetsystems.com",
        "enabled": True,
        "firstName": "Test",
        "lastName": "User",
        "attributes": {
            "tenant_id": ["1"],
            "display_duration": ["0"],
            "modified_by": ["vunetadmin"],
            "data_access_role": ["175f4022-9281-4905-a027-7a25db03e782"],
            "matching_keywords": []
        },
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

    response = requests.post(USERS_URL, json=user_data, headers=headers, verify=False)

    if response.status_code == 201:
        user_id = response.headers["Location"].split("/")[-1]
        print(f"✅ User {username} created")
        return user_id
    else:
        print(f"❌ Failed to create user {username}: {response.status_code} - {response.text}")
        return None


def add_user_to_group(admin_token, user_id, group_id):
    headers = {"Authorization": f"Bearer {admin_token}"}
    url = f"{USERS_URL}/{user_id}/groups/{group_id}"

    response = requests.put(url, headers=headers, verify=False)

    if response.status_code in (204, 201):
        print(f"👥 User added to group '{GROUP_NAME}'")
    else:
        print(f"❌ Failed to add user to group: {response.status_code} - {response.text}")


def generate_username(prefix="load_user_"):
    return f"{prefix}{random.randint(1000, 99999)}"


def login_vusmartmaps_get_cookies(username):
    try:
        with sync_playwright() as p:
            browser = p.chromium.launch(
                headless=True,
                args=["--no-sandbox", "--disable-dev-shm-usage"]
            )
            context = browser.new_context(ignore_https_errors=True)
            page = context.new_page()

            print(f"🔑 Logging in as {username}")
            page.goto(f"{BASE_URL}/vui/login", wait_until="networkidle")

            page.fill("input[name=username]", username)
            page.fill("input[name=password]", COMMON_PASSWORD)
            page.click("button[type=submit]")

            page.wait_for_url(f"{BASE_URL}/vui/*", timeout=20000)

            cookies = context.cookies()
            browser.close()

            return {c["name"]: c["value"] for c in cookies}

    except Exception as e:
        print(f"⚠️ Login failed for {username}: {e}")
        return None


# ===== MAIN =====
def main(num_users):
    start_time = datetime.now().strftime("%Y-%m-%d %H:%M:%S")

    with open("timeout.txt", "w") as f:
        f.write(f"Script started at: {start_time}\n")
        f.write(f"Target users: {num_users}\n\n")

    admin_token = get_admin_token()
    group_id = get_group_id(admin_token, GROUP_NAME)

    created_count = 0

    with open("user_cookies.txt", "w") as file:
        file.write("username,password,vunet_session,X-VuNet-HTTP-Info,grafana_session_expiry\n")

        for _ in range(num_users):
            username = generate_username()

            user_id = create_user(admin_token, username)
            if not user_id:
                continue

            add_user_to_group(admin_token, user_id, group_id)

            time.sleep(2)

            cookies = login_vusmartmaps_get_cookies(username)
            if not cookies:
                continue

            file.write(
                f"{username},{COMMON_PASSWORD},"
                f"{cookies.get('vunet_session','')},"
                f"{cookies.get('X-VuNet-HTTP-Info','')},"
                f"{cookies.get('grafana_session_expiry','')}\n"
            )

            print(f"🍪 Cookies saved for {username}")
            created_count += 1

    end_msg = (
        f"\n🎉 {created_count}/{num_users} users created with attributes "
        f"and added to '{GROUP_NAME}' at {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n"
    )

    print(end_msg)
    with open("timeout.txt", "a") as f:
        f.write(end_msg)


# ===== ENTRY POINT =====
if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="Create Keycloak users with attributes, add to group, and fetch vuSmartMaps cookies"
    )
    parser.add_argument("num_users", type=int, help="Number of users to create")
    args = parser.parse_args()
    main(args.num_users)
