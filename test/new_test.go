package test

import (
	"gin-core/core"
	"net/http"
	"net/http/httptest"
	"testing"
)

func runRequest(B *testing.B, r *core.Engine, method, path string) {
	// create fake request
	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		panic(err)
	}
	w := httptest.NewRecorder()
	B.ReportAllocs()
	B.ResetTimer()
	for i := 0; i < B.N; i++ {
		r.ServeHTTP(w, req)
	}

}

func BenchmarkJson(b *testing.B) {
	for i := 0; i < b.N; i++ {
		router := core.New()
		router.GET("/json", func(c *core.Context) { c.JSON(map[string]any{"hello": "world"}) })
		router.TestInit()
		runRequest(b, router, "GET", "/json")
	}
}

package cnet

import (
	"fmt"
	"io"
	"net"
)

const MaxBufferSize = 8192 //单次读取最大长度
const HeaderLen = 6        // 包头长度

type Reader struct {
	Conn      net.Conn
	Buff      []byte
	Start     uint //数据读取开始位置
	End       uint //数据读取结束位置
	BuffLen   uint //数据接收缓冲区大小
	HeaderLen uint //包头长度
}

func NewReader(conn net.Conn) *Reader {
	return &Reader{
		Conn:      conn,
		Buff:      make([]byte, MaxBufferSize),
		Start:     0,
		End:       0,
		BuffLen:   MaxBufferSize,
		HeaderLen: HeaderLen,
	}
}

func (r *Reader) Read() error {
	for true {
		r.Move()

		if r.End == r.BuffLen {
			// 缓冲区无法容纳一条消息的长度
			return fmt.Errorf("one message is too large: %v", r)
		}

		var (
			msgLen int
			err    error
		)
		msgLen, err = r.Conn.Read(r.Buff[r.End:])
		if err != nil && err != io.EOF {
			return err
		}

		r.End += uint(msgLen)
		if err = r.GetData(); err != nil {
			fmt.Println(err)
			return err
		}
	}

	return io.EOF
}

func (r *Reader) Move() {
	if r.Start == 0 {
		return
	}

	// 将 src[r.Buff[r.Start:r.End]] 复制到 dst[r.Buff]
	copy(r.Buff, r.Buff[r.Start:r.End])
	r.End -= r.Start
	r.Start = 0
}

func (r *Reader) GetData() error {
	if (r.End - r.Start) < r.HeaderLen {
		// 包头的长度不够, 继续接收
		return nil
	}

	//读取包头数据
	var headerData = r.Buff[r.Start : r.Start+r.HeaderLen]
	var contentLen = uint(headerData[r.HeaderLen-1])
	var code = int16(uint(headerData[r.HeaderLen-2]))
	if r.End-r.Start-r.HeaderLen < contentLen {
		// 包体长度不够, 继续接收
		return nil
	}

	// 读取包体数据
	var start = r.Start + r.HeaderLen
	var bodyData = r.Buff[start : start+contentLen]
	go callback(code, headerData, bodyData[:])
	// 数据读取完成 start 后移
	r.Start += r.HeaderLen + contentLen

	return r.GetData()
}

func callback(code int16, header, body []byte) {
	fmt.Println("header: ", string(header))
	fmt.Println("body: \n", string(body))
}


package main

import (
	"GoFishing/cnet"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
)

var bodies = []string{
	`{"rankid":1,"userid":2000881,"nickname":"包子要多读书","score":28492800,"avatar":0,"avatar_url":"https://hidog.oss-cn-hangzhou.aliyuncs.com/fish/pic/tx33.png"}`,
	`{"ID":18607,"list":[{"rankid":3,"userid":2000847,"nickname":"Player_fMP7","score":10940,"avatar":0,"avatar_url":"https://hidog.oss-cn-hangzhou.aliyuncs.com/fish/pic/tx13.png"}]}`,
	`{"ID":18603,"flag":1,"msg":"奖励已失效"}`,
	`{"ID":18605,"flag":0,"module":{"data":[[{"c_scheme":0,"module_id":1021,"st":2,"t_scheme":1},{"c_scheme":0,"module_id":1022,"st":2,"t_scheme":0},{"c_scheme":0,"module_id":1023,"st":1,"t_scheme":0}],[{"c_scheme":0,"module_id":2021,"st":1,"t_scheme":5000}]],"wid":13001},"msg":"success"}`,
}

var addr = "localhost:8080"

func main001() {
	// 下载同步配置文件

	// 初始化 DB, CS 服务器长链接

	// 初始化 websocket

	var conn, err = net.Dial("tcp", addr)
	if err != nil {
		fmt.Println(err)
		return
	}
	for i := 2000; i < 2100; i++ {
		var message = make([]byte, 0)
		for _, body := range bodies {
			// 协议号 + 长度
			var l = byte(len(body))
			message = append(message, append([]byte(strconv.Itoa(i)), l)...)
			// 主要内容
			message = append(message, []byte(body)...)

			if _, err = conn.Write(message); err != nil {
				fmt.Println("数据发送失败: ", err)
			}
		}

		time.Sleep(time.Second * 5)
	}

	_, _ = os.Stdin.Read(make([]byte, 1))
}

func main() {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Println(err)
		return
	}

	defer func() {
		_ = listener.Close()
	}()

	go func() {
		<-time.After(time.Second * 5)
		fmt.Println("---------- 2.开始发送数据 ----------")
		main001()
	}()
	for true {
		var conn, err001 = listener.Accept()
		if err001 != nil {
			break
		}

		go handle(conn)
	}
}

func handle(conn net.Conn) {
	defer func() {
		_ = conn.Close()
	}()

	fmt.Println("---------- 1.开始读取数据 ----------")
	var reader = cnet.NewReader(conn)
	var err = reader.Read()
	if err != nil {
		fmt.Println("数据读取错误: ", err)
	}
}

https://studygolang.com/articles/25278
https://cloud.tencent.com/developer/article/1801065
