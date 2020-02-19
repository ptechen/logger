package logger

import (
	_ "code.cloudfoundry.org/go-diodes"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/diode"
	"github.com/rs/zerolog/log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	logParams *LogParams
	once      sync.Once
	onceLog   sync.Once
	logger    *zerolog.Logger
)

const (
	DATEFORMAT = "2006-01-02"

	TimeFormatDefault = time.RFC3339
	// TimeFormatUnix defines a time format that makes time fields to be
	// serialized as Unix timestamp integers.
	TimeFormatUnix = ""

	// TimeFormatUnixMs defines a time format that makes time fields to be
	// serialized as Unix timestamp integers in milliseconds.
	TimeFormatUnixMs = "UNIXMS"

	// TimeFormatUnixMicro defines a time format that makes time fields to be
	// serialized as Unix timestamp integers in microseconds.
	TimeFormatUnixMicro = "UNIXMICRO"
)

const (
	// TraceLevel defines trace log level.
	TraceLevel = iota - 1
	// DebugLevel defines debug log level.
	DebugLevel
	// InfoLevel defines info log level.
	InfoLevel
	// WarnLevel defines warn log level.
	WarnLevel
	// ErrorLevel defines error log level.
	ErrorLevel
	// FatalLevel defines fatal log level.
	FatalLevel
	// PanicLevel defines panic log level.
	PanicLevel
	// NoLevel defines an absent log level.
	NoLevel
	// Disabled disables the logger.
	Disabled
)

type LogParams struct {
	// Log level
	Level           int8   `yaml:"level" toml:"level"`

	// The terminal prints the log to enable the log color mode.
	Color           bool   `yaml:"color" toml:"color"`
	LogFilePath     string `yaml:"log_file_path" toml:"log_file_path"`
	LogTimeFormat   string `yaml:"log_time_format" toml:"log_time_format"`
	LogFileSize     string `yaml:"log_file_size" toml:"log_file_size"`
	logSize         int64  `json:"log_size"`
	LogExpDays      int    `yaml:"log_exp_days" toml:"log_exp_days"`
	WriteChanSize   int    `yaml:"write_chan_size" toml:"write_chan_size"`
	IsConsole       bool   `yaml:"is_console" toml:"is_console"`
	TimeFieldFormat string `yaml:"time_field_format" toml:"time_field_format"`
	// Enables logging of file names and lines.
	Caller          bool   `yaml:"caller" toml:"caller"`
	// Enable the default configuration.
	Default         bool   `yaml:"default" toml:"default"`
	ServerName      string `yaml:"server_name" toml:"server_name"`
	logFile         *os.File

	// TimestampFieldName is the field name used for the timestamp field.
	TimestampFieldName string `json:"timestamp_field_name"`

	// LevelFieldName is the field name used for the level field.
	LevelFieldName string `json:"level_field_name"`

	// MessageFieldName is the field name used for the message field.
	MessageFieldName string `json:"message_field_name"`

	// ErrorFieldName is the field name used for error fields.
	ErrorFieldName string `json:"error_field_name"`

	// CallerFieldName is the field name used for caller field.
	CallerFieldName string `json:"caller_field_name"`

	// ErrorStackFieldName is the field name used for error stacks.
	ErrorStackFieldName string `json:"error_stack_field_name"`
}

func New() *LogParams {
	once.Do(func() {
		logParams = &LogParams{
			Level:              -1,
			Color:              false,
			LogFilePath:        "./log/log.log",
			IsConsole:          true,
			TimeFieldFormat:    "",
			WriteChanSize:      1000,
			LogExpDays:         30,
			Caller:             true,
			TimestampFieldName: "t",
			LevelFieldName:     "l",
			MessageFieldName:   "msg",
			ErrorFieldName:     "err",
		}
	})
	return logParams
}

func (p *LogParams) InitParams() *LogParams {
	if p.LogFilePath == "" && p.IsConsole == false {
		panic("config file err")
	}
	if p.Default {
		p.MessageFieldName = "msg"
		p.ErrorFieldName = "err"
		p.TimestampFieldName = "t"
		p.LevelFieldName = "l"
	}
	p.parseLogFileSize()
	return p
}

