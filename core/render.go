package core

import (
	"fmt"
	"gin-core/core/color"
	"io"
	"net/http"
	"time"
)

// Render 公共写入响应流的方法
type Render interface {
	Render(w http.ResponseWriter, v any) error
}

func writeContentType(w http.ResponseWriter, contentType string) {
	w.Header().Del("Content-Type")
	w.Header().Set("Content-Type", contentType)
}

// #---------------------------------------------------
type JsonRender struct {
	Serializer color.Serializer
}

func (j JsonRender) Render(w http.ResponseWriter, v any) error {
	writeContentType(w, "application/json;charset=utf-8")

	return j.Serializer.Encode(w, v)
}

// #---------------------------------------------------
type FileAttachmentRender struct {
	FileName string
	FilePath string
	Request  *http.Request
}

func (f FileAttachmentRender) Render(w http.ResponseWriter, v any) error {
	w.Header().Set("Content-Disposition", "attachment; filename=\""+f.FileName+"\"")

	http.ServeFile(w, f.Request, f.FilePath)
	return nil
}

// #---------------------------------------------------
type RedirectRender struct {
	RedirectURL string
	Code        int
	Request     *http.Request
}

func (r RedirectRender) Render(w http.ResponseWriter, v any) error {
	if r.Code == 0 {
		r.Code = http.StatusFound
	}

	http.Redirect(w, r.Request, r.RedirectURL, r.Code)
	return nil
}

// #---------------------------------------------------
type StringRender struct {
	Format string
	Data   []any
}

func (s StringRender) Render(w http.ResponseWriter, v any) error {
	writeContentType(w, "text/html;charset=utf-8")

	if len(s.Data) > 0 {
		var _, err = fmt.Fprintf(w, s.Format, s.Data...)
		return err
	}

	var _, err = w.Write(color.StringToByte(s.Format))

	return err
}

// #---------------------------------------------------
type XmlRender struct {
	Serializer color.Serializer
}

func (x XmlRender) Render(w http.ResponseWriter, v any) error {
	writeContentType(w, "text/xml;charset=utf-8")

	return x.Serializer.Encode(w, v)
}

// #---------------------------------------------------
type ContentRender struct {
	Name    string
	ModTime time.Time
	Request *http.Request
	Content io.ReadSeeker
}

func (c ContentRender) Render(w http.ResponseWriter, v any) error {
	http.ServeContent(w, c.Request, c.Name, c.ModTime, c.Content)

	return nil
}
