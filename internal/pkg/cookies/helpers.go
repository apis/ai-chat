package cookies

import (
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"net/http"
)

const CookieName = "chat-session-id"

var SecretKey = []byte(`2pC5z.3;Kk3wsr20,Ool{h;C%:Gq4eN=q\6F"Dfa£GMB[.j0`)

func GetIdFromCookie(request *http.Request) uuid.UUID {
	cookie, err := request.Cookie(CookieName)
	if err != nil {
		log.Error().Err(err).Msg("session id cookie can't be retrieved")
		return uuid.Nil
	}

	sessionId, err := VerifySignedKeyValue(cookie.Name, cookie.Value, SecretKey)
	if err != nil {
		log.Error().Err(err).Msg("session id cookie value can't be verified")
		return uuid.Nil
	}

	id, err := uuid.Parse(sessionId)
	if err != nil {
		log.Error().Err(err).Msg("session id cookie contains invalid id")
		return uuid.Nil
	}
	return id
}

func SetIdToCookie(id uuid.UUID) *http.Cookie {
	value, err := SignKeyValue(CookieName, id.String(), SecretKey)
	if err != nil {
		log.Error().Err(err).Msg("cookies.SignKeyValue() failed")
		return nil
	}

	cookie := http.Cookie{
		Name:     CookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

	return &cookie
}