func (p *LogParams) setZeroTimeFieldFormat() *LogParams {
	zerolog.TimeFieldFormat = p.TimeFieldFormat
	return p
}

func (p *LogParams) InitLog() *zerolog.Logger {
	onceLog.Do(func() {
		logger = &log.Logger
		p.setFileName()
		p.caller()
		p.output()
	})
	monitor(p)
	return logger
}

func (p *LogParams) setFileName() {
	p.setZeroTimeFieldFormat()

	if p.TimestampFieldName != "" {
		zerolog.TimestampFieldName = p.TimestampFieldName
	}
	if p.LevelFieldName != "" {
		zerolog.LevelFieldName = p.LevelFieldName
	}

	if p.MessageFieldName != "" {
		zerolog.MessageFieldName = p.MessageFieldName
	}

	if p.ErrorFieldName != "" {
		zerolog.ErrorFieldName = p.ErrorFieldName
	}

	if p.CallerFieldName != "" {
		zerolog.ErrorFieldName = p.ErrorFieldName
	}

	if p.ErrorStackFieldName != "" {
		zerolog.ErrorStackFieldName = p.ErrorStackFieldName
	}
}

func (p *LogParams) output() {
	if p.LogFilePath != "" && p.IsConsole == false {
		p.initFile()
		w := diode.NewWriter(p.logFile, p.WriteChanSize, 10*time.Millisecond, func(missed int) {
			logger.Warn().Msgf("Logger Dropped %d messages", missed)
		})
		*logger = (logger.Output(w)).Level(zerolog.Level(p.Level))
	}

	if p.IsConsole {
		if p.Color {
			*logger = logger.Output(zerolog.ConsoleWriter{Out: os.Stdout}).Level(zerolog.Level(p.Level))
		} else {
			*logger = logger.Output(os.Stdout).Level(zerolog.Level(p.Level))
		}
	}
}

func (p *LogParams) caller() {
	if p.Caller {
		*logger = logger.With().Caller().Logger()
	}
}

func (p *LogParams) initFile() {
	var err error
lab:
	p.logFile, err = os.OpenFile(p.LogFilePath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		logger.Err(err).Str("init_file", "failed").Msgf("%#v", p.logFile)
		goto lab
	}
	logger.Info().Str("init_file", "success").Msgf("%#v", p.logFile)
}

func (p *LogParams) fileSize() int64 {
	f, e := os.Stat(p.LogFilePath)
	if e != nil {
		return 0
	}
	return f.Size()
}

func (p *LogParams) isExist() bool {
	_, err := os.Stat(p.LogFilePath)
	return err == nil || os.IsExist(err)
}

func monitor(params *LogParams) {
	t := time.NewTicker(time.Second * 3)
	day := time.NewTicker(time.Hour * 24)

	go func() {
		defer t.Stop()
		defer day.Stop()
		for {
			select {
			case <-t.C:
				logger.Info().Msg("check file size")
				isExist := params.isExist()
				if !isExist {
					params.output()
				}
				size := params.fileSize()
				logger.Info().Str("size", fmt.Sprintf("%d", size)).Msg("check file size")
				if size > params.logSize {
					logger.Info().Msg("rename log file")
					params.rename2File()
					params.output()
				}
			case <-day.C:

			}
		}
	}()
}

func (p *LogParams) rename2File() {
	now := time.Now()
	if p.LogTimeFormat == "" {
		p.LogTimeFormat = "2006-01-02 15:04:05"
	}
	newLogFileName := fmt.Sprintf("%s.%s", p.LogFilePath, now.Format(p.LogTimeFormat))
	_ = os.Rename(p.LogFilePath, newLogFileName)
}

func (p *LogParams) parseLogFileSize() {
	if p.LogFileSize == "" {
		p.LogFileSize = "1G"
	}
	if strings.Contains(p.LogFileSize, "G") {
		n, _ := strconv.Atoi(strings.Split(p.LogFileSize, "G")[0])
		p.logSize = int64(n) * 1024 * 1024 * 1024

	} else if strings.Contains(p.LogFileSize, "MB") {
		n, _ := strconv.Atoi(strings.Split(p.LogFileSize, "MB")[0])
		p.logSize = int64(n) * 1024 * 1024
	}
}

func (p *LogParams) deletedData() {

}
