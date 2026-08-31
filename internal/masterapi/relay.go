package masterapi

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

const (
	RelayRoleHost   = "host"
	RelayRoleClient = "client"

	maxRelayHandshakeBytes = 4096
	relayHandshakeTimeout  = 5 * time.Second
)

type RelayHandshake struct {
	Role     string `json:"role"`
	ID       string `json:"id"`
	Protocol int    `json:"protocol"`
}

type RelayAck struct {
	OK        bool   `json:"ok"`
	Error     string `json:"error,omitempty"`
	Challenge string `json:"challenge,omitempty"`
}

type RelayProof struct {
	Proof string `json:"proof"`
}

func WriteRelayHandshake(conn net.Conn, hs RelayHandshake) error {
	return writeRelayJSON(conn, hs)
}

func ReadRelayHandshake(conn net.Conn) (RelayHandshake, error) {
	var hs RelayHandshake
	if err := readRelayJSON(conn, &hs); err != nil {
		return RelayHandshake{}, err
	}
	return hs, nil
}

func WriteRelayAck(conn net.Conn, ack RelayAck) error {
	return writeRelayJSON(conn, ack)
}

func ReadRelayAck(conn net.Conn) (RelayAck, error) {
	var ack RelayAck
	if err := readRelayJSON(conn, &ack); err != nil {
		return RelayAck{}, err
	}
	return ack, nil
}

func WriteRelayProof(conn net.Conn, proof RelayProof) error {
	return writeRelayJSON(conn, proof)
}

func ReadRelayProof(conn net.Conn) (RelayProof, error) {
	var proof RelayProof
	if err := readRelayJSON(conn, &proof); err != nil {
		return RelayProof{}, err
	}
	return proof, nil
}

func NewRelayChallenge() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func RelayProofForSecret(secret, challenge string) string {
	secret = strings.TrimSpace(secret)
	challenge = strings.TrimSpace(challenge)
	if secret == "" || challenge == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(challenge))
	return hex.EncodeToString(mac.Sum(nil))
}

func VerifyRelayProof(secret, challenge, proof string) bool {
	expected := RelayProofForSecret(secret, challenge)
	proof = strings.TrimSpace(proof)
	if expected == "" || proof == "" {
		return false
	}
	expectedBytes, err := hex.DecodeString(expected)
	if err != nil {
		return false
	}
	proofBytes, err := hex.DecodeString(proof)
	if err != nil {
		return false
	}
	return hmac.Equal(expectedBytes, proofBytes)
}

func writeRelayJSON(conn net.Conn, value any) error {
	if conn == nil {
		return errors.New("nil relay connection")
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if len(payload) == 0 || len(payload) > maxRelayHandshakeBytes {
		return errors.New("relay handshake too large")
	}
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(payload)))
	_ = conn.SetWriteDeadline(time.Now().Add(relayHandshakeTimeout))
	defer conn.SetWriteDeadline(time.Time{})
	if err := writeFull(conn, header[:]); err != nil {
		return err
	}
	return writeFull(conn, payload)
}

func readRelayJSON(conn net.Conn, dst any) error {
	if conn == nil || dst == nil {
		return errors.New("invalid relay read")
	}
	_ = conn.SetReadDeadline(time.Now().Add(relayHandshakeTimeout))
	var header [4]byte
	if _, err := io.ReadFull(conn, header[:]); err != nil {
		return err
	}
	n := int(binary.BigEndian.Uint32(header[:]))
	if n <= 0 || n > maxRelayHandshakeBytes {
		return fmt.Errorf("invalid relay handshake size: %d", n)
	}
	payload := make([]byte, n)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return err
	}
	dec := json.NewDecoder(bytesReader(payload))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	_ = conn.SetReadDeadline(time.Time{})
	_ = conn.SetWriteDeadline(time.Time{})
	return nil
}

func writeFull(w io.Writer, data []byte) error {
	for len(data) > 0 {
		n, err := w.Write(data)
		if err != nil {
			return err
		}
		if n <= 0 {
			return io.ErrUnexpectedEOF
		}
		data = data[n:]
	}
	return nil
}

type byteReader struct {
	b []byte
	i int
}

func bytesReader(b []byte) *byteReader { return &byteReader{b: b} }

func (r *byteReader) Read(p []byte) (int, error) {
	if r.i >= len(r.b) {
		return 0, io.EOF
	}
	n := copy(p, r.b[r.i:])
	r.i += n
	return n, nil
}
