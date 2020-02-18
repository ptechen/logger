package logger

import (
	_ "code.cloudfoundry.org/go-diodes"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/diode"
	"github.com/rs/zerolog/log"
	"os"
	"strconv"
	"sync"
	"time"
)
var (
	dailyRolling bool = true
	checkMustRenameTime int64
	maxFileCount int32
	maxFileSize int64 = 1024
	logParams *LogParams
	once sync.Once
	onceLog sync.Once
	logger zerolog.Logger
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
	Level           int8   `json:"level"`
	Color           bool   `json:"color"`
	FilePath        string `json:"file_path"`
	IsConsole       bool   `json:"is_console"`
	TimeFieldFormat string `json:"time_field_format"`
	Caller          bool   `json:"caller"`
	ServerName      string `json:"server_name"`
	Default         bool   `json:"default"`
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
			FilePath:           "log.log",
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
	if p.FilePath == "" && p.IsConsole == false {
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

func (p *LogParams) SetWriteFilePath(filePath string) *LogParams {
	p.FilePath = filePath
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
	if p.FilePath != "" && p.IsConsole == false {
		p.initFile()
		w := diode.NewWriter(p.logfile, 10000, 10*time.Millisecond, func(missed int) {
			logger.Warn().Msgf("Logger Dropped %d messages", missed)
		})
		logger = logger.Output(w).Level(zerolog.Level(p.Level))
	}

	if p.IsConsole {
		if p.Color {
			logger = logger.Output(zerolog.ConsoleWriter{Out: os.Stdout}).Level(zerolog.Level(p.Level))
		} else {
			logger = logger.Output(os.Stdout).Level(zerolog.Level(p.Level))
		}
	}
}

func (p *LogParams) caller() {
	if p.Caller {
		logger = logger.With().Caller().Logger()
	}
}

func (p *LogParams) InitLog() zerolog.Logger {
	onceLog.Do(func() {
		logger = log.Logger
		p.setFileName()
		p.caller()
		p.output()
	})
	t := time.NewTicker(time.Second * 3)
	go func() {
		for {
			select {
			case <- t.C:
				must := p.isMustRename()
				if must {
					p.rename()
				}
			}
		}
	}()
	return logger
}


func (p *LogParams) isMustRename() bool {
	now := time.Now()
	//3秒检查一次，不然太频繁
	if checkMustRenameTime != 0 && now.Unix()-checkMustRenameTime < 3 {
		return false
	}
	checkMustRenameTime = now.Unix()
	//if dailyRolling {
	//	t, _ := time.Parse(DATEFORMAT, now.Format(DATEFORMAT))
	//	if t.After(*p._date) {
	//		return true
	//	}
	//} else {
	//	if maxFileCount > 1 {
	//		if fileSize(p.FilePath) >= maxFileSize {
	//			return true
	//		}
	//	}
	//}
	if fileSize(p.FilePath) >= maxFileSize {
		return true
	}
	return false
}

func (p *LogParams) rename() {
	if dailyRolling {
		t, _ := time.Parse(DATEFORMAT, time.Now().Format(DATEFORMAT))
		p._date = &t
		fn := p.FilePath + p._date.Format(DATEFORMAT) + fmt.Sprintf("-%d", maxFileCount)
		if !isExist(fn) {
			if p.logfile != nil {
				_ = p.logfile.Close()
			}
			err := os.Rename(p.FilePath, fn)
			if err != nil {
				logger.Err(err).Msg("rename log file failed")
			}

			p.initFile()
			maxFileCount += 1
		}
	} else {
		p.coverNextOne()
	}
}

func (p *LogParams) nextSuffix() int {
	return int(p._suffix%int(maxFileCount) + 1)
}

func (p *LogParams) coverNextOne() {
	p._suffix = p.nextSuffix()
	if p.logfile != nil {
		_ = p.logfile.Close()
	}
	if isExist(p.FilePath + "." + strconv.Itoa(int(p._suffix))) {
		_ = os.Remove(p.FilePath + "." + strconv.Itoa(int(p._suffix)))
	}
	_ = os.Rename(p.FilePath, p.FilePath+"."+strconv.Itoa(int(p._suffix)))
	p.initFile()
}

func (p *LogParams) initFile() {
	var err error
	p.logfile, err = os.OpenFile(p.FilePath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		logger.Err(err).Str("init_file", "failed").Msgf("%#v",p.logfile)
		panic("create log file failed")
	}
	fmt.Println("success")
	logger.Info().Str("init_file", "success").Msgf("%#v",p.logfile)
}

func fileSize(file string) int64 {
	f, e := os.Stat(file)
	if e != nil {
		return 0
	}
	return f.Size()
}

func isExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}