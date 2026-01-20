package logging

import (
	"bytes"
	"testing"
)

func TestInitLogger_SetsSingleton(t *testing.T) {
	defer ResetLogger()

	logger := InitLogger("info", false, nil)

	if logger != Logger() {
		t.Error("Logger() should return same instance as InitLogger returned")
	}
}

func TestLogger_PanicsBeforeInit(t *testing.T) {
	defer ResetLogger()

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when calling Logger() before InitLogger()")
		}
	}()

	Logger()
}

func TestInitLogger_PanicsOnDoubleInit(t *testing.T) {
	defer ResetLogger()

	InitLogger("info", false, nil)

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic on double InitLogger call")
		}
	}()

	InitLogger("info", false, nil)
}

func TestResetLogger_AllowsReinit(t *testing.T) {
	defer ResetLogger()

	InitLogger("info", false, nil)
	ResetLogger()

	// Should not panic after reset
	InitLogger("debug", false, nil)
	if Logger() == nil {
		t.Error("Logger() should return valid logger after reset and reinit")
	}
}

func TestInitLogger_WritesToBuffer(t *testing.T) {
	defer ResetLogger()

	var buf bytes.Buffer
	InitLogger("info", false, &buf)

	Logger().Info("test message")

	output := buf.String()
	if output == "" {
		t.Error("Expected log output in buffer")
	}
	if !bytes.Contains(buf.Bytes(), []byte("test message")) {
		t.Errorf("Expected 'test message' in output, got: %s", output)
	}
}
