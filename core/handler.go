package core

import "net/http"

type HandleFunc func(ctx *Context)

type handleFuncNode struct {
	HandleFunc HandleFunc
	BluePrint  *BluePrint
}

// HandleNotFound 404业务
func HandleNotFound(ctx *Context) {
	http.NotFound(ctx.ResponseWriter, ctx.Request)
}

/*// RawHandlerFunc 传入 func(ResponseWriter, *Request) 返回 func(*Context)
func RawHandlerFunc(handler http.HandlerFunc) HandleFunc {
	return func(ctx *Context) {
		SetContextIntoRequest(ctx)

		handler(ctx.ResponseWriter, ctx.Request)
	}
}*/

func RawHandlerFuncGroup(handlers ...http.HandlerFunc) []HandleFunc {
	var middleware = make([]HandleFunc, len(handlers))

	for idx, handler := range handlers {
		//middleware[idx] = RawHandlerFunc(handler)
		middleware[idx] = func(ctx *Context) {

			//转换请求上下文
			SetContextIntoRequest(ctx)

			//执行http请求
			handler(ctx.ResponseWriter, ctx.Request)
		}
	}

	return middleware
}

// RecoverHandler 处理崩溃业务
func RecoverHandler(handler func(ctx *Context, rec any)) HandleFunc {
	return func(ctx *Context) {
		defer func() {
			if err := recover(); err != nil {
				handler(ctx, err)
			}
		}()

		ctx.Next()
	}
}
