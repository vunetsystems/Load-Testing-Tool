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
LOGIN_URL = f"{BASE_URL}/vui/login/generic_oauth"
KEYCLOAK_URL = "https://164.52.213.158/realms/vunet/protocol/openid-connect/token"
ADMIN_URL = "https://164.52.213.158/admin/realms/vunet/users"
CLIENT_ID = "admin-cli"
ADMIN_USERNAME = "vunetadmin"
ADMIN_PASSWORD = "Qwerty@123"
COMMON_PASSWORD = "Password123!"

def get_access_token():
    data = {
        "client_id": CLIENT_ID,
        "username": ADMIN_USERNAME,
        "password": ADMIN_PASSWORD,
        "grant_type": "password"
    }
    headers = {"Content-Type": "application/x-www-form-urlencoded"}
    response = requests.post(KEYCLOAK_URL, data=data, headers=headers, verify=False)
    response.raise_for_status()
    return response.json()['access_token']

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

def get_user_id(access_token, username):
    url = f"{ADMIN_URL}?username={username}"
    headers = {"Authorization": f"Bearer {access_token}", "Content-Type": "application/json"}
    response = requests.get(url, headers=headers, verify=False)
    
    if response.status_code == 200 and response.json():
        user_id = response.json()[0]['id']
        print(f"Retrieved User ID: {user_id} for Username: {username}")
        return user_id
    else:
        print(f"Failed to get user ID for {username}: {response.status_code} - {response.text}")
        return None

def generate_username(prefix="load_user_"):
    return f"{prefix}{random.randint(1, 10000)}"

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
    
    final_redirect = auth_response.headers.get("Location")
    if not final_redirect:
        print("Failed to get final redirect URL")
        return None
    
    session.get(LOGIN_URL, verify=False)
    cookies_dict = {cookie.name: cookie.value for cookie in session.cookies}
    return cookies_dict

def add_user_to_group(access_token, user_id, group_name="load_test"):
    url = f"{BASE_URL}/admin/realms/vunet/groups"
    headers = {
        "Authorization": f"Bearer {access_token}",
        "Content-Type": "application/json"
    }
    response = requests.get(url, headers=headers, verify=False)
    if response.status_code == 200:
        groups = response.json()
        group_id = next((group['id'] for group in groups if group['name'] == group_name), None)
        print(f"Retrieved Group ID: {group_id} for Group Name: {group_name}")
        
        if group_id:
            url = f"{BASE_URL}/admin/realms/vunet/users/{user_id}/groups/{group_id}"
            response = requests.put(url, headers=headers, verify=False)
            if response.status_code == 204:
                print(f"User {user_id} added to group {group_name}.")
            else:
                print(f"Error adding user {user_id} to group {group_name}: {response.text}")
        else:
            print(f"Group {group_name} not found.")
    else:
        print(f"Error fetching groups: {response.status_code} - {response.text}")

def main(num_users):
    access_token = get_access_token()
    with open("user_cookies1.txt", "a") as file:
        for _ in range(num_users):
            username = generate_username()
            create_user(access_token, username)
            user_id = get_user_id(access_token, username)
            if user_id:
                add_user_to_group(access_token, user_id)
            cookies = extract_cookies(username, COMMON_PASSWORD)
            if cookies:
                vunet_session = cookies.get('vunet_session', '')
                X_VuNet_HTTP_Info = cookies.get('X-VuNet-HTTP-Info', '')
                grafana_session_expiry = cookies.get('grafana_session_expiry', '')
                file.write(f"{username},{COMMON_PASSWORD},{access_token},{vunet_session},{X_VuNet_HTTP_Info},{grafana_session_expiry}\n")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Create users and extract cookies")
    parser.add_argument("num_users", type=int, help="Number of users to create")
    args = parser.parse_args()
    main(args.num_users)

