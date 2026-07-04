package logger

import (
	"fmt"
	"log/slog"
	"os"
)

func New(level, format string) (*slog.Logger, error) {
	// file, err := os.OpenFile("./pkg/logger/taskflow.log", os.O_CREATE|os.O_APPEND|os.O_APPEND, 0755)
	// if err != nil {
	// 	return nil, fmt.Errorf("open log file: %w", err)
	// }
	if format == "Json" {
		switch level {
		case "Debug":
			logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			})).With(slog.String("service", "taskflow"))
			return logger, nil
		case "Info":
			logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			})).With(slog.String("service", "taskflow"))
			return logger, nil
		case "Warn":
			logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelWarn,
			})).With(slog.String("service", "taskflow"))
			return logger, nil
		case "Error":
			logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelError,
			})).With(slog.String("service", "taskflow"))
			return logger, nil
		default:
			return nil, fmt.Errorf("invalid level")
		}
	} else {
		switch level {
		case "Debug":
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			})).With(slog.String("service", "taskflow"))
			return logger, nil
		case "Info":
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			})).With(slog.String("service", "taskflow"))
			return logger, nil
		case "Warn":
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelWarn,
			})).With(slog.String("service", "taskflow"))
			return logger, nil
		case "Error":
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelError,
			})).With(slog.String("service", "taskflow"))
			return logger, nil
		default:
			return nil, fmt.Errorf("invalid level")
		}
	}
}
