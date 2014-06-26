package macaco

import (
	"bytes"
	"strings"
	"time"
)

type testStderrWriter struct {
	test   *Test
	buf    bytes.Buffer
	stderr bytes.Buffer
}

func (t *testStderrWriter) Write(b []byte) (int, error) {
	t.buf.Write(b)
	t.stderr.Write(b)
	if bytes.IndexByte(t.buf.Bytes(), '\n') > 0 {
		t.test.Errors = append(t.test.Errors, &TestError{
			Message:   strings.TrimSpace(t.buf.String()),
			Timestamp: time.Now(),
		})
		t.buf.Reset()
	}
	return len(b), nil
}

func (t *testStderrWriter) String() string {
	return t.stderr.String()
}

type TestError struct {
	Message   string
	Timestamp time.Time
}

type Test struct {
	Name     string
	Started  time.Time
	Finished time.Time
	Errors   []*TestError
	Stdout   string
	Stderr   string
}

func (t *Test) Elapsed() time.Duration {
	return t.Finished.Sub(t.Started)
}

func (t *Test) Passed() bool {
	return len(t.Errors) == 0
}

func (t *Test) stderrWriter() *testStderrWriter {
	return &testStderrWriter{
		test: t,
	}
}
