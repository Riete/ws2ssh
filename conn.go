package ws2ssh

import (
	"errors"
	"io"
	"net"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/gorilla/websocket"
)

type wsConn struct {
	*websocket.Conn
	buff []byte
}

func NewNetConnFromWsConn(conn *websocket.Conn) net.Conn {
	c := wsConn{
		Conn: conn,
	}
	return &c
}

// Read is not threads safe though that's okay since there
// should never be more than one reader
func (c *wsConn) Read(dst []byte) (int, error) {
	ldst := len(dst)
	// use buffer or read new message
	var src []byte
	if len(c.buff) > 0 {
		src = c.buff
		c.buff = nil
	} else {
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			return 0, err
		}
		src = msg
	}
	// copy src->dest
	var n int
	if len(src) > ldst {
		// copy as much as possible of src into dst
		n = copy(dst, src[:ldst])
		// copy remainder into buffer
		r := src[ldst:]
		lr := len(r)
		c.buff = make([]byte, lr)
		copy(c.buff, r)
	} else {
		// copy all src into dst
		n = copy(dst, src)
	}
	// return bytes copied
	return n, nil
}

func (c *wsConn) Write(b []byte) (int, error) {
	if err := c.Conn.WriteMessage(websocket.BinaryMessage, b); err != nil {
		return 0, err
	}
	return len(b), nil
}

func (c *wsConn) SetDeadline(t time.Time) error {
	if err := c.Conn.SetReadDeadline(t); err != nil {
		return err
	}
	return c.Conn.SetWriteDeadline(t)
}

var ErrorInvalidConnection = errors.New("invalid connection")

type sshConn struct {
	dst  io.ReadWriteCloser
	conn ssh.Conn
}

func NewNetConnFromSSHConn(conn ssh.Conn, remote string) (net.Conn, error) {
	if conn == nil {
		return nil, errors.New("the ssh connection is nil")
	}

	dst, reqs, err := conn.OpenChannel("ssh-ch", []byte(remote))
	if err != nil {
		return nil, err
	}
	go ssh.DiscardRequests(reqs)

	return &sshConn{
		dst:  dst,
		conn: conn,
	}, nil
}

func (s *sshConn) Read(b []byte) (n int, err error) {
	if s != nil {
		return s.dst.Read(b)
	}
	return 0, ErrorInvalidConnection
}

func (s *sshConn) Write(b []byte) (n int, err error) {
	if s != nil {
		return s.dst.Write(b)
	}

	return 0, ErrorInvalidConnection
}

func (s *sshConn) Close() error {
	if s != nil {
		return s.dst.Close()
	}
	return nil
}

func (s *sshConn) LocalAddr() net.Addr {
	return s.conn.LocalAddr()
}

func (s *sshConn) RemoteAddr() net.Addr {
	return s.conn.RemoteAddr()
}

func (s *sshConn) SetDeadline(t time.Time) error {
	return nil
}

func (s *sshConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (s *sshConn) SetWriteDeadline(t time.Time) error {
	return nil // no-op
}

func (s *sshConn) Network() string {
	return ""
}

func (s *sshConn) String() string {
	return ""
}
