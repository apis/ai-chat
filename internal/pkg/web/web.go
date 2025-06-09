package web

import (
	"github.com/rs/zerolog/log"

	"bytes"
	"html/template"
	"net/http"
)

func RenderResponse(status int, templates *template.Template, templateName string, data any, headers Headers, cookie *http.Cookie) *Response {
	var buffer bytes.Buffer
	if err := templates.ExecuteTemplate(&buffer, templateName, data); err != nil {
		log.Error().Err(err).Str("template_name", templateName).Msg("templates.ExecuteTemplate() failed")
		return GetEmptyResponse(http.StatusInternalServerError, nil, nil)
	}

	return &Response{
		Status:      status,
		ContentType: "text/html",
		Content:     buffer.Bytes(),
		Headers:     headers,
		Cookie:      cookie,
	}
}

func GetEmptyResponse(status int, headers Headers, cookie *http.Cookie) *Response {
	return GetResponse(status, []byte(""), headers, cookie)
}

func GetResponse(status int, content []byte, headers Headers, cookie *http.Cookie) *Response {
	return &Response{
		Status:  status,
		Content: content,
		Headers: headers,
		Cookie:  cookie,
	}
}
