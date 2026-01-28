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
	"maps"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/opsmx/ai-guardian-api/pkg/database"
	"github.com/opsmx/ai-guardian-api/pkg/models"

	// Redis backend Session Store
	"github.com/rbcervilla/redisstore/v9"

	// Posgres backend Session Store
	"github.com/antonlindstrom/pgstore"
)

// Thread-safe map for logged in users
var (
	loggedInUsers = make(map[string]bool)
	usersMutex    sync.RWMutex
)

func saveUser(username string) {
	usersMutex.Lock()
	defer usersMutex.Unlock()
	loggedInUsers[username] = true
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

	usersMutex.Lock()
	loggedInUsers = make(map[string]bool)
	usersMutex.Unlock()

	saMu.Lock()
	sessionAccess = make(map[string]models.AuthUser)
	saMu.Unlock()
	startSessionTracking(float64(sessionTimeout))
}

// Initialize the Session Store to DB for Redis backend sessions
func InitRedisSessionsStore(sessionTimeout int, hostAndPort, username, password string) error {
	// Use the centralized Redis client from database package
	client := database.GetRedis()

	// Initialize session store
	store, err := redisstore.NewRedisStore(context.Background(), client)
	if err != nil {
		log.Printf("failed to create redis session store:%v", err)
		return err
	}

	store.KeyPrefix("ai_guardian_")
	store.Options(sessions.Options{
		Path:     "/", // Changed from "/ai-gyardian" to "/"
		Domain:   "",  // Changed from "opsmx.com" to "" (empty means current domain)
		MaxAge:   sessionTimeout,
		Secure:   false,
		HttpOnly: true,
	})

	gob.Register(models.AuthUser{})
	sessionStore = store

	usersMutex.Lock()
	loggedInUsers = make(map[string]bool) // Initialize the map
	usersMutex.Unlock()

	saMu.Lock()
	sessionAccess = make(map[string]models.AuthUser)
	saMu.Unlock()
	startSessionTracking(float64(sessionTimeout))

	return nil
}

// Initialize the Session Store to DB for Postres backend sessions
func InitPgSessionsStore(sessionTimeout int, dbuser, pass, hostAndPort, dbname, sslmode string) error {
	store, err := pgstore.NewPGStore("postgres://"+dbuser+":"+pass+"@"+hostAndPort+
		"/"+dbname+"?sslmode="+sslmode, []byte(generateRand()))
	if err != nil {
		log.Printf("error in postgress session store:%v", err)
		return err
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

	usersMutex.Lock()
	loggedInUsers = make(map[string]bool) // Initialize the map
	usersMutex.Unlock()

	saMu.Lock()
	sessionAccess = make(map[string]models.AuthUser)
	saMu.Unlock()
	startSessionTracking(float64(sessionTimeout))

	return nil
}

// On login success, create a Session ID, set a cookie and register the user
// in the session storage. User name is added in
func CreateSession(w http.ResponseWriter, r *http.Request, refreshToken, username string) {
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
	cookieSplitted := strings.Split(w.Header().Get("Set-Cookie"), ";")
	if len(cookieSplitted) > 0 {
		extSessionId := strings.ReplaceAll(cookieSplitted[0], "SESSION=", "")
		saveUserSession(extSessionId, *user)
	}
	log.Printf("Session created successfully for user")
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

	session, err := sessionStore.Get(r, "SESSION")
	if err != nil {
		log.Printf("Error getting session in GetSessionExists: %v", err)
		return nil
	}

	user := session.Values["user"]
	if user == nil {
		return nil
	}

	if userTmp, ok := user.(models.AuthUser); ok {
		authUser := &models.AuthUser{
			Username:      userTmp.Username,
			Authenticated: true,
		}

		cookie, err := r.Cookie("SESSION")
		if err == nil {
			extSessionId := cookie.Value
			saveUserSession(extSessionId, *authUser)
		}
		return authUser
	} else {
		log.Printf("Session Value was not a User")
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
		usersMutex.Lock()
		delete(loggedInUsers, user.Username)
		usersMutex.Unlock()
	}

	session.Values["user"] = models.AuthUser{}
	session.Options.MaxAge = -1

	cookie, err := r.Cookie("SESSION")
	if err == nil {
		extSessionId := cookie.Value
		saveUserSession(extSessionId, *user)
	}

	err = session.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("User Status AFTER logout")
}

// method to return the current session store
func GetCurrentSessionStore() sessions.Store {
	return sessionStore
}

// Check if a user is currently logged in
func IsUserLoggedIn(username string) bool {
	usersMutex.RLock()
	defer usersMutex.RUnlock()
	return loggedInUsers[username]
}

// Get all currently logged in users
func GetLoggedInUsers() []string {
	usersMutex.RLock()
	defer usersMutex.RUnlock()
	users := make([]string, 0, len(loggedInUsers))
	for username := range loggedInUsers {
		users = append(users, username)
	}
	return users
}

// Get count of logged in users
func GetLoggedInUserCount() int {
	usersMutex.RLock()
	defer usersMutex.RUnlock()
	return len(loggedInUsers)
}

func generateRand() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		log.Printf("Unexpected error in rand.Read: %v", err)
	}

	state := base64.URLEncoding.EncodeToString(b)
	return state
}

