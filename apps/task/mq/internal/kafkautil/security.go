package kafkautil

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/zeromicro/go-queue/kq"
)

func Dialer(c kq.KqConf) (*kafka.Dialer, error) {
	dialer := &kafka.Dialer{}
	if c.Username != "" || c.Password != "" {
		if c.Username == "" || c.Password == "" {
			return nil, fmt.Errorf("both Kafka username and password are required")
		}
		dialer.SASLMechanism = plain.Mechanism{Username: c.Username, Password: c.Password}
	}
	tlsConfig, err := TLSConfig(c.CaFile)
	if err != nil {
		return nil, err
	}
	dialer.TLS = tlsConfig
	return dialer, nil
}

func Transport(c kq.KqConf) (*kafka.Transport, error) {
	transport := &kafka.Transport{ClientID: c.Name}
	if c.Username != "" || c.Password != "" {
		if c.Username == "" || c.Password == "" {
			return nil, fmt.Errorf("both Kafka username and password are required")
		}
		transport.SASL = plain.Mechanism{Username: c.Username, Password: c.Password}
	}
	tlsConfig, err := TLSConfig(c.CaFile)
	if err != nil {
		return nil, err
	}
	transport.TLS = tlsConfig
	return transport, nil
}

func TLSConfig(caFile string) (*tls.Config, error) {
	if caFile == "" {
		return nil, nil
	}
	pem, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("read Kafka CA file: %w", err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(pem) {
		return nil, fmt.Errorf("Kafka CA file contains no certificate")
	}
	return &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}, nil
}
