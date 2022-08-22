package bind

import (
	"gin-core/core/color"
	"net/http"
	"net/url"
)

type Binder interface {
	Bind(req *http.Request, v any) error
}

type QueryBinder struct {
}

func (q QueryBinder) Bind(req *http.Request, v any) error {
	var binder = URLValueBinder{
		TagName:     formTag,
		BindTagName: bindTag,
	}

	return binder.BindForm(req.URL.Query(), v)
}

type FormBinder struct {
}

func (f FormBinder) Bind(r *http.Request, v any) error {
	var binder = URLValueBinder{TagName: formTag, BindTagName: bindTag}

	return binder.BindForm(r.Form, v)
}

type MultipartFormBodyBinder struct {
}

func (m MultipartFormBodyBinder) Bind(r *http.Request, v any) error {
	var binder = HttpMultipartFormBinder{
		URLValueBinder: URLValueBinder{TagName: formTag, BindTagName: bindTag},
		FieldTag:       fileTag,
	}

	return binder.BindMultipartForm(r.MultipartForm, v)
}

type JsonBodyBinder struct {
	Serializer color.Serializer
}

func (j JsonBodyBinder) Bind(r *http.Request, v any) error {
	return j.Serializer.Decode(r.Body, v)
}

type XmlBodyBinder struct {
	Serializer color.Serializer
}

func (x XmlBodyBinder) Bind(r *http.Request, v any) error {
	return x.Serializer.Decode(r.Body, v)
}

type HeaderBinder struct {
}

func (h HeaderBinder) Bind(r *http.Request, v any) error {
	var binder = URLValueBinder{
		TagName:     headerTag,
		BindTagName: bindTag,
	}

	return binder.BindForm(url.Values(r.Header), v)
}

type URIParamContextKey struct {
}

type URIBinder struct {
	Values url.Values
}

func (u URIBinder) Bind(r *http.Request, v any) error {
	var binder = URLValueBinder{TagName: uriTag, BindTagName: bindTag}

	return binder.BindForm(u.Values, v)
}
