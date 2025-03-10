package sl

import "log/slog"

// custom helper function for putting erros inside logs
func Err(err error) slog.Attr {
	return slog.Attr{
		Key:   "error",
		Value: slog.StringValue(err.Error()),
	}
}
