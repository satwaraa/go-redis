package protocol

import "fmt"

// RESP (REdis Serialization Protocol) formatters

// FormatOK returns a RESP simple string "+OK\r\n"
func FormatOK() string {
	return "+OK\r\n"
}

// FormatPong returns a RESP simple string "+PONG\r\n"
func FormatPong() string {
	return "+PONG\r\n"
}

// FormatError returns a RESP error "-ERR <msg>\r\n"
func FormatError(msg string) string {
	return fmt.Sprintf("-ERR %s\r\n", msg)
}

// FormatBulkString returns a RESP bulk string "$<len>\r\n<data>\r\n"
func FormatBulkString(val string) string {
	return fmt.Sprintf("$%d\r\n%s\r\n", len(val), val)
}

// FormatNull returns a RESP null bulk string "$-1\r\n"
func FormatNull() string {
	return "$-1\r\n"
}

// FormatInteger returns a RESP integer ":<n>\r\n"
func FormatInteger(n int64) string {
	return fmt.Sprintf(":%d\r\n", n)
}
