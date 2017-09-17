package main

import (
	"fmt"
	"log"
	"os"
	"time"
)

const LOG_OUTPUT_BUFFER = 1024

const (
	LevelDebug = iota
	LevelInfo
	LevelNotice
	LevelWarn
	LevelError
)

const (
	AllEvents = iota
	AuditedEvents
	BlockedEvents
)

type logMesg struct {
	Level int
	Mesg  string
}

type LoggerHandler interface {
	Setup(config map[string]interface{}, isTimestamped bool) error
	Write(mesg *logMesg)
}

type GoDNSLogger struct {
	level         int
	mesgs         chan *logMesg
	isTimestamped bool
	outputs       map[string]LoggerHandler
}

func NewLogger() *GoDNSLogger {
	logger := &GoDNSLogger{
		mesgs:   make(chan *logMesg, LOG_OUTPUT_BUFFER),
		outputs: make(map[string]LoggerHandler),
	}
	go logger.Run()
	return logger
}

func (l *GoDNSLogger) SetLogger(handlerType string, config map[string]interface{}, isTimestamped bool) {
	var handler LoggerHandler
	switch handlerType {
	case "console":
		handler = NewConsoleHandler()
	case "file":
		handler = NewFileHandler()
	default:
		panic("Unknown log handler.")
	}

	handler.Setup(config, isTimestamped)
	l.outputs[handlerType] = handler
}

func (l *GoDNSLogger) SetLevel(level int) {
	l.level = level
}

func (l *GoDNSLogger) Run() {
	for {
		select {
		case mesg := <-l.mesgs:
			for _, handler := range l.outputs {
				handler.Write(mesg)
			}
		}
	}
}

func (l *GoDNSLogger) writeMesg(mesg string, level int) {
	lm := &logMesg{
		Level: level,
		Mesg:  mesg,
	}

	l.mesgs <- lm
}

func (l *GoDNSLogger) Debug(format string, v ...interface{}) {
	if l.level > LevelDebug {
		return
	}
	mesg := fmt.Sprintf("[DEBUG] "+format, v...)
	l.writeMesg(mesg, LevelDebug)
}

func (l *GoDNSLogger) Info(format string, v ...interface{}) {
	if l.level > LevelInfo {
		return
	}
	mesg := fmt.Sprintf("[INFO] "+format, v...)
	l.writeMesg(mesg, LevelInfo)
}

func (l *GoDNSLogger) Notice(format string, v ...interface{}) {
	if l.level > LevelNotice {
		return
	}
	mesg := fmt.Sprintf("[NOTICE] "+format, v...)
	l.writeMesg(mesg, LevelNotice)
}

func (l *GoDNSLogger) Warn(format string, v ...interface{}) {
	if l.level > LevelWarn {
		return
	}
	mesg := fmt.Sprintf("[WARN] "+format, v...)
	l.writeMesg(mesg, LevelWarn)
}

func (l *GoDNSLogger) Error(format string, v ...interface{}) {
	if l.level > LevelError {
		return
	}
	mesg := fmt.Sprintf("[ERROR] "+format, v...)
	l.writeMesg(mesg, LevelError)
}

/* We want exactly the https://github.com/golang/go/blob/release-branch.go1.8/src/log/log.go#L76
   don't we?
*/
func itoa(buf *[]byte, i int, wid int) {
	// Assemble decimal in reverse order.
	var b [20]byte
	bp := len(b) - 1
	for i >= 10 || wid > 1 {
		wid--
		q := i / 10
		b[bp] = byte('0' + i - q*10)
		bp--
		i = q
	}
	// i < 10
	b[bp] = byte('0' + i)
	*buf = append(*buf, b[bp:]...)
}

func formatTimestampAsInGoLog(t time.Time) []byte {
	var buf []byte
	t = t.UTC()
	year, month, day := t.Date()
	itoa(&buf, year, 4)
	buf = append(buf, '/')
	itoa(&buf, int(month), 2)
	buf = append(buf, '/')
	itoa(&buf, day, 2)
	buf = append(buf, ' ')
	hour, min, sec := t.Clock()
	itoa(&buf, hour, 2)
	buf = append(buf, ':')
	itoa(&buf, min, 2)
	buf = append(buf, ':')
	itoa(&buf, sec, 2)
	buf = append(buf, '.')
	itoa(&buf, t.Nanosecond()/1e3, 6)
	return buf
}

func (l *GoDNSLogger) Audited(clientAddress string, query string) {
	if l.level > AuditedEvents {
		return
	}
	mesg := fmt.Sprintf("{\"timestamp\":\"%s\",\"client_ip\":\"%s\",\"domain\":\"%s\",\"action\":\"audit\"}", formatTimestampAsInGoLog(time.Now()), clientAddress, query)
	l.writeMesg(mesg, AuditedEvents)
}

func (l *GoDNSLogger) Blocked(clientAddress string, query string) {
	if l.level > BlockedEvents{
		return
	}
	mesg := fmt.Sprintf("{\"timestamp\":\"%s\",\"client_ip\":\"%s\",\"domain\":\"%s\",\"action\":\"block\"}", formatTimestampAsInGoLog(time.Now()), clientAddress, query)
	l.writeMesg(mesg, BlockedEvents)
}

type ConsoleHandler struct {
	level  int
	logger *log.Logger
}

func NewConsoleHandler() LoggerHandler {
	return new(ConsoleHandler)
}

func (h *ConsoleHandler) Setup(config map[string]interface{}, isTimestamped bool) error {
	if _level, ok := config["level"]; ok {
		level := _level.(int)
		h.level = level
	}
	if(isTimestamped) {
		h.logger = log.New(os.Stdout, "", log.Ldate | log.Ltime)
	} else {
		h.logger = log.New(os.Stdout, "", 0)
	}
	return nil
}

func (h *ConsoleHandler) Write(lm *logMesg) {
	if h.level <= lm.Level {
		h.logger.Println(lm.Mesg)
	}
}

type FileHandler struct {
	level  int
	file   string
	logger *log.Logger
}

func NewFileHandler() LoggerHandler {
	return new(FileHandler)
}

func (h *FileHandler) Setup(config map[string]interface{}, isTimestamped bool) error {
	if level, ok := config["level"]; ok {
		h.level = level.(int)
	}

	if file, ok := config["file"]; ok {
		h.file = file.(string)
		output, err := os.Create(h.file)
		if err != nil {
			return err
		}
		if(isTimestamped) {
			h.logger = log.New(output, "", log.Ldate|log.Ltime|log.Lmicroseconds)
		} else {
			h.logger = log.New(output, "", 0)
		}
	}

	return nil
}

func (h *FileHandler) Write(lm *logMesg) {
	if h.logger == nil {
		return
	}

	if h.level <= lm.Level {
		h.logger.Println(lm.Mesg)
	}
}
