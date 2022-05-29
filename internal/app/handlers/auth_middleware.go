package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/polosaty/go-dev-final/internal/app/storage"
	"io"
	"net/http"
)

func NewSession(token string) *storage.Session {
	return &storage.Session{
		Token: token,
	}
}

type RequestContextKeyType string

const RequestContextKey = RequestContextKeyType("Session")

func authMiddleware(secretKey []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var session *storage.Session
			cookie, err := r.Cookie("auth")
			if err != nil {
				//if errors.Is(err, http.ErrNoCookie) {
				//	session = NewSession()
				//	session.signSession(secretKey)
				//	sessionJSON, err := json.Marshal(session)
				//	if err != nil {
				//		w.WriteHeader(http.StatusInternalServerError)
				//		io.WriteString(w, err.Error())
				//		return
				//	}
				//
				//	cookie = &http.Cookie{
				//		Path:  "/",
				//		Name:  "auth",
				//		Value: base64.URLEncoding.EncodeToString(sessionJSON),
				//		//Expires: time.Now().Add(48 * time.Hour),
				//	}
				//	r.AddCookie(cookie)
				//	http.SetCookie(w, cookie)
				//} else {
				//	w.WriteHeader(http.StatusInternalServerError)
				//	io.WriteString(w, err.Error())
				//	return
				//}
			} else {

				cookieJSON, err := base64.URLEncoding.DecodeString(cookie.Value)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					io.WriteString(w, err.Error())
					return
				}
				err = json.Unmarshal(cookieJSON, &session)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					io.WriteString(w, err.Error())
					return
				}
				//if !session.checkSignature(secretKey) {
				//	w.WriteHeader(http.StatusBadRequest)
				//	io.WriteString(w, "bad session signature")
				//	return
				//}
			}

			r = r.WithContext(context.WithValue(r.Context(), RequestContextKey, session))

			next.ServeHTTP(w, r)
		})
	}
}

func GetSessionFromCookie(req *http.Request) *storage.Session {
	sessCtx := req.Context().Value(RequestContextKey)
	sess, _ := sessCtx.(*storage.Session)
	return sess
}
