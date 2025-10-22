import requests
import random
import argparse
import json

# Disable SSL warnings
requests.packages.urllib3.disable_warnings()

# ==== CONFIGURATION ====
BASE_URL = "https://164.52.213.158"
KEYCLOAK_BASE = "https://164.52.214.184"
REALM = "vunet"

TOKEN_URL = f"{KEYCLOAK_BASE}/realms/{REALM}/protocol/openid-connect/token"
ADMIN_URL = f"{KEYCLOAK_BASE}/admin/realms/{REALM}/users"

CLIENT_ID = "nairobi"
ADMIN_CLIENT_ID = "admin-cli"
ADMIN_USERNAME = "vunetadmin"
ADMIN_PASSWORD = "Qwerty@123"
COMMON_PASSWORD = "Password123!"
REDIRECT_URI = f"{BASE_URL}/vui/login/generic_oauth"


# ==== FUNCTIONS ====

def get_access_token():
    """Get admin access token from Keycloak."""
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


def create_user(access_token, username):
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
        "Authorization": f"Bearer {access_token}",
        "Content-Type": "application/json"
    }
    response = requests.post(ADMIN_URL, json=user_data, headers=headers, verify=False)
    if response.status_code == 201:
        print(f"✅ User {username} created successfully.")
    else:
        print(f"❌ Failed to create user {username}: {response.status_code} - {response.text}")


def generate_username(prefix="load_user_"):
    """Generate a random username."""
    return f"{prefix}{random.randint(1, 10000)}"


def get_user_tokens(username, password):
    """Log in using Keycloak's token endpoint."""
    data = {
        "client_id": "nairobi",
        "client_secret": "95z5sjMZLE6qQjRrVrVGtOge3r1k8p4a",
        "grant_type": "password",
        "username": username,
        "password": password,
        "scope": "openid email profile"
    }
    response = requests.post(TOKEN_URL, data=data, verify=False)
    if response.status_code != 200:
        print(f"❌ Failed to get token for {username}: {response.text}")
        return None

    tokens = response.json()
    print(f"🔑 Tokens fetched for {username}")
    return tokens


def main(num_users):
    """Main function to create users and fetch tokens."""
    admin_token = get_access_token()
    with open("user_tokens.txt", "w") as file:
        for _ in range(num_users):
            username = generate_username()
            create_user(admin_token, username)
            tokens = get_user_tokens(username, COMMON_PASSWORD)
            if not tokens:
                print(f"⚠️ Skipping {username}, failed to get tokens.")
                continue

            access_token = tokens.get("access_token")
            refresh_token = tokens.get("refresh_token")
            id_token = tokens.get("id_token")

            file.write(f"{username},{COMMON_PASSWORD},{access_token},{refresh_token},{id_token}\n")

            print(f"✅ Tokens saved for {username}")


# ==== ENTRY POINT ====
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Create users and get Keycloak tokens")
    parser.add_argument("num_users", type=int, help="Number of users to create")
    args = parser.parse_args()
    main(args.num_users)

