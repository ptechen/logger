package logger

import (
	// depend
	_ "code.cloudfoundry.org/go-diodes"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/diode"
	"github.com/rs/zerolog/log"
	"io/ioutil"
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
	// TimeFormatDefault default time format 2006-01-02T15:04:05Z07:00
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

// LogParams is log config params.
type LogParams struct {
	// Log level
	Level int8 `yaml:"level" toml:"level"`

	// The terminal prints the log to enable the log color mode.
	Color                 bool   `yaml:"color" toml:"color"`
	logFilePath           string `yaml:"log_file_path" toml:"log_file_path"`
	LogPathDir            string `yaml:"log_path_dir" toml:"log_path_dir"`
	LogFileName           string `yaml:"log_file_name" toml:"log_file_name"`
	LogFileNameTimeFormat string `yaml:"log_file_name_time_format" toml:"log_file_name_time_format"`
	LogFileSize           string `yaml:"log_file_size" toml:"log_file_size"`
	logSize               int64  `json:"log_size"`
	LogExpDays            int64  `yaml:"log_exp_days" toml:"log_exp_days"`
	WriteChanSize         int    `yaml:"write_chan_size" toml:"write_chan_size"`
	IsConsole             bool   `yaml:"is_console" toml:"is_console"`
	TimeFieldFormat       string `yaml:"time_field_format" toml:"time_field_format"`
	// Enables logging of file names and lines.
	Caller bool `yaml:"caller" toml:"caller"`
	// Enable the default configuration.
	Default    bool   `yaml:"default" toml:"default"`
	ServerName string `yaml:"server_name" toml:"server_name"`
	logFile    *os.File

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

// New is create LogParams.
func New() *LogParams {
	once.Do(func() {
		logParams = &LogParams{
			Level:              -1,
			Color:              false,
			logFilePath:        "./log/log.log",
			IsConsole:          true,
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

// InitParams is init LogParams.
func (p *LogParams) InitParams() *LogParams {
	if p.logFilePath == "" && p.IsConsole == false {
		panic("config file err")
	}
	if p.Default {
		p.MessageFieldName = "msg"
		p.ErrorFieldName = "err"
		p.TimestampFieldName = "t"
		p.LevelFieldName = "l"
	}
	p.parseLogFileSize()
	p.setLogFilePath()
	p.setLogExpDays()
	p.setLogTimeFormat()
	p.setWriteChanSize()
	return p
}

func (p *LogParams) setWriteChanSize() {
	if p.WriteChanSize == 0 {
		p.WriteChanSize = 1000
	}
}

func (p *LogParams) setLogTimeFormat() {
	if p.LogFileNameTimeFormat == "" {
		p.LogFileNameTimeFormat = "2006-01-02 15:04:05"
	}
}

func (p *LogParams) setLogExpDays() {
	if p.LogExpDays == 0 {
		p.LogExpDays = 30
	}
}

func (p *LogParams) setLogFilePath() {
	if p.LogPathDir == "" {
		p.LogPathDir = "/opt/log"
	}

	if p.LogFileName == "" {
		p.LogFileName = "log.log"
	}
	p.logFilePath = p.LogPathDir + "/" + p.LogFileName
}

func (p *LogParams) setZeroTimeFieldFormat() *LogParams {
	zerolog.TimeFieldFormat = p.TimeFieldFormat
	return p
}

// InitLog is init log.
func (p *LogParams) InitLog() *zerolog.Logger {
	onceLog.Do(func() {
		logger = &log.Logger
		p.setLogFieldsName()
		p.caller()
		p.output()
	})
	monitor(p)
	return logger
}

func (p *LogParams) setLogFieldsName() {
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
	if p.logFilePath != "" && p.IsConsole == false {
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
	times := 0
lab:
	p.logFile, err = os.OpenFile(p.logFilePath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		if times == 3 {
			panic("create log file failed")
		}
		times += 1
		logger.Err(err).Str("init_file", "failed").Msgf("%#v", p.logFile)
		goto lab
	}
	logger.Info().Str("init_file", "success").Msgf("%#v", p.logFile)
}

func (p *LogParams) fileSize() int64 {
	f, e := os.Stat(p.logFilePath)
	if e != nil {
		return 0
	}
	return f.Size()
}

func (p *LogParams) isExist() bool {
	_, err := os.Stat(p.logFilePath)
	return err == nil || os.IsExist(err)
}

func monitor(params *LogParams) {
	t := time.NewTicker(time.Second * 3)
	deleted := time.NewTicker(time.Hour * 24)

	go func() {
		defer t.Stop()
		defer deleted.Stop()
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

			case <-deleted.C:
				params.deletedData()
			}
		}
	}()
}

func (p *LogParams) rename2File() {
	now := time.Now()
	newLogFileName := fmt.Sprintf("%s.%s", p.logFilePath, now.Format(p.LogFileNameTimeFormat))
	_ = os.Rename(p.logFilePath, newLogFileName)
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
	files, _ := ioutil.ReadDir(p.LogPathDir)
	for _, file := range files {
		if file.IsDir() {

		} else {
			logger.Info().Msg(file.Name())
			if file.Name() != p.LogFileName && strings.Contains(file.Name(), p.LogFileName) {
				createTime := strings.Split(file.Name(), p.LogFileName+".")[1]
				date, err := time.Parse(p.LogFileNameTimeFormat, createTime)
				if err != nil {
					logger.Err(err).Msg("log file time format err")
					continue
				}
				dateUnix := date.Unix()
				currentUnix := time.Now().Unix()
				if currentUnix-dateUnix > p.LogExpDays*60*60*24 {
					currentFileName := p.LogPathDir + "/" + file.Name()
					err = os.Remove(currentFileName)
					if err != nil {
						logger.Err(err).Msgf("remove %s failed", currentFileName)
					}
					logger.Info().Msgf("remove %s success", currentFileName)
				}
			}
		}
	}

}
