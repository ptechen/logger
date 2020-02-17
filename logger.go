package main

import (
	"bufio"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"io"
	"os"
	"sync"
	"time"
)

type Logger struct {
	WriteFilePath   string    `json:"write_file_path"`
	IsConsole       bool      `json:"is_console"`
	Output          io.Writer `json:"output"`
	TimeFieldFormat string    `json:"time_field_format"`
	Caller          bool      `json:"caller"`
}

var logger *Logger
var once *sync.Once

func New() *Logger{
	once.Do(func() {
		logger = &Logger{
			WriteFilePath:   "",
			IsConsole:       false,
			Output:          nil,
			TimeFieldFormat: "",
			Caller:          true,
		}
	})
	return logger
}

func (p *Logger) InitNew() zerolog.Logger {
	t := time.NewTicker(time.Second * 3)
	if p.Caller {
		log.Logger = log.With().Caller().Logger()
	}
	if p.WriteFilePath != "" {
		f, err := os.Create(p.WriteFilePath) //创建文件
		if err != nil {
			panic("create file fail")
		}
		w := bufio.NewWriter(f) //创建新的 Writer 对象
		log.Output(w)
		go func() {
			for {
				select {
				case <- t.C:
					w.Flush()
				}
			}
		}()
	}
	if p.IsConsole == true {
		log.Output(os.Stdout)
	}
	return log.Logger
}

//func CallerName(skip int) (name, file string, line int, ok bool) {
//	var pc uintptr
//	if pc, file, line, ok = runtime.Caller(skip + 1); !ok {
//		return
//	}
//	name = runtime.FuncForPC(pc).Name()
//	return
//}
//
//func PrintCallerName(skip int, comment string) bool {
//	name, file, line, ok := CallerName(skip + 1)
//	if !ok {
//		return false
//	}
//	fmt.Printf("skip = %v, comment = %s\n", skip, comment)
//	fmt.Printf("  file = %v, line = %d\n", file, line)
//	fmt.Printf("  name = %v\n", name)
//	return true
//}
//
//var a = PrintCallerName(0, "main.a")
//var b = PrintCallerName(0, "main.b")
//
//func init() {
//	a = PrintCallerName(0, "main.init.a")
//}
//
//func init() {
//	b = PrintCallerName(0, "main.init.b")
//	func() {
//		b = PrintCallerName(0, "main.init.b[1]")
//	}()
//}
//
//func main() {
//	a = PrintCallerName(0, "main.main.a")
//	b = PrintCallerName(0, "main.main.b")
//	func() {
//		b = PrintCallerName(0, "main.main.b[1]")
//		func() {
//			b = PrintCallerName(0, "main.main.b[1][1]")
//		}()
//		b = PrintCallerName(0, "main.main.b[2]")
//	}()
//}
