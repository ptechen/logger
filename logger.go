package logger

import (
	_ "code.cloudfoundry.org/go-diodes"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/diode"
	"github.com/rs/zerolog/log"
	"os"
	"sync"
	"time"
)

var (
	dailyRolling        bool = true
	checkMustRenameTime int64
	maxFileCount        int32
	maxFileSize         int64 = 1024
	logParams           *LogParams
	once                sync.Once
	onceLog             sync.Once
	logger              *zerolog.Logger
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

type LogParams struct {
	Level           int8   `yaml:"level" toml:"level"`
	Color           bool   `yaml:"color" toml:"color"`
	LogFilePath     string `yaml:"log_file_path" toml:"log_file_path"`
	LogTimeFormat   string `yaml:"log_time_format" toml:"log_time_format"`
	LogFileSize     int64  `yaml:"log_file_size" toml:"log_file_size"`
	IsConsole       bool   `yaml:"is_console" toml:"is_console"`
	TimeFieldFormat string `yaml:"time_field_format" toml:"time_field_format"`
	Caller          bool   `yaml:"caller" toml:"caller"`
	ServerName      string `yaml:"server_name" toml:"server_name"`
	Default         bool   `yaml:"default" toml:"default"`
	_suffix         int
	_date           *time.Time
	mu              *sync.RWMutex
	logfile         *os.File

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
			LogFilePath:        "log.log",
			IsConsole:          true,
			TimeFieldFormat:    "",
			Caller:             true,
			TimestampFieldName: "t",
			LevelFieldName:     "l",
			MessageFieldName:   "msg",
			ErrorFieldName:     "err",
		}
	})
	return logParams
}

func (p *LogParams) New() *LogParams {
	if p.LogFilePath == "" && p.IsConsole == false {
		panic("config file err")
	}
	if p.Default {
		p.MessageFieldName = "msg"
		p.ErrorFieldName = "err"
		p.TimestampFieldName = "t"
		p.LevelFieldName = "l"
	}
	return p
}

func (p *LogParams) SetLevel(level int8) *LogParams {
	p.Level = level
	return p
}

func (p *LogParams) SetColor(color bool) *LogParams {
	p.Color = color
	return p
}

func (p *LogParams) SetWriteFilePath(logFilePath string) *LogParams {
	p.LogFilePath = logFilePath
	return p
}

func (p *LogParams) SetIsConsole(isConsole bool) *LogParams {
	p.IsConsole = isConsole
	return p
}

func (p *LogParams) SetTimeFieldFormat(timeFieldFormat string) *LogParams {
	p.TimeFieldFormat = timeFieldFormat
	return p
}

func (p *LogParams) SetZeroTimeFieldFormat() *LogParams {
	zerolog.TimeFieldFormat = p.TimeFieldFormat
	return p
}

func (p *LogParams) SetCaller(caller bool) *LogParams {
	p.Caller = caller
	return p
}

func (p *LogParams) setFileName() {

	p.SetZeroTimeFieldFormat()

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
		w := diode.NewWriter(p.logfile, 1000000, 10*time.Millisecond, func(missed int) {
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

//func (p *LogParams) isMustRename() bool {
//	now := time.Now()
//	//3秒检查一次，不然太频繁
//	if checkMustRenameTime != 0 && now.Unix()-checkMustRenameTime < 3 {
//		return false
//	}
//	checkMustRenameTime = now.Unix()
//	//if dailyRolling {
//	//	t, _ := time.Parse(DATEFORMAT, now.Format(DATEFORMAT))
//	//	if t.After(*p._date) {
//	//		return true
//	//	}
//	//} else {
//	//	if maxFileCount > 1 {
//	//		if fileSize(p.FilePath) >= maxFileSize {
//	//			return true
//	//		}
//	//	}
//	//}
//	if fileSize(p.LogFilePath) >= maxFileSize {
//		return true
//	}
//	return false
//}

//func (p *LogParams) rename() {
//	if dailyRolling {
//		t, _ := time.Parse(DATEFORMAT, time.Now().Format(DATEFORMAT))
//		p._date = &t
//		fn := p.LogFilePath + p._date.Format(DATEFORMAT) + fmt.Sprintf("-%d", maxFileCount)
//		if !isExist(fn) {
//			if p.logfile != nil {
//				_ = p.logfile.Close()
//			}
//			err := os.Rename(p.LogFilePath, fn)
//			if err != nil {
//				logger.Err(err).Msg("rename log file failed")
//			}
//			maxFileCount += 1
//			p.output()
//		}
//	} else {
//		p.coverNextOne()
//	}
//}

func (p *LogParams) nextSuffix() int {
	return int(p._suffix%int(maxFileCount) + 1)
}

//func (p *LogParams) coverNextOne() {
//	p._suffix = p.nextSuffix()
//	if p.logfile != nil {
//		_ = p.logfile.Close()
//	}
//	if isExist(p.LogFilePath + "." + strconv.Itoa(int(p._suffix))) {
//		_ = os.Remove(p.LogFilePath + "." + strconv.Itoa(int(p._suffix)))
//	}
//	_ = os.Rename(p.LogFilePath, p.LogFilePath+"."+strconv.Itoa(int(p._suffix)))
//	p.initFile()
//}

func (p *LogParams) initFile() {
	var err error
	p.logfile, err = os.OpenFile(p.LogFilePath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		logger.Err(err).Str("init_file", "failed").Msgf("%#v", p.logfile)
		panic("create log file failed")
	}
	fmt.Println("success")
	logger.Info().Str("init_file", "success").Msgf("%#v", p.logfile)
}

func fileSize(file string) int64 {
	f, e := os.Stat(file)
	if e != nil {
		return 0
	}
	return f.Size()
}
//
//func isExist(path string) bool {
//	_, err := os.Stat(path)
//	return err == nil || os.IsExist(err)
//}

func monitor(params *LogParams) {
	t := time.NewTicker(time.Second * 1)
	go func() {
		for {
			select {
			case <- t.C:
				logger.Info().Msg("check file size")
				size := fileSize(params.LogFilePath)
				logger.Info().Str("size", fmt.Sprintf("%d", size)).Msg("check file size")
				if size > params.LogFileSize {
					logger.Info().Msg("rename log file")
					rename2File(params)
					params.output()
				}
			}
		}
	}()
}


func rename2File(params *LogParams) {
	now := time.Now()
	if params.LogTimeFormat == "" {
		params.LogTimeFormat = "2006-01-02 15:04:05"
	}
	newLogFileName := fmt.Sprintf("%s.%s", params.LogFilePath, now.Format(params.LogTimeFormat))
	_ = os.Rename(params.LogFilePath, newLogFileName)
}