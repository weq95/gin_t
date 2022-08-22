package core

import (
	"context"
	"errors"
	"gin-core/core/bind"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	defaultMultipartMemory = 32 << 20
)

type Context struct {
	matched        bool //url是否匹配
	escape         bool //是否返回上下文
	index          uint8
	abortIndex     uint8
	status         uint //状态码
	written        bool
	queryCache     url.Values //地址栏参数
	formCache      url.Values //body参数
	items          map[string]any
	lock           sync.RWMutex
	group          []*handleFuncNode
	Request        *http.Request
	ResponseWriter http.ResponseWriter
	Engine         *Engine
	Params         Params
	fullPath       string
}

func (c *Context) reset() {
	c.index = 0
	c.matched = false
	c.queryCache = nil
	c.formCache = nil
	c.status = 0
	c.written = false
	c.abortIndex = 0
}

func (c *Context) start() {
	defer c.finish()
	c.Next()
}

// 不认为这是一个好的设计 还写尼玛啊
func (c *Context) finish() {
	if c.status != 0 && !c.written {
		c.ResponseWriter.WriteHeader(int(c.status))
	}
}

func (c *Context) IsMatched() bool {
	return c.matched
}

func (c *Context) Next() {
	c.index++

	for c.index <= uint8(len(c.group)) && !c.IsAborted() {
		c.group[c.index-1].HandleFunc(c)

		c.index++
	}
}

func (c *Context) IsAborted() bool {
	return c.abortIndex != 0
}

func (c *Context) Flusher() http.Flusher {
	//为啥这里时类型断言吖
	return c.ResponseWriter.(http.Flusher)
}

func (c *Context) SaveUploadFile(name string) (string, error) {
	var fs = c.BluePrint().FileStorage()

	return c.SaveUploadFileWith(fs, name)
}

func (c *Context) BluePrint() *BluePrint {
	return c.group[c.index-1].BluePrint
}

func (c *Context) Logger() Logger {
	return c.BluePrint().Logger()
}

func (c *Context) SaveUploadFileWith(fs FileStorage, name string) (string, error) {
	if fs == nil {
		return "", errors.New("`FileStorage` can be nil type")
	}

	var file, fileHeader, err = c.Request.FormFile(name)
	if err != nil {
		return "", err
	}

	if err = file.Close(); err != nil {
		return "", err
	}

	return fs.Save(fileHeader)
}

// Data 解析数据, 这里是重点代码块
func (c *Context) Data(v any) error {
	if err := c.BluePrint().Parsers().Parse(c, v); err != nil {
		return err
	}

	return c.BluePrint().Validator().Validate(v)
}

func (c *Context) Query() url.Values {
	if c.queryCache == nil {
		c.queryCache = c.Request.URL.Query()
	}

	return c.queryCache
}

// QueryValue 通过 net.value 查询参数
func (c *Context) QueryValue(key string) Value {
	var value = c.Query().Get(key)

	return Value(value)
}

func (c *Context) QueryValues(key string) Values {
	return NewValues(c.Query()[key])
}

func (c *Context) From() url.Values {
	if c.formCache == nil {
		_ = c.Request.ParseForm()

		c.formCache = c.Request.PostForm
	}

	return c.formCache
}

// FormValue 获取底层url上的参数
func (c *Context) FormValue(key string) Value {
	return Value(c.From().Get(key))
}

func (c *Context) FormValues(key string) Values {
	return NewValues(c.From()[key])
}

// FullPath 返回当前请求的完整路径
func (c *Context) FullPath() string {
	return c.fullPath
}

func (c *Context) ContentType() string {
	return c.Request.Header.Get("Content-Type")
}

// Bind 实现绑定业务
func (c *Context) Bind(binder bind.Binder, v any) error {
	return binder.Bind(c.Request, v)
}

// BindQuery GET 查询参数bind
func (c *Context) BindQuery(v any) error {
	return c.Bind(bind.QueryBinder{}, v)
}

func (c *Context) BindForm(v any) error {
	if err := c.Request.ParseForm(); err != nil {
		return err
	}

	return c.Bind(bind.FormBinder{}, v)
}

func (c *Context) BindMultipartForm(v any) error {
	if err := c.Request.ParseMultipartForm(c.Engine.MultipartMemory); err != nil {
		return err
	}

	return c.Bind(bind.MultipartFormBodyBinder{}, v)
}

func (c *Context) BindJSON(v any) error {
	//获取json 格式化插件
	var serialize = c.BluePrint().JSONSerializer()

	return c.Bind(bind.JsonBodyBinder{Serializer: serialize}, v)
}

func (c *Context) BindXML(v any) error {
	var serializer = c.BluePrint().XMLSerializer()

	return c.Bind(bind.XmlBodyBinder{Serializer: serializer}, v)
}

