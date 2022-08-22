package core

import (
	"gin-core/core/color"
	"gin-core/core/validators"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

type handleNode struct {
	path      string
	handles   []HandleFunc //多个回调函数
	blueprint *BluePrint   //相当于工具箱
}

// BluePrint 相当于工具箱, 各种组件绑定在他身上
type BluePrint struct {
	Name           string
	Data           any         //当前BluePrint数据
	fileStorage    FileStorage //文件存储器
	parsers        Parsers
	validator      validators.Validator
	logger         Logger
	xmlSerializer  color.Serializer
	jsonSerializer color.Serializer
	parent         *BluePrint
	methodsTree    map[string][]*handleNode //各类请求方式对应的, 请求回调处理函数
	middleware     []HandleFunc
	prefix         string
}

// Use 添加中间件
func (b *BluePrint) Use(middleware ...HandleFunc) {
	b.middleware = append(b.middleware, middleware...)
}

// SetPrefix 设置前缀
func (b *BluePrint) SetPrefix(path string) {
	b.prefix = path
}

// GET 注册GET业务处理器
func (b *BluePrint) GET(path string, middleware ...HandleFunc) {
	b.Handle(http.MethodGet, path, middleware...)
}

// POST 注册POST业务处理器
func (b *BluePrint) POST(path string, middleware ...HandleFunc) {
	b.Handle(http.MethodPost, path, middleware...)
}

// PUT 注册PUT业务处理器
func (b *BluePrint) PUT(path string, middleware ...HandleFunc) {
	b.Handle(http.MethodPut, path, middleware...)
}

// PATCH is a shortcut for Handle("PATCH", path, middleware...)
func (b *BluePrint) PATCH(path string, middleware ...HandleFunc) {
	b.Handle(http.MethodPatch, path, middleware...)
}

// DELETE is a shortcut for Handle("DELETE", path, middleware...)
func (b *BluePrint) DELETE(path string, middleware ...HandleFunc) {
	b.Handle(http.MethodDelete, path, middleware...)
}

// HEAD is a shortcut for Handle("HEAD", path, middleware...)
func (b *BluePrint) HEAD(path string, middleware ...HandleFunc) {
	b.Handle(http.MethodHead, path, middleware...)
}

// OPTIONS is a shortcut for Handle("OPTIONS", path, middleware...)
func (b *BluePrint) OPTIONS(path string, middleware ...HandleFunc) {
	b.Handle(http.MethodOptions, path, middleware...)
}

// ANY 注册所有请求方式
func (b *BluePrint) ANY(path string, middleware ...HandleFunc) {
	for _, method := range httpMethods {
		b.Handle(method, path, middleware...)
	}
}

// RAW 这里会把 func(ResponseWriter, *Request) 转换成 func(*Context)
func (b *BluePrint) RAW(method, path string, handlers ...http.HandlerFunc) {
	b.Handle(method, path, RawHandlerFuncGroup(handlers...)...)
}

// Handle 注册回调函数, 放松请求时会来查找并处理
func (b *BluePrint) Handle(method, path string, middleware ...HandleFunc) {
	middleware = append(b.middleware, middleware...)

	b.register(method, &handleNode{
		path:      b.prefix + path,
		handles:   middleware, // func(*Context)
		blueprint: b,
	})
}

// register 注册业务
func (b *BluePrint) register(method string, node *handleNode) {
	if b.methodsTree == nil {
		b.methodsTree = make(map[string][]*handleNode, 0)
	}
	var methods = []string{method}
	if method == ALLMethod { //支持所有的请求方式
		methods = httpMethods[:]
	}

	for _, mt := range methods {
		var m = strings.ToUpper(mt)

		//一个请求方法对应的多个业务逻辑
		b.methodsTree[m] = append(b.methodsTree[m], node)
	}
}

// Include 把子集的处理器添加到 总的管理器中
func (b *BluePrint) Include(prefix string, branch *BluePrint) {
	branch.parent = b //设置父级
	//遍历所有方法对应的 BluePrint
	for method, nodes := range branch.methodsTree {
		for _, node := range nodes {
			b.register(method, &handleNode{
				path:      prefix + node.path, //这里已经有多个前缀了
				handles:   node.handles,
				blueprint: branch,
			})
		}
	}
}

// detectIllegalMethod 检测请求METHOD 是否合法
func detectIllegalMethod(mapping map[string]string) map[string]string {
	var cleaned = make(map[string]string, 0)
	for handleName, reqMethod := range mapping {
		var method = strings.ToUpper(reqMethod)
		for i, httpMethod := range httpMethods {
			if method == httpMethod {
				break
			}

			//没有匹配到 请求方式, 直接panic
			if i == len(httpMethods)-1 && method != httpMethod {
				panic("invalid method" + reqMethod)
			}
		}

		cleaned[handleName] = reqMethod
	}

	return cleaned
}

// Bind 这里只能绑定struct 结构体
func (b *BluePrint) Bind(path string, v any, mappings ...map[string]string) {
	var value = reflect.Indirect(reflect.ValueOf(v))
	if value.Kind() != reflect.Struct {
		panic("`v` should be a struct")
	}

	for _, mapping := range mappings {
		for handleName, methodName := range detectIllegalMethod(mapping) {
			if method := value.MethodByName(handleName); method.IsValid() {

				//必须是一个 func(*Context) 的回调函数, 否则否则框架无法处理
				if handle, ok := method.Interface().(HandleFunc); ok {
					b.Handle(methodName, path, handle)
				}
			}
		}
	}
}

// BindMethod fn 其实是一个 func(*Context) 回调函数
func (b *BluePrint) BindMethod(path string, fn any, mappings ...map[string]string) {
	mappings = append(mappings, map[string]string{
		"GET":     http.MethodGet,
		"POST":    http.MethodPost,
		"PUT":     http.MethodPut,
		"PATCH":   http.MethodPatch,
		"DELETE":  http.MethodDelete,
		"OPTIONS": http.MethodOptions,
		"HEAD":    http.MethodHead,
		"TRACE":   http.MethodTrace,
	})

	b.Bind(path, fn, mappings...)
}

func (b *BluePrint) Parent() *BluePrint {
	return b.parent
}

func (b *BluePrint) IsRoot() bool {
	return b.parent == nil
}

func (b BluePrint) XMLSerializer() color.Serializer {
	if b.xmlSerializer != nil {
		return b.xmlSerializer
	}

	if !b.IsRoot() {
		return b.Parent().XMLSerializer()
	}

	return nil
}

func (b *BluePrint) SetXMLSerializer(xml color.Serializer) {
	if xml == nil {
		panic("xmlSerializer can not be nil")
	}

	b.xmlSerializer = xml
}

func (b *BluePrint) JSONSerializer() color.Serializer {
	if b.jsonSerializer != nil {
		return b.jsonSerializer
	}

	if !b.IsRoot() {
		return b.Parent().JSONSerializer()
	}

	return nil
}

func (b *BluePrint) SetJSONSerializer(json color.Serializer) {
	if json == nil {
		panic("jsonSerializer can not be nil")
	}

	b.jsonSerializer = json
}

func (b *BluePrint) FileStorage() FileStorage {
	if b.fileStorage != nil {
		return b.fileStorage
	}

	if !b.IsRoot() {
		return b.Parent().FileStorage()
	}

	return nil
}

func (b *BluePrint) SetFileStorage(store FileStorage) {
	if store == nil {
		panic("store can not be nil")
	}

	b.fileStorage = store
}

func (b *BluePrint) Parsers() Parsers {
	if b.parsers != nil {
		return b.parsers
	}

	if !b.IsRoot() {
		return b.Parent().Parsers()
	}

	return nil
}

func (b *BluePrint) SetParsers(parsers Parsers) {
	if parsers == nil {
		panic("parsers can not be nil")
	}

	b.parsers = parsers
}

func (b BluePrint) Validator() validators.Validator {
	if b.validator != nil {
		return b.validator
	}

	if !b.IsRoot() {
		return b.Parent().Validator()
	}

	return nil
}

func (b *BluePrint) SetValidator(validate validators.Validator) {
	if validate == nil {
		panic("validator can not be nil")
	}

	b.validator = validate
}

func (b *BluePrint) Logger() Logger {
	if b.logger != nil {
		return b.logger
	}

	if !b.IsRoot() {
		return b.Parent().Logger()
	}

	return nil
}

func (b *BluePrint) SetLogger(log Logger) {
	if log == nil {
		panic("log can not be nil")
	}

	b.logger = log
}

func NewBluePrint() *BluePrint {
	return &BluePrint{}
}

// Static 注册静态文件处理器
func (b *BluePrint) Static(url, dir string, middleware ...HandleFunc) {
	if strings.Contains(url, "*") {
		panic("`url` should not have wildcards")
	}

	var server = http.FileServer(http.Dir(dir))
	var fileHandle = func(ctx *Context) {
		var path = ctx.Params.Get("static").Text()
		ctx.Request.URL.Path = path

		var p = filepath.Join(dir, path)
		if _, err := os.Stat(p); err != nil {
			ctx.matched = false
			ctx.Engine.NotFoundHandle(ctx)
			ctx.Abort()
			return
		}

		var ext = filepath.Ext(path)
		var cnt = mime.TypeByExtension(ext)
		if len(cnt) == 0 {
			cnt = "application/octet-stream"
		}

		//设置header
		ctx.SetHeader("Content-Type", cnt)
		server.ServeHTTP(ctx.ResponseWriter, ctx.Request)
	}

	middleware = append(middleware, fileHandle)
	if strings.HasSuffix(url, "static") && strings.HasSuffix(url, "/") {
		url += "/"
	}

	url += "*static"
	b.Handle(http.MethodGet, url, middleware...)
}

func (b *BluePrint) Default() *BluePrint {
	b.SetFileStorage(&LocalFileStorage{})                                                 // 设置存储器
	b.SetValidator(&validators.Default{})                                                 //设置验证器
	b.SetParsers(Parsers{JsonParser{}, FormParser{}, MultipartFormParser{}, XMLParser{}}) //设置解析器
	b.SetLogger(NewLogger())                                                              //设置日志处理器
	b.SetJSONSerializer(color.JsonSerializer{})                                           //设置json解析器
	b.SetXMLSerializer(color.XmlSerializer{})                                             //设置xml解析器

	return b
}
