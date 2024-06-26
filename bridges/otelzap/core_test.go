// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelzap

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/log/logtest"
)

var (
	testMessage = "log message"
	loggerName  = "name"
)

func TestCore(t *testing.T) {
	rec := logtest.NewRecorder()
	zc := NewCore(loggerName, WithLoggerProvider(rec))
	logger := zap.New(zc)

	logger.Info(testMessage)

	// TODO (#5580): Not sure why index 1 is populated with results and not 0.
	got := rec.Result()[0].Records[0]
	assert.Equal(t, testMessage, got.Body().AsString())
	assert.Equal(t, log.SeverityInfo, got.Severity())
}

func TestCoreEnabled(t *testing.T) {
	enabledFunc := func(c context.Context, r log.Record) bool {
		return r.Severity() >= log.SeverityInfo
	}

	r := logtest.NewRecorder(logtest.WithEnabledFunc(enabledFunc))
	logger := zap.New(NewCore(loggerName, WithLoggerProvider(r)))

	logger.Debug(testMessage)
	assert.Empty(t, r.Result()[0].Records)

	if ce := logger.Check(zap.DebugLevel, testMessage); ce != nil {
		ce.Write()
	}
	assert.Empty(t, r.Result()[0].Records)

	if ce := logger.Check(zap.InfoLevel, testMessage); ce != nil {
		ce.Write()
	}
	require.Len(t, r.Result()[0].Records, 1)
	got := r.Result()[0].Records[0]
	assert.Equal(t, testMessage, got.Body().AsString())
	assert.Equal(t, log.SeverityInfo, got.Severity())
}

func TestNewCoreConfiguration(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		r := logtest.NewRecorder()
		prev := global.GetLoggerProvider()
		defer global.SetLoggerProvider(prev)
		global.SetLoggerProvider(r)

		var h *Core
		require.NotPanics(t, func() { h = NewCore(loggerName) })
		require.NotNil(t, h.logger)
		require.Len(t, r.Result(), 1)

		want := &logtest.ScopeRecords{Name: loggerName}
		got := r.Result()[0]
		assert.Equal(t, want, got)
	})

	t.Run("Options", func(t *testing.T) {
		r := logtest.NewRecorder()
		var h *Core
		require.NotPanics(t, func() {
			h = NewCore(
				loggerName,
				WithLoggerProvider(r),
				WithVersion("1.0.0"),
				WithSchemaURL("url"),
			)
		})
		require.NotNil(t, h.logger)
		require.Len(t, r.Result(), 1)

		want := &logtest.ScopeRecords{Name: loggerName, Version: "1.0.0", SchemaURL: "url"}
		got := r.Result()[0]
		assert.Equal(t, want, got)
	})
}

func TestConvertLevel(t *testing.T) {
	tests := []struct {
		level       zapcore.Level
		expectedSev log.Severity
	}{
		{zapcore.DebugLevel, log.SeverityDebug},
		{zapcore.InfoLevel, log.SeverityInfo},
		{zapcore.WarnLevel, log.SeverityWarn},
		{zapcore.ErrorLevel, log.SeverityError},
		{zapcore.DPanicLevel, log.SeverityFatal1},
		{zapcore.PanicLevel, log.SeverityFatal2},
		{zapcore.FatalLevel, log.SeverityFatal3},
		{zapcore.InvalidLevel, log.SeverityUndefined},
	}

	for _, test := range tests {
		result := convertLevel(test.level)
		if result != test.expectedSev {
			t.Errorf("For level %v, expected %v but got %v", test.level, test.expectedSev, result)
		}
	}
}
