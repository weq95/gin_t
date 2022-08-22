package core

import (
	"net/http"
	"sync"
)

type Engine struct {
	*BluePrint                         //所有的处理程序都将注册到其中
	Router          HttpRouter         //分发请求
	NotFoundHandle  func(ctx *Context) //404页面
	interceptors    []*handleFuncNode  //中间件处理器
	starters        []Starter          //服务启动运行时, 就是一堆接口
	Warehouse       Warehouse          //储存的其他信息
	MultipartMemory int64              //request max body size
	pool            sync.Pool
	server          *http.Server
}

func (e *Engine) dispatchContext() *Context {
	return &Context{
		Engine: e,
	}
}

// AddInterceptors 添加中间件
func (e *Engine) AddInterceptors(middlewares ...HandleFunc) {
	var groups = make([]*handleFuncNode, len(middlewares))

	for _, handleFuncCtx := range middlewares {
		groups = append(groups, &handleFuncNode{
			HandleFunc: handleFuncCtx, //中间键加入其中
			BluePrint:  e.BluePrint,
		})
	}

	e.interceptors = append(e.interceptors, groups...)
}

// AddStarter 程序启动时会调用
func (e *Engine) AddStarter(starts ...Starter) {
	e.starters = append(e.starters, starts...)
}

func (c *Engine) TestInit() error {
	return c.init()
}

// init 初始化路由器
func (e *Engine) init() error {
	for method, nodes := range e.methodsTree {
		for _, node := range nodes { //对应的每个请求路径
			var hns = []*handleFuncNode{}
			var handles = append(e.middleware, node.handles...)

			for _, handle := range handles { // 里边放置的多个回调函数
				hns = append(hns, &handleFuncNode{
					HandleFunc: handle,
					BluePrint:  node.blueprint,
				})
			}

			e.Router.Insert(method, node.path, hns)
		}
	}

	for _, starter := range e.starters {
		if err := starter.Start(e); err != nil {
			return err
		}
	}

	return nil
}

func (e *Engine) Run(addr string) error {
	return e.ListenAndServe(addr)
}

// setup 尝试初始化并启动服务
func (e *Engine) setup() error {
	if err := e.init(); err != nil {
		return err
	}

	e.server = &http.Server{Handler: e}
	return nil
}

// ListenAndServe 启动 HTTP 服务
func (e *Engine) ListenAndServe(addr string) error {
	if err := e.setup(); err != nil {
		return err
	}

	e.server.Addr = addr

	return e.server.ListenAndServe()
}

// ListenAndServeTLS 启动 HTTPS 服务
func (e *Engine) ListenAndServeTLS(addr, certFile, keyFile string) error {
	if err := e.setup(); err != nil {
		return err
	}

	e.server.Addr = addr

	return e.server.ListenAndServeTLS(certFile, keyFile)
}

// ServeHTTP 实现 HTTP 接口
func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var ctx = e.pool.Get().(*Context)
	ctx.Request = r
	ctx.ResponseWriter = w

	//查找所有的处理程序
	ctx.matched = e.Router.Match(ctx)
	if ctx.matched {
		if len(e.interceptors) != 0 {
			ctx.group = append(e.interceptors, ctx.group...)
		}
	} else {
		ctx.group = []*handleFuncNode{
			{
				HandleFunc: e.NotFoundHandle,
				BluePrint:  e.BluePrint,
			},
		}
	}

	//开始下一项处理
	ctx.start()

	//设置上下文
	if !ctx.escape {
		ctx.reset()
		e.pool.Put(ctx)
	}
}

func (e *Engine) Server() *http.Server {
	return e.server
}

func (e *Engine) CloneServer() *http.Server {
	//这里的clone其实是有问题的, 因为切片数据没有实现真正的clone
	//slice会互相影响
	return &http.Server{Handler: e}
}

func New() *Engine {
	var engine = &Engine{
		Router:          HttpRouter{},
		BluePrint:       NewBluePrint().Default(), //初始化各类解析器
		NotFoundHandle:  HandleNotFound,
		Warehouse:       warehouse{},            //其他数据存储器
		MultipartMemory: defaultMultipartMemory, //默认请求大小限制
	}

	engine.pool = sync.Pool{
		New: func() any {
			//这里其实是获取默认的context
			return engine.dispatchContext()
		},
	}

	return engine
}

func Default() *Engine {
	var engine = New()
	engine.AddStarter(&BannerStarter{
		Banner: "Logo(其实就是一个logo)",
	}, &UrlInfoStarter{})

	return engine
}
