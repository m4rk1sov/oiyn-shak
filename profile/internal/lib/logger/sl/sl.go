package sl

import (
	"log/slog"
)

// Err - custom helper function for putting errors inside logs
func Err(err error) slog.Attr {
	return slog.Attr{
		Key:   "error",
		Value: slog.StringValue(err.Error()),
	}
}
