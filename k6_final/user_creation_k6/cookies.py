import requests
import re
from http.cookiejar import MozillaCookieJar

# Disable SSL warnings
requests.packages.urllib3.disable_warnings()

BASE_URL = "https://164.52.214.184"
LOGIN_URL = f"{BASE_URL}/vui/login/generic_oauth"

# Step 1: Get the initial redirect URL
print("Step 1: Getting initial redirect URL...")
response = requests.get(LOGIN_URL, verify=False, allow_redirects=False)
redirect_url = response.headers.get("Location")

if not redirect_url:
    print("Failed to get redirect URL")
    exit(1)

print(f"Redirect URL: {BASE_URL}{redirect_url}")

# Step 2: Follow the redirect to get the login page
print("\nStep 2: Following redirect to get login page...")
session = requests.Session()
session.cookies = MozillaCookieJar("cookies.txt")
login_page = session.get(f"{BASE_URL}{redirect_url}", verify=False)
login_html = login_page.text

# Step 3: Extract the form action URL
print("\nStep 3: Extracting form action URL...")
match = re.search(r'action="([^"]+)"', login_html)
form_action = match.group(1) if match else None

if not form_action:
    print("Failed to extract form action URL")
    exit(1)

print(f"Form action URL: {form_action}")

# Step 4: Submit login credentials
print("\nStep 4: Submitting login credentials...")
data = {
    "username": "load_user_13293",
    "password": "Password123!"
}
headers = {
    "User-Agent": "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:136.0) Gecko/20100101 Firefox/136.0",
    "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
    "Content-Type": "application/x-www-form-urlencoded"
}
auth_response = session.post(form_action, data=data, headers=headers, verify=False, allow_redirects=False)

# Step 5: Extract the final redirect URL
print("\nStep 5: Extracting final redirect URL...")
final_redirect = auth_response.headers.get("Location")

if not final_redirect:
    print("Failed to get final redirect URL")
    print("Authentication response:", auth_response.text)
    exit(1)

print(f"Final redirect URL: {final_redirect}")

# Step 6: Store oauth_code_verifier and oauth_state
print("\nStep 6: Storing oauth_code_verifier and oauth_state from cookies...")
session.get(LOGIN_URL, verify=False)
session.cookies.save()

# Step 7: Extract required cookies
print("\nStep 7: Extract required cookies for API request")
cookies_dict = {cookie.name: cookie.value for cookie in session.cookies}

oauth_code_verifier = cookies_dict.get("oauth_code_verifier")
oauth_state = cookies_dict.get("oauth_state")
vunet_session = cookies_dict.get("vunet_session")
x_vunet_http_info = cookies_dict.get("X-VuNet-HTTP-Info")
grafana_session_expiry = cookies_dict.get("grafana_session_expiry")

print(f"oauth_code_verifier: {oauth_code_verifier}")
print(f"oauth_state: {oauth_state}")
print(f"vunet_session: {vunet_session}")
print(f"X-VuNet-HTTP-Info: {x_vunet_http_info}")
print(f"grafana_session_expiry: {grafana_session_expiry}")

