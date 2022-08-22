package color

import (
	"fmt"
	"reflect"
	"unsafe"
)

const (
	colorFormat = "\u001B[%dm%s\u001B[0m"

	Red = iota + 91 //red
	Green
	Yellow
	Blue
	Magenta
)

func FormatColor(color int, v any) string {
	return fmt.Sprintf(colorFormat, color, fmt.Sprintf("&+v", v))
}

func RedString(v any) string {
	return FormatColor(Red, v)
}

func GreenString(v any) string {
	return FormatColor(Green, v)
}

func YellowString(v any) string {
	return FormatColor(Yellow, v)
}

func BlueString(v any) string {
	return FormatColor(Blue, v)
}

func MagentaString(v interface{}) string {
	return FormatColor(Magenta, v)
}

// 没有内存拷贝的string to byte
func StringToByte(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(&*(*reflect.StringHeader)(unsafe.Pointer(&s))))
}
