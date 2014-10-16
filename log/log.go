// Wrapper for the built in log package using syslog severity formats
package log

import (
  golog "log"
  "strings"
  "fmt"
)

func SetPrefix(prefix string) {
  golog.SetPrefix(prefix)
}

func Info(v ...interface{}) {
  log("info", v)
}

func Notice(v ...interface{}) {
  log("notice", v)
}

func Warning(v ...interface{}) {
  log("warning", v)
}

func Error(v ...interface{}) {
  log("error", v)
}

func Critical(v ...interface{}) {
  alert("critical", v)
}

func Panic(v ...interface{}) {
  panic("panic", v)
}

func panic(level string, v []interface{}) {
  golog.Fatal(combineParams(levelfmt(level), v)...)
}

func alert(level string, v []interface{}) {
  golog.Panic(combineParams(levelfmt(level), v)...)
}

func log(level string, v []interface{}) {
  golog.Print(combineParams(levelfmt(level), v)...)
}

func combineParams(level string, parts []interface{}) []interface{} {
  preparts := make([]interface{}, 1)
  preparts[0] = level

  parts[0] = fmt.Sprintf("[%s] ", parts[0])
  return append(preparts, parts...)
}

func levelfmt(level string) string {
  return "[" + strings.ToUpper(level) + "] "
}
