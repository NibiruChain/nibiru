package server

import (
	// Use "log/slog" from the Go std lib because Geth migrated to support
	// slog and deprecated the original go-ethereum/log implementation.
	// For more info on the migration, see https://github.com/ethereum/go-ethereum/pull/28187
	"context"
	"log/slog"

	cmtlog "github.com/cometbft/cometbft/libs/log"
	gethlog "github.com/ethereum/go-ethereum/log"
)

// Ensure LogHandler implements slog.Handler
var _ slog.Handler = (*LogHandler)(nil)

// LogHandler implements slog.Handler, which is needed to construct the
// conventional go-ethereum.Logger, using a CometBFT logger
// ("github.com/cometbft/cometbft/libs/log".Logger).
type LogHandler struct {
	CmtLogger cmtlog.Logger
	attrs     []slog.Attr // Attributes gathered via WithAttrs
	group     string      // Group name gathered via WithGroup (simple implementation)
}

// Enabled decides whether a log record should be processed.
// We let the underlying CometBFT logger handle filtering.
func (h *LogHandler) Enabled(_ context.Context, level slog.Level) bool {
	// You could potentially check the CmtLogger's level here if needed,
	// but returning true is usually sufficient.
	return true
}

// Handle processes the log record and sends it to the CometBFT logger.
func (h *LogHandler) Handle(_ context.Context, r slog.Record) error {
	// 1. Determine the corresponding CometBFT log function
	var logFunc func(msg string, keyvals ...interface{})
	switch r.Level {
	// Check against Geth's custom levels first if they exist
	// This handler covers all defined slog and go-ethereum log levels.
	case gethlog.LevelTrace, slog.LevelDebug: // Handles -8, -4
		logFunc = h.CmtLogger.Debug
	case slog.LevelInfo, slog.LevelWarn: // Handles 0, 4
		logFunc = h.CmtLogger.Info
	case gethlog.LevelCrit, slog.LevelError: // Handles 12, 8
		// Map Geth Crit level along with standard Error
		logFunc = h.CmtLogger.Error
	default: // Handle any other levels (e.g., below Debug)
		logFunc = h.CmtLogger.Debug // Default to Debug or Info as appropriate
	}

	// 2. Collect attributes (key-value pairs)
	// Preallocate assuming 2 slots per attribute plus handler attrs
	keyvals := make([]interface{}, 0, (r.NumAttrs()+len(h.attrs))*2)

	// Add attributes stored in the handler first (from WithAttrs)
	currentGroup := h.group
	for _, attr := range h.attrs {
		key := attr.Key
		if currentGroup != "" {
			key = currentGroup + "." + key // Basic grouping
		}
		keyvals = append(keyvals, key, attr.Value.Any())
	}

	// Add attributes from the specific log record
	r.Attrs(func(attr slog.Attr) bool {
		key := attr.Key
		if currentGroup != "" {
			key = currentGroup + "." + key // Basic grouping
		}
		keyvals = append(keyvals, key, attr.Value.Any())
		return true // Continue iterating
	})

	// 3. Call the CometBFT logger function
	logFunc(r.Message, keyvals...)

	return nil // No error to return in this basic implementation
}

// WithAttrs returns a new LogHandler with the provided attributes added.
func (h *LogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// Create a new handler, cloning existing state and adding new attrs
	newHandler := &LogHandler{
		CmtLogger: h.CmtLogger, // Keep the same underlying logger
		// Important: Clone slices to avoid modifying the original handler's state
		attrs: append(append([]slog.Attr{}, h.attrs...), attrs...),
		group: h.group,
	}
	return newHandler
}

// WithGroup returns a new LogHandler associated with the specified group.
func (h *LogHandler) WithGroup(name string) slog.Handler {
	// Create a new handler, cloning attributes and setting/appending the group
	newHandler := &LogHandler{
		CmtLogger: h.CmtLogger,
		attrs:     append([]slog.Attr{}, h.attrs...), // Clone attributes
		// Basic implementation: Overwrites group. Could concatenate if nesting needed.
		group: name,
	}
	// If nested groups are needed:
	// if h.group != "" {
	//  name = h.group + "." + name
	// }
	// newHandler.group = name
	return newHandler
}