// Below code is for session tracking

// Thread-safe map for user sessions
var (
	sessionAccess = make(map[string]models.AuthUser)
	saMu          sync.Mutex
)

var dbStore database.Postgres

func saveUserSession(sessionId string, user models.AuthUser) {
	saMu.Lock()
	defer saMu.Unlock()
	now := time.Now()
	if s, sOk := sessionAccess[sessionId]; sOk {
		s.Authenticated = user.Authenticated
		s.LastAccessed = now
		sessionAccess[sessionId] = s
	} else {
		user.LastAccessed = now
		sessionAccess[sessionId] = user
	}
}

func deleteSessionAccess(sessionId string) {
	saMu.Lock()
	defer saMu.Unlock()
	delete(sessionAccess, sessionId)
}

func getAllSessionAccess() map[string]models.AuthUser {
	saMu.Lock()
	defer saMu.Unlock()
	clone := maps.Clone(sessionAccess)
	return clone
}

func startSessionTracking(sessionMaxDuration float64) {
	dbStore = database.GetPostgres()
	go func() {
		for {
			sessions := getAllSessionAccess()
			now := time.Now()
			var (
				ids               []string
				lastAccessedTimes []time.Time
			)
			for sessionId, user := range sessions {
				ids = append(ids, sessionId)
				lastAccessedTimes = append(lastAccessedTimes, user.LastAccessed)
				// keep duration matched with session duration
				if !user.Authenticated || now.Sub(user.LastAccessed).Seconds() > sessionMaxDuration {
					deleteSessionAccess(sessionId)
				}
			}
			// ops here for update
			if len(ids) != 0 && len(ids) == len(lastAccessedTimes) {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				if err := batchUpdateSessionLastAccessed(ctx, ids, lastAccessedTimes); err != nil {
					log.Println("failed to update last accessed:", err)
				}
				cancel()
			}
			// each 10 seconds we will update active sessions in db
			time.Sleep(time.Duration(time.Second * 10))
		}
	}()
}

func batchUpdateSessionLastAccessed(
	ctx context.Context,
	sessionIds []string,
	lastAccessed []time.Time,
) error {
	query := `
		INSERT INTO user_sessions (id, last_accessed)
		SELECT * FROM UNNEST($1::text[], $2::timestamptz[])
		ON CONFLICT (id)
		DO UPDATE SET last_accessed = EXCLUDED.last_accessed
		WHERE user_sessions.last_accessed IS DISTINCT FROM EXCLUDED.last_accessed;
	`
	_, err := dbStore.Exec(ctx, query, sessionIds, lastAccessed)
	return err
}
