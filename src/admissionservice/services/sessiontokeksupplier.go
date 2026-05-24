package services

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
)

const WaitingRoomSessionTokenCookie = "waiting_room_session"

func HandleSessionToken(w http.ResponseWriter, r *http.Request) (string, error) {
	var sessionToken string
	cookie, err := r.Cookie(WaitingRoomSessionTokenCookie)
	if err == nil {
		if cookie.Value == "" {
			sessionToken = uuid.New().String()
			setSessionToken(w, sessionToken)
			return sessionToken, nil
		}
		sessionToken = cookie.Value
		setSessionToken(w, sessionToken)
		return sessionToken, nil
	}
	if errors.Is(err, http.ErrNoCookie) {
		sessionToken = uuid.New().String()
		setSessionToken(w, sessionToken)
		return sessionToken, nil
	}
	return "", err
}

func setSessionToken(w http.ResponseWriter, sessionToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     WaitingRoomSessionTokenCookie,
		Value:    sessionToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
