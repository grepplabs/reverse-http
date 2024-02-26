package agent

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/grepplabs/reverse-http/pkg/logger"
	"github.com/quic-go/quic-go"
)

const MaxAuthMessageLength = 1024 * 1024

type authFlow struct {
	timeout time.Duration
	logger  *logger.Logger
}

func (r *authFlow) authenticate(ctx context.Context, conn quic.Connection, token string) error {
	deadline := time.Now().Add(r.timeout)
	ctx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		return fmt.Errorf("auth open failed: %v", err)
	}
	defer stream.Close()
	_ = stream.SetDeadline(deadline)

	err = writeString(stream, token)
	if err != nil {
		return fmt.Errorf("auth write failed: %v", err)
	}
	_, err = readString(stream)
	if err != nil {
		return fmt.Errorf("auth read failed: %v", err)
	}
	return nil
}

func (r *authFlow) verify(ctx context.Context, conn quic.Connection, verifier func(token string) (*Attributes, error)) (*Attributes, error) {
	deadline := time.Now().Add(r.timeout)
	ctx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	stream, err := conn.AcceptStream(ctx)
	if err != nil {
		return nil, fmt.Errorf("verify accept failed: %v", err)
	}
	defer stream.Close()

	token, err := readString(stream)
	if err != nil {
		return nil, fmt.Errorf("verify read failed: %v", err)
	}
	attrs, err := verifier(token)
	if err != nil {
		return nil, err
	}
	err = writeString(stream, "authenticated")
	if err != nil {
		return nil, fmt.Errorf("verify write failed: %v", err)
	}
	return attrs, nil
}

func writeString(stream quic.Stream, message string) error {
	bs := []byte(message)
	length := uint32(len(bs))
	if length > MaxAuthMessageLength {
		return fmt.Errorf("write message too long: %d", length)
	}
	err := binary.Write(stream, binary.BigEndian, length)
	if err != nil {
		return err
	}
	_, err = io.Copy(stream, bytes.NewReader(bs))
	if err != nil {
		return err
	}
	return err
}

func readString(stream quic.Stream) (string, error) {
	var length uint32
	err := binary.Read(stream, binary.BigEndian, &length)
	if err != nil {
		return "", err
	}
	if length > MaxAuthMessageLength {
		return "", fmt.Errorf("read message too long: %d", length)
	}
	buf := make([]byte, length)
	_, err = io.ReadFull(stream, buf)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}
