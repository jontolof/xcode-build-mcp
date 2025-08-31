package common

type Logger interface {
	Printf(format string, v ...interface{})
	Println(v ...interface{})
}
