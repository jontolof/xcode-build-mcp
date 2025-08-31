package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"log"
	"strings"
	"testing"
)

func TestStdioTransport_ReadRequest(t *testing.T) {
	message := `{"jsonrpc":"2.0","id":1,"method":"test"}`
	reader := strings.NewReader(message + "\n")
	writer := &bytes.Buffer{}
	logger := log.New(bytes.NewBuffer(nil), "", 0)

	transport := &StdioTransport{
		reader: bufio.NewReader(reader),
		writer: writer,
		logger: logger,
	}

	req, err := transport.ReadRequest()
	if err != nil {
		t.Fatalf("Failed to read request: %v", err)
	}

	if req.Method != "test" {
		t.Errorf("Method mismatch: got %v, want test", req.Method)
	}
}

func TestStdioTransport_WriteResponse(t *testing.T) {
	reader := strings.NewReader("")
	writer := &bytes.Buffer{}
	logger := log.New(bytes.NewBuffer(nil), "", 0)

	transport := &StdioTransport{
		reader: bufio.NewReader(reader),
		writer: writer,
		logger: logger,
	}

	response := &Response{
		JSONRPC: "2.0",
		ID:      1,
		Result:  "ok",
	}

	if err := transport.WriteResponse(response); err != nil {
		t.Fatalf("Failed to write response: %v", err)
	}

	output := writer.String()

	// Check that output ends with newline
	if !strings.HasSuffix(output, "\n") {
		t.Error("Output should end with newline")
	}

	// Check that the JSON is valid
	var decoded Response
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &decoded); err != nil {
		t.Fatalf("Failed to unmarshal output: %v", err)
	}

	if decoded.Result != "ok" {
		t.Errorf("Result mismatch: got %v, want ok", decoded.Result)
	}
}

func TestStdioTransport_ReadRequest_EOF(t *testing.T) {
	reader := strings.NewReader("")
	writer := &bytes.Buffer{}
	logger := log.New(bytes.NewBuffer(nil), "", 0)

	transport := &StdioTransport{
		reader: bufio.NewReader(reader),
		writer: writer,
		logger: logger,
	}

	_, err := transport.ReadRequest()
	if err == nil {
		t.Fatal("Should return error on EOF")
	}
	if !strings.Contains(err.Error(), "connection closed") {
		t.Errorf("Should return connection closed error, got: %v", err)
	}
}

func TestStdioTransport_ReadRequest_InvalidJSON(t *testing.T) {
	reader := strings.NewReader("invalid json\n")
	writer := &bytes.Buffer{}
	logger := log.New(bytes.NewBuffer(nil), "", 0)

	transport := &StdioTransport{
		reader: bufio.NewReader(reader),
		writer: writer,
		logger: logger,
	}

	_, err := transport.ReadRequest()
	if err == nil {
		t.Fatal("Should return error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "unmarshal") {
		t.Errorf("Should return unmarshal error, got: %v", err)
	}
}

func TestStdioTransport_Close(t *testing.T) {
	reader := strings.NewReader("")
	writer := &bytes.Buffer{}
	logger := log.New(bytes.NewBuffer(nil), "", 0)

	transport := &StdioTransport{
		reader: bufio.NewReader(reader),
		writer: writer,
		logger: logger,
	}

	// Close should not return an error for stdio transport
	if err := transport.Close(); err != nil {
		t.Errorf("Close should not return an error: %v", err)
	}
}

func TestNewStdioTransport(t *testing.T) {
	logger := log.New(bytes.NewBuffer(nil), "", 0)

	transport, err := NewStdioTransport(logger)
	if err != nil {
		t.Fatalf("Failed to create StdioTransport: %v", err)
	}

	if transport == nil {
		t.Fatal("Transport should not be nil")
	}
	if transport.reader == nil {
		t.Fatal("Reader should not be nil")
	}
	if transport.writer == nil {
		t.Fatal("Writer should not be nil")
	}
	if transport.logger == nil {
		t.Fatal("Logger should not be nil")
	}
}

func TestStdioTransport_WriteResponse_MarshalError(t *testing.T) {
	reader := strings.NewReader("")
	writer := &bytes.Buffer{}
	logger := log.New(bytes.NewBuffer(nil), "", 0)

	transport := &StdioTransport{
		reader: bufio.NewReader(reader),
		writer: writer,
		logger: logger,
	}

	// Create a response with an unmarshalable field
	response := &Response{
		JSONRPC: "2.0",
		ID:      1,
		Result:  make(chan int), // channels cannot be marshaled to JSON
	}

	err := transport.WriteResponse(response)
	if err == nil {
		t.Fatal("Should return error when marshaling fails")
	}
	if !strings.Contains(err.Error(), "marshal") {
		t.Errorf("Should return marshal error, got: %v", err)
	}
}

// MockWriter that returns an error on Write
type errorWriter struct{}

func (w *errorWriter) Write(p []byte) (n int, err error) {
	return 0, io.ErrClosedPipe
}

func TestStdioTransport_WriteResponse_WriteError(t *testing.T) {
	reader := strings.NewReader("")
	writer := &errorWriter{}
	logger := log.New(bytes.NewBuffer(nil), "", 0)

	transport := &StdioTransport{
		reader: bufio.NewReader(reader),
		writer: writer,
		logger: logger,
	}

	response := &Response{
		JSONRPC: "2.0",
		ID:      1,
		Result:  "ok",
	}

	err := transport.WriteResponse(response)
	if err == nil {
		t.Fatal("Should return error when write fails")
	}
	if !strings.Contains(err.Error(), "write") {
		t.Errorf("Should return write error, got: %v", err)
	}
}
