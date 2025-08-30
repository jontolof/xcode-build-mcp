package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
)

type Transport interface {
	ReadRequest() (*Request, error)
	WriteResponse(*Response) error
	Close() error
}

type StdioTransport struct {
	reader *bufio.Reader
	writer io.Writer
	logger *log.Logger
}

func NewStdioTransport(logger *log.Logger) (*StdioTransport, error) {
	return &StdioTransport{
		reader: bufio.NewReader(os.Stdin),
		writer: os.Stdout,
		logger: logger,
	}, nil
}

func (t *StdioTransport) ReadRequest() (*Request, error) {
	line, err := t.reader.ReadBytes('\n')
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("connection closed")
		}
		return nil, fmt.Errorf("failed to read line: %w", err)
	}

	if t.logger != nil {
		t.logger.Printf("Received: %s", string(line))
	}

	var req Request
	if err := json.Unmarshal(line, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request: %w", err)
	}

	return &req, nil
}

func (t *StdioTransport) WriteResponse(resp *Response) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	if t.logger != nil {
		t.logger.Printf("Sending: %s", string(data))
	}

	if _, err := t.writer.Write(data); err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}

	if _, err := t.writer.Write([]byte("\n")); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	return nil
}

func (t *StdioTransport) Close() error {
	return nil
}