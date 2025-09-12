// Implemented using gorilla/sessions package.
// Supporting in-mem, redis and postgres
// For most users, in-mem is adequate, pod-scaling is not possible with in-mem
package session

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"log"
	"net/http"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/opsmx/ai-gyardian-api/pkg/models"

	// Redis backend Session Store
	"github.com/rbcervilla/redisstore/v9"
	"github.com/redis/go-redis/v9"

	// Posgres backend Session Store
	"github.com/antonlindstrom/pgstore"
)

// Temporary HACK: We need to store this in Redis
// var loggedInUsers []User  // We store the logged in users here.
var loggedInUsers map[string]bool // We store the logged in users here.
func saveUser(username string) {
	loggedInUsers[username] = true // Save the user as logged in
}

// sessionStore is the actual gorilla/sessions store
var sessionStore sessions.Store

// In memory, securecookie storage
// Lifted from : https://github.com/CurtisVermeeren/gorilla-sessions-tutorial
func InitMemSessionsStore(sessionTimeout int) {
	authKeyOne := securecookie.GenerateRandomKey(64)
	encryptionKeyOne := securecookie.GenerateRandomKey(32)

	store := sessions.NewCookieStore(
		authKeyOne,
		encryptionKeyOne,
	)

	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   sessionTimeout,
		HttpOnly: true,
	}

	gob.Register(models.AuthUser{})
	sessionStore = store
	loggedInUsers = make(map[string]bool)
}

// Initialize the Session Store to DB for Redis backend sessions
func InitRedisSessionsStore(sessionTimeout int, hostAndPort, username, password string) error {
	client := redis.NewUniversalClient(&redis.UniversalOptions{
		Username: username,
		Password: password,
		Addrs:    []string{hostAndPort},
	})
	// Initialize session store
	store, err := redisstore.NewRedisStore(context.Background(), client)
	if err != nil {
		log.Printf("failed to create redis session store:%v", err)
		return err
	}

	store.KeyPrefix("ai_gyardian_")
	store.Options(sessions.Options{
		Path:     "/", // Changed from "/ai-gyardian" to "/"
		Domain:   "",  // Changed from "opsmx.com" to "" (empty means current domain)
		MaxAge:   sessionTimeout,
		Secure:   false,
		HttpOnly: true,
	})

	gob.Register(models.AuthUser{})
	sessionStore = store
	loggedInUsers = make(map[string]bool) // Initialize the map
	return nil
}

// Initialize the Session Store to DB for Postres backend sessions
func InitPgSessionsStore(sessionTimeout int, dbuser, pass, hostAndPort, dbname, sslmode string) error {
	log.Printf("Connect String:%s\n", "postgres://"+dbuser+":"+"REDACTED@"+hostAndPort+
		"/"+dbname+"?sslmode="+sslmode)
	store, err := pgstore.NewPGStore("postgres://"+dbuser+":"+pass+"@"+hostAndPort+
		"/"+dbname+"?sslmode="+sslmode, []byte(generateRand()))
	if err != nil {
		log.Printf("error in postgress session store:%v", err)
	}

	store.Options = &sessions.Options{
		Path:     "/",
		Domain:   "",
		MaxAge:   sessionTimeout,
		Secure:   false,
		HttpOnly: true,
		SameSite: 0,
	}
	gob.Register(models.AuthUser{})
	sessionStore = store
	loggedInUsers = make(map[string]bool) // Initialize the map
	return nil
}

// On login success, create a Session ID, set a cookie and register the user
// in the session storage. User name is added in
func CreateSession(w http.ResponseWriter, r *http.Request, refreshToken, username string) {
	log.Printf("Creating session for user: %s", username)

	if sessionStore == nil {
		log.Printf("Session store is nil!")
		http.Error(w, "Session store not initialized", http.StatusInternalServerError)
		return
	}

	session, err := sessionStore.Get(r, "SESSION")
	if err != nil {
		log.Printf("Error getting session: %v", err)
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	user := &models.AuthUser{
		Username:      username,
		Authenticated: true,
	}

	session.Values["user"] = user
	err = session.Save(r, w)
	if err != nil {
		log.Printf("session.Save(r, SESSION):%v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	saveUser(user.Username) // Save user as logged in
	log.Printf("Session created successfully for user: %s", user.Username)
}

// Method to check if an active session exists. Check if cookie exists
// and the user exists in session storage (user might have hit logout, session is still valid)
func GetSession(r *http.Request) (string, error) {
	user := GetSessionExists(r)
	if user == nil {
		return "", errors.New("user not logged in")
	}
	// log.Printf("User was already logged in:%s", user.Username)
	return user.Username, nil
}

// Return userName if cookie is valid
func GetSessionExists(r *http.Request) *models.AuthUser {
	if sessionStore == nil {
		log.Printf("Session store is nil in GetSessionExists!")
		return nil
	}

	log.Printf("Getting session for request to: %s", r.URL.Path)

	session, err := sessionStore.Get(r, "SESSION")
	if err != nil {
		log.Printf("Error getting session in GetSessionExists: %v", err)
		return nil
	}

	log.Printf("Session retrieved, checking values...")
	user := session.Values["user"]
	if user == nil {
		log.Printf("No user found in session values")
		return nil
	}

	log.Printf("User found in session: %+v", user)
	if userTmp, ok := user.(models.AuthUser); ok {
		log.Printf("User type assertion successful: %s", userTmp.Username)
		return &models.AuthUser{
			Username:      userTmp.Username,
			Authenticated: true,
		}
	} else {
		log.Printf("Session Value was not a User:%T:%+v\n", user, user)
	}
	return nil
}

// logout, we need a timer and delete these periodically
func DeleteSession(w http.ResponseWriter, r *http.Request) {
	if sessionStore == nil {
		log.Printf("Session store is nil in DeleteSession!")
		http.Error(w, "Session store not initialized", http.StatusInternalServerError)
		return
	}

	session, err := sessionStore.Get(r, "SESSION")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get username before clearing session
	user := GetSessionExists(r)
	if user != nil {
		delete(loggedInUsers, user.Username)
	}

	session.Values["user"] = models.AuthUser{}
	session.Options.MaxAge = -1

	err = session.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("User Status AFTER logout:%+v", session.Values["user"])
}

// method to return the current session store
func GetCurrentSessionStore() sessions.Store {
	return sessionStore
}

// Check if a user is currently logged in
func IsUserLoggedIn(username string) bool {
	return loggedInUsers[username]
}

// Get all currently logged in users
func GetLoggedInUsers() []string {
	users := make([]string, 0, len(loggedInUsers))
	for username := range loggedInUsers {
		users = append(users, username)
	}
	return users
}

// Get count of logged in users
func GetLoggedInUserCount() int {
	return len(loggedInUsers)
}

func generateRand() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		log.Printf("Unexpected error in rand.Read: %v", err)
	}

	state := base64.URLEncoding.EncodeToString(b)
	log.Printf("state=%s\n", state)
	return state
}
