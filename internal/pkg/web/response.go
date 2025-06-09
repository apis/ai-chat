package web

import (
	"net/http"
)

type Response struct {
	Status      int
	ContentType string
	Content     []byte
	Headers     Headers
	Cookie      *http.Cookie
}

func (response *Response) Write(responseWriter http.ResponseWriter) {
	if response != nil {
		if response.Cookie != nil {
			http.SetCookie(responseWriter, response.Cookie)
		}
		if response.ContentType != "" {
			responseWriter.Header().Set("Content-Type", response.ContentType)
		}
		for k, v := range response.Headers {
			responseWriter.Header().Set(k, v)
		}
		responseWriter.WriteHeader(response.Status)
		_, err := responseWriter.Write(response.Content)

		if err != nil {
			responseWriter.WriteHeader(http.StatusInternalServerError)
		}
	} else {
		responseWriter.WriteHeader(http.StatusOK)
	}
}
