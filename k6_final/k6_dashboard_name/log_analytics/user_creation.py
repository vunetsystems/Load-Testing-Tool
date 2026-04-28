import requests
import random
import argparse
from playwright.sync_api import sync_playwright
import time

# ===== CONFIGURATION =====
BASE_URL = "https://216.48.191.10"
KEYCLOAK_BASE = "https://216.48.191.10"
REALM = "vunet"

TOKEN_URL = f"{KEYCLOAK_BASE}/realms/{REALM}/protocol/openid-connect/token"
ADMIN_URL = f"{KEYCLOAK_BASE}/admin/realms/{REALM}/users"
GROUPS_URL = f"{KEYCLOAK_BASE}/admin/realms/{REALM}/groups"

CLIENT_ID = "nairobi"
CLIENT_SECRET = "95z5sjMZLE6qQjRrVrVGtOge3r1k8p4a"
ADMIN_CLIENT_ID = "admin-cli"
ADMIN_USERNAME = "vunetadmin"
ADMIN_PASSWORD = "Qwerty@123"
COMMON_PASSWORD = "Password123!"

GROUP_NAME = "load_test"


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


def get_group_id(admin_token, group_name):
    headers = {"Authorization": f"Bearer {admin_token}"}
    response = requests.get(GROUPS_URL, headers=headers, verify=False)
    response.raise_for_status()

    for group in response.json():
        if group["name"] == group_name:
            return group["id"]

    raise Exception(f"❌ Group '{group_name}' not found")


def add_user_to_group(admin_token, user_id, group_id):
    headers = {"Authorization": f"Bearer {admin_token}"}
    url = f"{ADMIN_URL}/{user_id}/groups/{group_id}"
    response = requests.put(url, headers=headers, verify=False)

    if response.status_code in (204, 201):
        print(f"👥 User added to group '{GROUP_NAME}'")
    else:
        print(f"❌ Failed to add user to group: {response.status_code} - {response.text}")


def create_user(admin_token, username):
    """Create a new user in Keycloak with attributes."""
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

    response = requests.post(ADMIN_URL, json=user_data, headers=headers, verify=False)

    if response.status_code == 201:
        user_id = response.headers["Location"].split("/")[-1]
        print(f"✅ User {username} created successfully.")
        return user_id
    else:
        print(f"❌ Failed to create user {username}: {response.status_code} - {response.text}")
        return None


def generate_username(prefix="load_user_"):
    """Generate a random username."""
    return f"{prefix}{random.randint(1, 10000)}"


def get_user_auth_token(username):
    """Get user auth token from Keycloak."""
    data = {
        "client_id": CLIENT_ID,
        "client_secret": CLIENT_SECRET,
        "username": username,
        "password": COMMON_PASSWORD,
        "grant_type": "password"
    }
    headers = {"Content-Type": "application/x-www-form-urlencoded"}
    response = requests.post(TOKEN_URL, data=data, headers=headers, verify=False)
    if response.status_code == 200:
        token = response.json()["access_token"]
        print(f"🔐 Auth token retrieved for {username}")
        return token
    else:
        print(f"⚠️ Failed to get auth token for {username}: {response.status_code} - {response.text}")
        return ""


def login_vusmartmaps_get_cookies(username):
    """Use Playwright to login to vuSmartMaps and get cookies."""
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        context = browser.new_context(ignore_https_errors=True)
        page = context.new_page()

        print(f"🔑 Logging in as {username}...")
        page.goto(f"{BASE_URL}/vui/login", wait_until="networkidle")

        page.fill("input[name=username]", username)
        page.fill("input[name=password]", COMMON_PASSWORD)
        page.click("button[type=submit]")

        try:
            page.wait_for_url(f"{BASE_URL}/vui/*", timeout=20000)
        except:
            print(f"❌ Failed to login {username} to vuSmartMaps")
            browser.close()
            return None

        cookies = context.cookies()
        browser.close()
        return {c["name"]: c["value"] for c in cookies}


# ===== MAIN FUNCTION =====

def main(num_users):
    admin_token = get_admin_token()
    group_id = get_group_id(admin_token, GROUP_NAME)

    with open("user_cookies_module.txt", "w") as file:
        file.write(
            "username,password,auth_token,vunet_session,"
            "X-VuNet-HTTP-Info,grafana_session_expiry\n"
        )

        for _ in range(num_users):
            username = generate_username()
            user_id = create_user(admin_token, username)
            if not user_id:
                continue

            add_user_to_group(admin_token, user_id, group_id)
            time.sleep(2)

            auth_token = get_user_auth_token(username)
            cookies = login_vusmartmaps_get_cookies(username)
            if not cookies:
                print(f"⚠️ Skipping {username}, failed to get cookies.")
                continue

            file.write(
                f"{username},{COMMON_PASSWORD},{auth_token},"
                f"{cookies.get('vunet_session','')},"
                f"{cookies.get('X-VuNet-HTTP-Info','')},"
                f"{cookies.get('grafana_session_expiry','')}\n"
            )

            print(f"✅ Saved cookies & token for {username}")


# ===== ENTRY POINT =====
if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="Create users and get vuSmartMaps cookies + auth token"
    )
    parser.add_argument("num_users", type=int, help="Number of users to create")
    args = parser.parse_args()
    main(args.num_users)
