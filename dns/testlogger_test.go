package dns

import (
	"sync"
	"sync/atomic"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
)

type testLogger struct {
	core   atomic.Value // type: zapcore.Core
	coreMu sync.RWMutex
}

func newTestLogger(t *testing.T) *zap.Logger {
	l := &testLogger{}
	core := zaptest.NewLogger(t).Core()
	l.core.Store(&core)
	t.Cleanup(l.Cleanup)
	return zap.New(l)
}

func (l *testLogger) Cleanup() {
	core := zapcore.NewNopCore()
	l.coreMu.Lock()
	defer l.coreMu.Unlock()
	l.core.Store(&core)
}

func (l *testLogger) getCore() zapcore.Core { //nolint:ireturn // Internal core type changes after test cleans up.
	return *l.core.Load().(*zapcore.Core)
}

func (l *testLogger) Enabled(level zapcore.Level) bool  { return l.getCore().Enabled(level) }
func (l *testLogger) With([]zapcore.Field) zapcore.Core { return l } //nolint:ireturn // Implements zapcore.Core
func (l *testLogger) Check(entry zapcore.Entry, checkedEntry *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if l.Enabled(entry.Level) {
		return checkedEntry.AddCore(entry, l)
	}
	return checkedEntry
}

func (l *testLogger) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	l.coreMu.RLock()
	defer l.coreMu.RUnlock()
	return l.getCore().Write(entry, fields)
}
func (l *testLogger) Sync() error { return l.getCore().Sync() }
