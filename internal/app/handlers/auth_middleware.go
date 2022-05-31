package handlers

import (
	"context"
	"errors"
	"github.com/polosaty/go-dev-final/internal/app/storage"
	"log"
	"net/http"
)

func NewSession(token string) *storage.Session {
	return &storage.Session{
		Token: token,
	}
}

type requestContextKeyType string

const requestContextKey = requestContextKeyType("Session")

func authMiddleware(repo storage.Repository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			cookie, err := r.Cookie("auth")
			if err != nil {
				if errors.Is(err, http.ErrNoCookie) {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
			}

			userID, err := repo.GetUserByToken(ctx, cookie.Value)
			if err != nil {
				log.Println("error while get user by token:", err)
				if errors.Is(err, storage.ErrWrongToken) {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			session := &storage.Session{
				Token:  cookie.Value,
				UserID: userID,
			}

			r = r.WithContext(context.WithValue(r.Context(), requestContextKey, session))

			next.ServeHTTP(w, r)
		})
	}
}

func GetSession(req *http.Request) *storage.Session {
	sessCtx := req.Context().Value(requestContextKey)
	sess, _ := sessCtx.(*storage.Session)
	return sess
}
