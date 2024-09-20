package middleware

import (
	"fmt"
	"net/http"

	"avitointern/pkg/session"
)

var (
	noAuthUrls = map[string]struct{}{
		"/login": struct{}{},
	}
	noSessUrls = map[string]struct{}{
		"/": struct{}{},
	}
)

func Auth(sm *session.SessionsManager, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("auth middleware")
		if _, ok := noAuthUrls[r.URL.Path]; ok {
			next.ServeHTTP(w, r)
			return
		}
		sess, err := sm.Check(r)
		_, canbeWithouthSess := noSessUrls[r.URL.Path]
		if err != nil && !canbeWithouthSess {
			fmt.Println("no auth")
			http.Redirect(w, r, "/", 302)
			return
		}
		// fmt.Println("sess in auth ", sess, " ", sess.User.OrganizationID)
		ctx := session.ContextWithSession(r.Context(), sess)
		// fmt.Println("ctx in auth ", ctx)
		// fmt.Println("\nr.WithContext(ctx) = ", r.WithContext(ctx))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
