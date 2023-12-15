package ws2ssh

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"golang.org/x/crypto/ssh"
)

func generateKey() ([]byte, error) {
	r := rand.Reader
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), r)
	if err != nil {
		return nil, err
	}
	b, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal ECDSA private key: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: b}), nil
}

func NewClientConfig(username, password string, privateKey []byte) *ssh.ClientConfig {
	auth := []ssh.AuthMethod{ssh.Password(password)}
	if signer, err := ssh.ParsePrivateKey(privateKey); err == nil {
		auth = append(auth, ssh.PublicKeys(signer))
	}
	return &ssh.ClientConfig{
		User:            username,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}
}

func NewServerConfig(username, password string, publicKey []byte) *ssh.ServerConfig {
	key, _ := generateKey()
	private, _ := ssh.ParsePrivateKey(key)
	conf := &ssh.ServerConfig{
		PasswordCallback: func(conn ssh.ConnMetadata, secret []byte) (*ssh.Permissions, error) {
			if conn.User() == username && string(secret) == password {
				return nil, nil
			}
			return nil, errors.New("password auth failed")
		},
	}
	conf.AddHostKey(private)
	if pubKey, _, _, _, err := ssh.ParseAuthorizedKey(publicKey); err == nil {
		conf.PublicKeyCallback = func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if conn.User() == username && string(pubKey.Marshal()) == string(key.Marshal()) {
				return nil, nil
			}
			return nil, errors.New("public key auth failed")
		}
	}
	return conf
}

func pipe(src io.ReadWriteCloser, dst io.ReadWriteCloser) {
	var wg sync.WaitGroup
	var o sync.Once
	closeReader := func() {
		_ = src.Close()
		_ = dst.Close()
	}

	wg.Add(2)
	go func() {
		_, _ = io.Copy(src, dst)
		o.Do(closeReader)
		wg.Done()
	}()

	go func() {
		_, _ = io.Copy(dst, src)
		o.Do(closeReader)
		wg.Done()
	}()
	wg.Wait()
}

type HandleChannelFunc func(src io.ReadWriteCloser, remote string)

var Direct = func(src io.ReadWriteCloser, remote string) {
	dst, err := net.Dial("tcp", remote)
	if err != nil {
		_ = src.Close()
	} else {
		pipe(src, dst)
	}
}

// Next data transfer by next tunnel
var Next = func(t *SSHTunnel) HandleChannelFunc {
	return func(src io.ReadWriteCloser, remote string) {
		if err := t.HandleIncoming(src, remote); err != nil {
			_ = src.Close()
		}
	}
}

// SSHTunnel
// basically client or server is meaningless in a tunnel, here we define
// incoming data -> tunnel left/right(AsClientSide) -> tunnel -> tunnel right/left(AsServerSide) -> outgoing data
type SSHTunnel struct {
	once          sync.Once
	conn          net.Conn
	sshConn       ssh.Conn
	sshReq        <-chan *ssh.Request
	sshNewChannel <-chan ssh.NewChannel
	asServerSide  bool
}

func (s *SSHTunnel) AsClientSide(config *ssh.ClientConfig, discardReq bool) error {
	var err error
	s.once.Do(func() {
		if config == nil {
			config = NewClientConfig("", "", nil)
		}
		s.sshConn, s.sshNewChannel, s.sshReq, err = ssh.NewClientConn(s.conn, "", config)
		if err == nil && discardReq {
			go ssh.DiscardRequests(s.sshReq)
		}
	})
	return err
}

func (s *SSHTunnel) AsServerSide(config *ssh.ServerConfig, discardReq bool) error {
	var err error
	s.once.Do(func() {
		if config == nil {
			config = NewServerConfig("", "", nil)
		}
		s.sshConn, s.sshNewChannel, s.sshReq, err = ssh.NewServerConn(s.conn, config)
		s.asServerSide = true
		if err == nil && discardReq {
			go ssh.DiscardRequests(s.sshReq)
		}
	})
	return err
}

func (s *SSHTunnel) IsServerSide() bool {
	return s.asServerSide
}

func (s *SSHTunnel) IsClientSide() bool {
	return !s.asServerSide
}

func (s *SSHTunnel) SSHConn() ssh.Conn {
	return s.sshConn
}

func (s *SSHTunnel) SSHNewChannel() <-chan ssh.NewChannel {
	return s.sshNewChannel
}

func (s *SSHTunnel) SSHReq() <-chan *ssh.Request {
	return s.sshReq
}

func (s *SSHTunnel) HandleIncoming(src io.ReadWriteCloser, remote string) error {
	ch, reqs, err := s.sshConn.OpenChannel("ssh-ch", []byte(remote))
	if err == nil {
		go ssh.DiscardRequests(reqs)
		go pipe(src, ch)
	}
	return err
}

func (s *SSHTunnel) HandleOutgoing(hf HandleChannelFunc) error {
	for ch := range s.sshNewChannel {
		stream, req, err := ch.Accept()
		if err != nil {
			return err
		}
		go ssh.DiscardRequests(req)
		go hf(stream, string(ch.ExtraData()))
	}
	return nil
}

func (s *SSHTunnel) Wait() error {
	return s.sshConn.Wait()
}

func NewSSHTunnel(conn *websocket.Conn) *SSHTunnel {
	return &SSHTunnel{conn: NewNetConnFromWsConn(conn)}
}
