package certmark

import (
	"crypto/md5"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"os"

	"moul.io/drunken-bishop/drunkenbishop"
)

const (
	GraphRandomart GraphType = "randomart"
	GraphIdenticon GraphType = "identicon"
)

type GraphType string

type GraphConfig struct {
	Type   GraphType
	Colour string
}

func GenerateGraphicFromCert(cert *x509.Certificate, config GraphConfig) []byte {
	hash := md5.Sum(cert.Raw)
	hashStr := hex.EncodeToString(hash[:])

	return GenerateGraphic([]byte(hashStr), config)
}

func GenerateGraphic(inputBytes []byte, config GraphConfig) []byte {
	switch config.Type {
	case GraphRandomart:
		room := drunkenbishop.FromBytes(inputBytes)
		return []byte(room.String())

	case GraphIdenticon:
		outBytes, err := GenerateIdenticon(inputBytes, DefaultIdenticonConfig)
		if err != nil {
			panic(err)
		}
		return outBytes

	default:
		return nil
	}
}

type TLSConnectionInfo struct {
	Version      uint16
	CipherSuite  uint16
	ServerName   string
}

func (t *TLSConnectionInfo) VersionString() string {
	switch t.Version {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("Unknown (0x%04x)", t.Version)
	}
}

func (t *TLSConnectionInfo) CipherSuiteString() string {
	return tls.CipherSuiteName(t.CipherSuite)
}

type CertResult struct {
	Cert       *x509.Certificate
	TLSInfo    *TLSConnectionInfo
}

func ReadCertFile(name string) (*CertResult, error) {
	certBytes, err := os.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", name, err)
	}

	block, _ := pem.Decode(certBytes)
	if block != nil {
		certBytes = block.Bytes
	}

	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cert data: %w", err)
	}

	return &CertResult{Cert: cert}, nil
}

func ReadCertHost(hostport string, insecureSkipVerify bool) (*CertResult, error) {
	DefaultPort := ":443"

	_, port, err := net.SplitHostPort(hostport)
	if err != nil {
		if errors.Is(err, errors.New("missing port in address")) {
			return nil, fmt.Errorf("invalid url: %s", err)
		}
	}

	if port == "" {
		hostport = fmt.Sprintf("%s%s", hostport, DefaultPort)
	}

	conf := &tls.Config{
		InsecureSkipVerify: insecureSkipVerify,
	}

	conn, err := tls.Dial("tcp", hostport, conf)
	if err != nil {
		return nil, fmt.Errorf("failed to reach host %s: %w", hostport, err)
	}
	defer conn.Close()

	state := conn.ConnectionState()
	certs := state.PeerCertificates

	return &CertResult{
		Cert: certs[0],
		TLSInfo: &TLSConnectionInfo{
			Version:     state.Version,
			CipherSuite: state.CipherSuite,
			ServerName:  state.ServerName,
		},
	}, nil
}