func (c *Context) BindHeader(v any) error {
	return c.Bind(bind.HeaderBinder{}, v)
}

func (c *Context) BindURI(v any) error {
	var value = c.Params.ToURLValues()

	return c.Bind(bind.URIBinder{Values: value}, v)
}

// GetValue 上下文附加值
func (c *Context) GetValue(key string) (any, bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	var value, ok = c.items[key]

	return value, ok
}

// SetValue 获取上下文附加值
func (c *Context) SetValue(key string, val any) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.items == nil {
		c.items = make(map[string]any, 0)
	}

	c.items[key] = val
}

func (c *Context) SetStatus(code uint) {
	c.status = code
}

// SetHeader 设置header头信息
func (c *Context) SetHeader(key, val string) {
	c.ResponseWriter.Header().Set(key, val)
}

func (c *Context) SetCookie(cookie *http.Cookie) {
	http.SetCookie(c.ResponseWriter, cookie)
}

func (c *Context) Render(render Render, v any) error {
	if c.status != 0 {
		c.ResponseWriter.WriteHeader(int(c.status))
		c.status = 0
	}

	if !bodyAllowedForStatus(int(c.status)) {
		return nil
	}

	c.written = true
	return render.Render(c.ResponseWriter, v)
}

func (c *Context) JSON(data any) error {
	var serializer = c.BluePrint().JSONSerializer()

	return c.Render(JsonRender{Serializer: serializer}, data)
}

func (c *Context) XML(v any) error {
	var serializer = c.BluePrint().XMLSerializer()

	return c.Render(XmlRender{Serializer: serializer}, v)
}

func (c *Context) String(format string, v ...any) error {
	return c.Render(StringRender{
		Format: format,
		Data:   v,
	}, nil)
}

func (c *Context) Redirect(code int, url string) error {
	return c.Render(RedirectRender{
		RedirectURL: url,
		Code:        code,
		Request:     c.Request,
	}, nil)
}

func (c *Context) ServeFile(filepath, filename string) error {
	return c.Render(FileAttachmentRender{
		Request:  c.Request,
		FileName: filename,
		FilePath: filepath,
	}, nil)
}

func (c *Context) ServeContent(name string, modTime time.Time, content io.ReadSeeker) error {
	return c.Render(ContentRender{
		Name:    name,
		ModTime: modTime,
		Request: c.Request,
		Content: content,
	}, nil)
}

func (c *Context) Write(data []byte) error {
	var _, err = c.ResponseWriter.Write(data)

	return err
}

// Escape 可以让上下文不返回池中
func (c *Context) Escape() {
	c.escape = true
}

// IsEscape 返回转移状态
func (c *Context) IsEscape() bool {
	return c.escape
}

func (c *Context) Abort() {
	c.abortIndex = c.index
}

func (c *Context) AbortHandler() HandleFunc {
	if !c.IsAborted() {
		return nil
	}

	//获取当前回调函数
	return c.group[c.abortIndex].HandleFunc
}

func (c *Context) AbortWithJSON(data any) {
	_ = c.JSON(data)
	c.Abort()
}

func (c *Context) AbortWithXML(data any) {
	_ = c.XML(data)
	c.Abort()
}

func (c *Context) AbortWithString(text string, data ...any) {
	_ = c.String(text, data...)

	c.Abort()
}

func (c *Context) AbortWithStatus(code uint) {
	c.SetStatus(code)
	c.Abort()
}

// IsWebsocket 判断是不是websocket连接
func (c *Context) IsWebsocket() bool {
	return strings.Contains(strings.ToLower(c.Request.Header.Get("Connection")), "upgrade") && strings.EqualFold(c.Request.Header.Get("Upgrade"), "websocket")
}

// IsAjax 检查当前是否是 ajax 请求
func (c *Context) IsAjax() bool {
	return strings.EqualFold(c.Request.Header.Get("X-Requested-With"), "XMLHttpRequest")
}

type contextKey struct {
}
type contextExits struct {
}

var (
	ContextKey   = contextKey{}
	contextExist = contextExits{}
)

// SetContextIntoRequest 将上下文设置为请求上下文
func SetContextIntoRequest(ctx *Context) {
	var c = ctx.Request.Context()
	if c.Value(contextExist) != nil {
		return
	}

	//其实这里的request 已经被重置了
	c = context.WithValue(c, contextExist, contextExist)
	c = context.WithValue(c, ContextKey, ctx)

	//把请求的context 重新设置
	ctx.Request = ctx.Request.WithContext(c)
}

func (c *Context) RemoteIP() string {
	var ip, _, err = net.SplitHostPort(strings.TrimSpace(c.Request.RemoteAddr))
	if err != nil {
		return ""
	}

	return ip
}
