package redisstore

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	addr     string
	password string
	db       int
	useTLS   bool
	timeout  time.Duration
}

func New(rawURL string, useTLS bool) (*Client, error) {
	if rawURL == "" {
		return nil, errors.New("redis url is required")
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	addr := u.Host
	if !strings.Contains(addr, ":") {
		addr += ":6379"
	}
	password, _ := u.User.Password()
	db := 0
	if path := strings.TrimPrefix(u.Path, "/"); path != "" {
		db, _ = strconv.Atoi(path)
	}
	return &Client{addr: addr, password: password, db: db, useTLS: useTLS || u.Scheme == "rediss", timeout: 2 * time.Second}, nil
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.command(ctx, "PING")
	return err
}

func (c *Client) IncrExpire(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	result, err := c.command(ctx, "INCR", key)
	if err != nil {
		return 0, err
	}
	count, ok := result.(int64)
	if !ok {
		return 0, errors.New("unexpected redis INCR response")
	}
	_, _ = c.command(ctx, "EXPIRE", key, strconv.Itoa(maxInt(1, int(ttl.Seconds()))))
	return count, nil
}

func (c *Client) SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	result, err := c.command(ctx, "SET", key, value, "NX", "EX", strconv.Itoa(maxInt(1, int(ttl.Seconds()))))
	if err != nil {
		return false, err
	}
	if result == nil {
		return false, nil
	}
	return true, nil
}

func (c *Client) Del(ctx context.Context, key string) error {
	_, err := c.command(ctx, "DEL", key)
	return err
}

func (c *Client) command(ctx context.Context, args ...string) (interface{}, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", c.addr)
	if err != nil {
		return nil, err
	}
	if c.useTLS {
		conn = tls.Client(conn, &tls.Config{MinVersion: tls.VersionTLS12})
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(c.timeout))
	reader := bufio.NewReader(conn)
	if c.password != "" {
		if _, err := writeCommand(conn, "AUTH", c.password); err != nil {
			return nil, err
		}
		if _, err := readRESP(reader); err != nil {
			return nil, err
		}
	}
	if c.db > 0 {
		if _, err := writeCommand(conn, "SELECT", strconv.Itoa(c.db)); err != nil {
			return nil, err
		}
		if _, err := readRESP(reader); err != nil {
			return nil, err
		}
	}
	if _, err := writeCommand(conn, args...); err != nil {
		return nil, err
	}
	return readRESP(reader)
}

func writeCommand(conn net.Conn, args ...string) (int, error) {
	var b strings.Builder
	b.WriteString("*")
	b.WriteString(strconv.Itoa(len(args)))
	b.WriteString("\r\n")
	for _, arg := range args {
		b.WriteString("$")
		b.WriteString(strconv.Itoa(len(arg)))
		b.WriteString("\r\n")
		b.WriteString(arg)
		b.WriteString("\r\n")
	}
	return conn.Write([]byte(b.String()))
}

func readRESP(r *bufio.Reader) (interface{}, error) {
	prefix, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
	switch prefix {
	case '+':
		return line, nil
	case '-':
		return nil, errors.New(line)
	case ':':
		return strconv.ParseInt(line, 10, 64)
	case '$':
		n, err := strconv.Atoi(line)
		if err != nil {
			return nil, err
		}
		if n < 0 {
			return nil, nil
		}
		buf := make([]byte, n+2)
		if _, err := r.Read(buf); err != nil {
			return nil, err
		}
		return string(buf[:n]), nil
	default:
		return nil, fmt.Errorf("unsupported redis response %q", prefix)
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
