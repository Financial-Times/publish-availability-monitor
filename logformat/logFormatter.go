package logformat

import (
	"fmt"
	"runtime"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	timeFmt = "2006-01-02 15:04:05.000"
	txIdKey = "transaction_id"
)

// Our default SLF4J format is: "%-5p [%d{ISO8601, GMT}] %c: %X{transaction_id} %m [%thread]%n%xEx"
type SLF4JFormatter struct {
}

func (f *SLF4JFormatter) Format(entry *log.Entry) ([]byte, error) {
	level := strings.ToUpper(entry.Level.String())
	// except for warn, which by default becomes WARNING
	if entry.Level == log.WarnLevel {
		level = "WARN"
	}

	timestamp := strings.Replace(entry.Time.UTC().Format(timeFmt), ".", ",", 1)
	codeLocation := f.findCodeLocation()
	tx := f.findTransactionId(entry.Data)
	msg := entry.Message
	for k, v := range entry.Data {
		if k == txIdKey {
			continue
		}

		msg = fmt.Sprintf("%s %s=%v", msg, k, v)
	}
	full := fmt.Sprintf("%-5s [%s] %s %s %s\n", level, timestamp, codeLocation, tx, msg)

	return []byte(full), nil
}

func (f *SLF4JFormatter) findCodeLocation() string {
	// start at 2 because we know 0 and 1 are within this file
	for i := 2; i < 10; i++ {
		_, file, lineNum, ok := runtime.Caller(i)
		if ok {
			return fmt.Sprintf("%s:%v:", file[strings.LastIndex(file, "/")+1:], lineNum)
		}
	}

	return "unknown"
}

func (f *SLF4JFormatter) findTransactionId(data map[string]interface{}) string {
	var tx string
	if v, found := data[txIdKey]; found {
		tx = v.(string)
		if tx != "" && !strings.HasPrefix(tx, txIdKey) {
			tx = txIdKey + "=" + tx
		}
	}

	return tx
}
