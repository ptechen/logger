package logger

import (
	"testing"
)



//
//func BenchmarkLogParams_New(b *testing.B) {
//	conf := config.Flag().SetEnv("test")
//	params := &Config{}
//	conf.ParseFile(&params)
//	log := params.Log.SetLevel(TraceLevel).New().InitLog()
//	for i := 0; i < b.N; i ++{
//		log.Info().
//			Str("foo", "bar").
//			Int("n", i).
//			Msg("hello world")
//	}
//}

func TestNew(t *testing.T) {
	data := New()
	if data.IsConsole == false {
		t.Error("New err")
	}
}

func TestLogParams_InitParams(t *testing.T) {
	data := New()
	data = data.InitParams()
	if data.LevelFieldName != "l" {
		t.Error("InitParams err")
	}
}

func TestLogParams_InitLog(t *testing.T) {
	data := New()
	log := data.InitParams().InitLog()
	log.Info().Msg("Hello World")
}

