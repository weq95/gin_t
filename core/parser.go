package core

import "strings"

const (
	minePostForm          = "application/x-www-form-urlencoded"
	mimeJson              = "application/json"
	mimeMultipartPostForm = "multipart/form-data"
	mimeText              = "text/plain"
	mimeXml               = "application/xml"
	mimeXml2              = "text/xml"
	mimeHtml              = "text/html"
)

type Parser interface {
	Parse(ctx *Context, v any) error
	Match(ctx *Context) bool
}

type Parsers []Parser

func (p Parsers) Parse(ctx *Context, v any) error {
	for _, parser := range p {
		if parser.Match(ctx) {
			return parser.Parse(ctx, v)
		}
	}

	return QueryParser{}.Parse(ctx, v)
}

type FormParser struct {
}

func (f FormParser) Parse(ctx *Context, v any) error {
	return ctx.BindForm(v)
}

func (f FormParser) Match(ctx *Context) bool {
	return strings.Contains(strings.ToLower(ctx.ContentType()), minePostForm)
}

type JsonParser struct {
}

func (j JsonParser) Parse(ctx *Context, v any) error {
	return ctx.BindJSON(v)
}

func (j JsonParser) Match(ctx *Context) bool {
	return strings.Contains(strings.ToLower(ctx.ContentType()), mimeJson)
}

type MultipartFormParser struct {
}

func (m MultipartFormParser) Parse(ctx *Context, v any) error {
	return ctx.BindMultipartForm(v)
}

func (m MultipartFormParser) Match(ctx *Context) bool {
	return strings.Contains(strings.ToLower(ctx.ContentType()), mimeMultipartPostForm)
}

type XMLParser struct {
}

func (x XMLParser) Parse(ctx *Context, v any) error {
	return ctx.BindXML(v)
}

func (x XMLParser) Match(ctx *Context) bool {
	var cType = strings.ToLower(ctx.ContentType())

	return strings.Contains(cType, mimeXml) || strings.Contains(cType, mimeXml2)
}

type QueryParser struct {
}

func (q QueryParser) Parse(ctx *Context, v any) error {
	return ctx.BindQuery(v)
}

func (q QueryParser) Match(ctx *Context) bool {
	return true
}
