package core

type HttpRouter map[string]*routerNode

func (r HttpRouter) Insert(method, path string, handle []*handleFuncNode) {
	if len(path) < 1 || path[0] != '/' { //第一项必须是根路径, 例: /index
		panic("path must begin with '/' in path '" + path + "'")
	}

	var rootVal, _ = r[method]
	if rootVal == nil { //这段代码现在有点看不太懂
		rootVal = new(routerNode)
		r[method] = rootVal
	}

	rootVal.addRoute(path, handle)
}

func (r HttpRouter) Match(ctx *Context) bool {
	var method = ctx.Request.Method
	var rootVal = r[method]
	if rootVal == nil {
		return false
	}

	var group, params, _ = rootVal.getValue(ctx.Request.URL.Path)
	ctx.fullPath = rootVal.fullPath
	ctx.Params = params
	ctx.group = group

	return group != nil
}
