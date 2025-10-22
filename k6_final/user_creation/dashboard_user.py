import requests
import random
import argparse
from playwright.sync_api import sync_playwright
import time

# ===== CONFIGURATION =====
BASE_URL = "https://164.52.213.158"
KEYCLOAK_BASE = "https://164.52.214.184"
REALM = "vunet"

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
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        # Ignore invalid SSL certs
        context = browser.new_context(ignore_https_errors=True)
        page = context.new_page()

        print(f"🔑 Logging in as {username}...")
        page.goto(f"{BASE_URL}/vui/login", wait_until="networkidle")

        # Fill login form (adjust selectors if needed)
        page.fill("input[name=username]", username)
        page.fill("input[name=password]", COMMON_PASSWORD)
        page.click("button[type=submit]")

        # Wait for dashboard to load
        try:
            page.wait_for_url(f"{BASE_URL}/vui/*", timeout=20000)
        except:
            print(f"❌ Failed to login {username} to vuSmartMaps")
            browser.close()
            return None

        cookies = context.cookies()
        browser.close()
        cookie_dict = {c["name"]: c["value"] for c in cookies}
        return cookie_dict


# ===== MAIN FUNCTION =====

def main(num_users):
    admin_token = get_admin_token()
    with open("user_cookies.txt", "w") as file:
        # Write header
        file.write("username,password,vunet_session,X-VuNet-HTTP-Info,grafana_session_expiry\n")

        for _ in range(num_users):
            username = generate_username()
            if not create_user(admin_token, username):
                continue

            # Small delay to allow user creation propagation
            time.sleep(2)

            cookies = login_vusmartmaps_get_cookies(username)
            if not cookies:
                print(f"⚠️ Skipping {username}, failed to get cookies.")
                continue

            vunet_session = cookies.get("vunet_session", "")
            X_VuNet_HTTP_Info = cookies.get("X-VuNet-HTTP-Info", "")
            grafana_session_expiry = cookies.get("grafana_session_expiry", "")

            file.write(f"{username},{COMMON_PASSWORD},{vunet_session},{X_VuNet_HTTP_Info},{grafana_session_expiry}\n")
            print(f"✅ Cookies saved for {username}")


# ===== ENTRY POINT =====
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Create users and get vuSmartMaps cookies")
    parser.add_argument("num_users", type=int, help="Number of users to create")
    args = parser.parse_args()
    main(args.num_users)

