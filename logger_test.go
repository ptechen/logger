package logger

import (
	"testing"
)

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
	data.LogPathDir = "."
	data.LogFileName = "log.log"
	data.IsConsole = false
	log := data.InitParams().InitLogger()
	log.Info().Msg("Hello World")

}
