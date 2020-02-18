package logger

import (
	"github.com/ptechen/config"
	"testing"
	"time"
)

type Config struct {
	Log *LogParams
}

func TestLogger_Trace(t *testing.T) {
	conf := config.Flag().SetEnv("test")
	params := &Config{}
	conf.ParseFile(&params)
	log := params.Log.New().InitLog()
	for i := 0; i < 100000; i ++{
		if i % 1000 == 0 {
			time.Sleep(time.Second)
		}
		log.Trace().
			Str("foo", "bar").
			Int("n", 123).
			Msg("hello world")
	}
	// Output: {"l":"trace","foo":"bar","n":123,"msg":"hello world"}
}

//func TestLogParams_InitLog(t *testing.T) {
//		conf := config.Flag().SetEnv("test")
//		params := &Config{}
//		conf.ParseFile(&params)
//		log := params.Log.New().InitLog()
//		log.Info().
//			Str("foo", "bar").
//			Int("n", 123).
//			Msg("hello world")
//}