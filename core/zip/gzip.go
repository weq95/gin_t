package zip

import (
	"compress/gzip"
	"gin-core/core"
	"net/http"
	"strings"
)

type gzipWriter struct {
	http.ResponseWriter
	zip *gzip.Writer
}

func (w *gzipWriter) Write(data []byte) (int, error) {
	return w.zip.Write(data)
}

// GZip Gzip 是用于 gzip 压缩的中间件，如果客户端接受 gzip 编码，
// 它将压缩响应正文，参数是压缩级别，
// 从 gzip.BestSpeed 到 gzip.BestCompression 中选择
func GZip(level int) core.HandleFunc {
	return func(ctx *core.Context) {
		if !strings.Contains(ctx.Request.Header.Get("Accept-Encoding"), "zip") {
			return
		}

		var zipWrite, err = gzip.NewWriterLevel(ctx.ResponseWriter, level)
		if err != nil {
			ctx.Logger().Error(err)
			return
		}

		ctx.SetHeader("Content-Encoding", "gzip")
		ctx.SetHeader("Vary", "Accept-Encoding")

		//重新设置response_write
		ctx.ResponseWriter = &gzipWriter{
			ResponseWriter: ctx.ResponseWriter,
			zip:            zipWrite,
		}

		defer func() {
			ctx.SetHeader("Content-Length", "0")
			if err = zipWrite.Close(); err != nil {
				ctx.Logger().Error(err)
			}
		}()
		ctx.Next()
	}
}
