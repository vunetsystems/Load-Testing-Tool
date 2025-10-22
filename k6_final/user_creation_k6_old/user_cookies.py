import requests
import re
import random
import string
import argparse
import json
from http.cookiejar import MozillaCookieJar

# Disable SSL warnings
requests.packages.urllib3.disable_warnings()

BASE_URL = "https://164.52.213.158"
#LOGIN_URL = f"{BASE_URL}/vui/login/generic_oauth"
LOGIN_URL = f"{BASE_URL}/realms/vunet/protocol/openid-connect/auth?client_id=nairobi&code_challenge=o58ieHwj_oqQjmFyi12wqGZ2giRoxJ7no_4jrtY-2Nk&code_challenge_method=S256&redirect_uri=https%3A%2F%2F164.52.213.158%2Fvui%2Flogin%2Fgeneric_oauth&response_type=code&scope=openid+email+offline_access&state=vtSU2OJckth9eIkrZV8Du3OfJeGAWJv6yM87TkMvAmY%3D"
KEYCLOAK_URL = "https://164.52.214.184/realms/vunet/protocol/openid-connect/token"
ADMIN_URL = "https://164.52.214.184/admin/realms/vunet/users"
CLIENT_ID = "admin-cli"
ADMIN_USERNAME = "vunetadmin"
ADMIN_PASSWORD = "Qwerty@123"
COMMON_PASSWORD = "Password123!"

# Function to get Keycloak access token
def get_access_token():
    data = {
        "client_id": "nairobi",
        "client_secret": "95z5sjMZLE6qQjRrVrVGtOge3r1k8p4a",
        "username": ADMIN_USERNAME,
        "password": ADMIN_PASSWORD,
        "grant_type": "password"
    }
    headers = {"Content-Type": "application/x-www-form-urlencoded"}
    response = requests.post(KEYCLOAK_URL, data=data, headers=headers, verify=False)
    response.raise_for_status()
    return response.json()['access_token']
    print(response.json())

# Function to create a new user
def create_user(access_token, username):
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
    headers = {"Authorization": f"Bearer {access_token}", "Content-Type": "application/json"}
    response = requests.post(ADMIN_URL, json=user_data, headers=headers, verify=False)
    if response.status_code == 201:
        print(f"User {username} created successfully.")
    else:
        print(f"Failed to create user {username}: {response.status_code} - {response.text}")

# Function to generate a random username
def generate_username(prefix="load_user_"):
    return f"{prefix}{random.randint(1, 10000)}"

# Function to extract cookies
def extract_cookies(username, password):
    session = requests.Session()
    session.cookies = MozillaCookieJar()
    
    print(f"Logging in as {username}...")
    response = session.get(LOGIN_URL, verify=False, allow_redirects=False)
    redirect_url = response.headers.get("Location")
    
    if not redirect_url:
        print("Failed to get redirect URL")
        return None
    
    login_page = session.get(f"{BASE_URL}{redirect_url}", verify=False)
    login_html = login_page.text
    
    
    match = re.search(r'action="([^"]+)"', login_html)
    form_action = match.group(1) if match else None
    
    if not form_action:
        print("Failed to extract form action URL")
        return None
    
    data = {"username": username, "password": password}
    headers = {"User-Agent": "Mozilla/5.0"}
    auth_response = session.post(form_action, data=data, headers=headers, verify=False, allow_redirects=False)
    
    
   # final_redirect = auth_response.headers.get("Location")
   # if not final_redirect:
   #     print("Failed to get final redirect URL")
   #     return None
    
    session.get(LOGIN_URL, verify=False)
    cookies_dict = {cookie.name: cookie.value for cookie in session.cookies}
    return cookies_dict

# Main function to create users and extract cookies
def main(num_users):
    access_token = get_access_token()
    with open("user_cookies.txt", "w") as file:
        for _ in range(num_users):
            username = generate_username()
            create_user(access_token, username)
            cookies = extract_cookies(username, COMMON_PASSWORD)
            if not cookies:
                print(f"Skipping {username}, failed to extract cookies.")
                continue  # Skip this user and move on

            vunet_session = cookies.get('vunet_session')
            X_VuNet_HTTP_Info = cookies.get('X-VuNet-HTTP-Info')
            grafana_session_expiry = cookies.get('grafana_session_expiry')

            if vunet_session and X_VuNet_HTTP_Info and grafana_session_expiry:
                file.write(f"{username},{COMMON_PASSWORD},{vunet_session},{X_VuNet_HTTP_Info},{grafana_session_expiry}\n")
            else:
                print(f"User {username} cookies missing required keys: {cookies}")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Create users and extract cookies")
    parser.add_argument("num_users", type=int, help="Number of users to create")
    args = parser.parse_args()
    main(args.num_users)

