package api

import (
	"net/http"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/context"
	"github.com/khulnasoft-lab/distro/api/helpers"
	"github.com/khulnasoft-lab/distro/db"
	"github.com/khulnasoft-lab/distro/util"
)

func authenticationHandler(w http.ResponseWriter, r *http.Request) bool {
	var userID int

	authHeader := strings.ToLower(r.Header.Get("authorization"))

	if len(authHeader) > 0 && strings.Contains(authHeader, "bearer") {
		token, err := helpers.Store(r).GetAPIToken(strings.Replace(authHeader, "bearer ", "", 1))

		if err != nil {
			if err != db.ErrNotFound {
				log.Error(err)
			}

			w.WriteHeader(http.StatusUnauthorized)
			return false
		}

		userID = token.UserID
	} else {
		// fetch session from cookie
		cookie, err := r.Cookie("distro")
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return false
		}

		value := make(map[string]interface{})
		if err = util.Cookie.Decode("distro", cookie.Value, &value); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return false
		}

		user, ok := value["user"]
		sessionVal, okSession := value["session"]
		if !ok || !okSession {
			w.WriteHeader(http.StatusUnauthorized)
			return false
		}

		userID = user.(int)
		sessionID := sessionVal.(int)

		// fetch session
		session, err := helpers.Store(r).GetSession(userID, sessionID)

		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return false
		}

		if time.Since(session.LastActive).Hours() > 7*24 {
			// more than week old unused session
			// destroy.
			if err := helpers.Store(r).ExpireSession(userID, sessionID); err != nil {
				// it is internal error, it doesn't concern the user
				log.Error(err)
			}

			w.WriteHeader(http.StatusUnauthorized)
			return false
		}

		if err := helpers.Store(r).TouchSession(userID, sessionID); err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusUnauthorized)
			return false
		}
	}

	user, err := helpers.Store(r).GetUser(userID)
	if err != nil {
		if err != db.ErrNotFound {
			// internal error
			log.Error(err)
		}
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}

	context.Set(r, "user", &user)
	return true
}

// nolint: gocyclo
func authentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok := authenticationHandler(w, r)
		if ok {
			next.ServeHTTP(w, r)
		}
	})
}

// nolint: gocyclo
func authenticationWithStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		store := helpers.Store(r)

		var ok bool

		db.StoreSession(store, r.URL.String(), func() {
			ok = authenticationHandler(w, r)
		})

		if ok {
			next.ServeHTTP(w, r)
		}
	})
}
