package core

import (
	"fmt"
	"gin-core/core/color"
)

// 所以输出内容前缀
const _log_prefix = "[WY]"

type Starter interface {
	Start(e *Engine) error
}

type BannerStarter struct {
	Banner string
}

func (b BannerStarter) Start(e *Engine) error {
	fmt.Println(b.Banner)

	return nil
}

type UrlInfoStarter struct {
}

// Start 输出路由器对应路径所有的回调函数
func (u UrlInfoStarter) Start(e *Engine) error {
	for _method, nodes := range e.methodsTree {
		var m = color.FormatColor(97, _method)
		for _, node := range nodes {
			var count = color.BlueString(fmt.Sprintf("%d handlers", len(node.handles)))
			var path = color.YellowString(node.path)
			fmt.Printf("%-15s %-18s %s\n", _log_prefix, m, count, path)
		}
	}

	return nil
}
