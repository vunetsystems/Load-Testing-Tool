package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	sessionStore *sessions.CookieStore
)

func getOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     os.Getenv("OAUTH_CLIENT_ID"),
		ClientSecret: os.Getenv("OAUTH_CLIENT_SECRET"),
		RedirectURL:  "http://" + os.Getenv("APP_PORT") + "/auth/callback",
		Scopes:       []string{"openid", "profile", "email"},
		Endpoint:     google.Endpoint,
	}
}

// HandleAuthLogin redirects to Google OAuth
func handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	log.Printf("DEBUG: handleAuthLogin called from %s", r.Referer())
	oauthConfig := getOAuthConfig()
	log.Printf("DEBUG: oauthConfig.ClientID: %s", maskSecret(oauthConfig.ClientID))
	log.Printf("DEBUG: oauthConfig.ClientSecret: %s", maskSecret(oauthConfig.ClientSecret))
	log.Printf("DEBUG: oauthConfig.RedirectURL: %s", oauthConfig.RedirectURL)
	url := oauthConfig.AuthCodeURL("state", oauth2.AccessTypeOffline)
	log.Printf("DEBUG: OAuth URL: %s", url)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// HandleAuthCallback processes the OAuth response
func handleAuthCallback(w http.ResponseWriter, r *http.Request) {
	log.Printf("DEBUG: handleAuthCallback called")
	code := r.URL.Query().Get("code")
	if code == "" {
		log.Printf("DEBUG: No code in request")
		http.Error(w, "No code in request", http.StatusBadRequest)
		return
	}
	log.Printf("DEBUG: Received OAuth code")

	oauthConfig := getOAuthConfig()
	token, err := oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
		return
	}

	// Fetch user info
	client := oauthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var userInfo struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		http.Error(w, "Failed to decode user info", http.StatusInternalServerError)
		return
	}

	// Restrict to vunetsystems.com domain
	if !strings.HasSuffix(userInfo.Email, "@vunetsystems.com") {
		http.Error(w, "Access denied: Only vunetsystems.com emails are allowed", http.StatusForbidden)
		return
	}

	// Set session
	session, _ := sessionStore.Get(r, "vuDataSim-session")
	session.Values["authenticated"] = true
	session.Values["user_email"] = userInfo.Email
	session.Values["user_name"] = userInfo.Name
	log.Printf("DEBUG: Setting session - authenticated: %v, email: %s, name: %s", session.Values["authenticated"], session.Values["user_email"], session.Values["user_name"])
	err = session.Save(r, w)
	if err != nil {
		log.Printf("DEBUG: Error saving session: %v", err)
	} else {
		log.Printf("DEBUG: Session saved successfully")
	}

	log.Printf("DEBUG: Redirecting to / after successful OAuth")
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

// HandleAuthLogout clears the session
func handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionStore.Get(r, "vuDataSim-session")
	session.Values["authenticated"] = false
	delete(session.Values, "user_email")
	delete(session.Values, "user_name")
	session.Save(r, w)
	http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
}

// HandleAuthUser returns current user info (for frontend)
func handleAuthUser(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionStore.Get(r, "vuDataSim-session")
	authenticated, _ := session.Values["authenticated"].(bool)
	email, _ := session.Values["user_email"].(string)
	name, _ := session.Values["user_name"].(string)

	log.Printf("DEBUG: handleAuthUser - authenticated: %v, email: %s, name: %s", authenticated, email, name)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"authenticated": authenticated,
		"email":         email,
		"name":          name,
	})
}
