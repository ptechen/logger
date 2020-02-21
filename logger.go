package logger

import (
	"bytes"
	// depend
	_ "code.cloudfoundry.org/go-diodes"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/diode"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	logParams *LogParams
	once      sync.Once
	onceLog   sync.Once
	Logger    *zerolog.Logger
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
	LogColor              bool   `yaml:"log_color" toml:"log_color"`
	logFilePath           string
	// Directory where logs are saved.
	LogPathDir            string `yaml:"log_path_dir" toml:"log_path_dir"`
	// Log filename
	LogFileName           string `yaml:"log_file_name" toml:"log_file_name"`
	// The rename log file is in a date format.
	LogFileNameTimeFormat string `yaml:"log_file_name_time_format" toml:"log_file_name_time_format"`
	// Log file size ,example "1G" or "512MB"
	LogFileSize           string `yaml:"log_file_size" toml:"log_file_size"`
	logSize               int64
	// Number of days the log is kept.
	LogExpDays            int64  `yaml:"log_exp_days" toml:"log_exp_days"`
	// Log chan size.
	LogChanSize           int    `yaml:"log_chan_size" toml:"log_chan_size"`
	// Enable terminal print log.
	IsConsole             bool   `yaml:"is_console" toml:"is_console"`
	// Log time field format.
	TimeFieldFormat       string `yaml:"time_field_format" toml:"time_field_format"`
	// Enables logging of file names and lines.
	Caller bool `yaml:"caller" toml:"caller"`
	// Enable the default configuration.
	Default    bool   `yaml:"default" toml:"default"`

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
			LogColor:              false,
			logFilePath:        "./log/log.log",
			IsConsole:          true,
			LogChanSize:      1000,
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
	if p.LogChanSize == 0 {
		p.LogChanSize = 1000
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
func (p *LogParams) InitLogger() *zerolog.Logger {
	onceLog.Do(func() {
		Logger = &log.Logger
		p.setLogFieldsName()
		p.caller()
		p.output()
	})
	monitor(p)
	return Logger
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
		w := diode.NewWriter(p.logFile, p.LogChanSize, 10*time.Millisecond, func(missed int) {
			Logger.Warn().Msgf("Logger Dropped %d messages", missed)
		})
		*Logger = (Logger.Output(w)).Level(zerolog.Level(p.Level))
	}

	if p.IsConsole {
		if p.LogColor {
			*Logger = Logger.Output(zerolog.ConsoleWriter{Out: os.Stdout}).Level(zerolog.Level(p.Level))
		} else {
			*Logger = Logger.Output(os.Stdout).Level(zerolog.Level(p.Level))
		}
	}
}

func (p *LogParams) caller() {
	if p.Caller {
		*Logger = Logger.With().Caller().Logger()
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
		Logger.Err(err).Str("init_file", "failed").Msgf("%#v", p.logFile)
		goto lab
	}
	Logger.Info().Str("init_file", "success").Msgf("%#v", p.logFile)
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
	del := time.NewTicker(time.Hour * 24)

	go func() {
		defer t.Stop()
		defer del.Stop()
		for {
			select {

			case <-t.C:
				isExist := params.isExist()
				if !isExist {
					params.output()
				}
				size := params.fileSize()
				if size > params.logSize {
					Logger.Info().Msg("rename log file")
					params.rename2File()
					params.output()
				}

			case <-del.C:
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
			// DO
		} else {
			if file.Name() != p.LogFileName && strings.Contains(file.Name(), p.LogFileName) {
				createTime := strings.Split(file.Name(), p.LogFileName+".")[1]
				date, err := time.Parse(p.LogFileNameTimeFormat, createTime)
				if err != nil {
					Logger.Err(err).Msg("log file time format err")
					continue
				}
				dateUnix := date.Unix()
				currentUnix := time.Now().Unix()
				if currentUnix-dateUnix > p.LogExpDays*60*60*24 {
					currentFileName := p.LogPathDir + "/" + file.Name()
					err = os.Remove(currentFileName)
					if err != nil {
						Logger.Err(err).Msgf("remove %s failed", currentFileName)
					}
					Logger.Info().Msgf("remove %s success", currentFileName)
				}
			}
		}
	}
}

var l = len("goroutine ")

func GoroutineID() string {
	var buf [32]byte
	n := runtime.Stack(buf[:], false)

	b := bytes.NewBuffer(buf[l:n])
	s, _ := b.ReadString(' ')

	return strings.TrimSpace(s)
}