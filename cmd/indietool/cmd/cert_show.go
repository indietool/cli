package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"math/big"
	"strings"
	"time"

	"github.com/indietool/cli/certmark"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var (
	certGraphType string
	certInsecure  bool
)

// certShowJSON is the machine-readable shape for cert show --json.
type certShowJSON struct {
	Source      string          `json:"source"` // "file" or "host"
	Target      string          `json:"target"`
	Serial      string          `json:"serial"`
	Subject     string          `json:"subject"`
	Issuer      string          `json:"issuer"`
	IssuerOrg   []string        `json:"issuer_org,omitempty"`
	NotBefore   time.Time       `json:"not_before"`
	NotAfter    time.Time       `json:"not_after"`
	DNSNames    []string        `json:"dns_names,omitempty"`
	TLS         *certShowTLSJSON `json:"tls,omitempty"`
	CrtSH       string          `json:"crt_sh,omitempty"`
	Fingerprint string          `json:"fingerprint_md5,omitempty"`
}

type certShowTLSJSON struct {
	Version     string `json:"version"`
	CipherSuite string `json:"cipher_suite"`
}

var certShowCmd = &cobra.Command{
	Use:   "show <cert-file|hostname>",
	Short: "Show certificate details and visual fingerprint",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]

		var result *certmark.CertResult
		var isHost bool

		result, err := certmark.ReadCertFile(target)
		if err != nil {
			log.Debugf("%s", err)
			var pe *fs.PathError
			if !errors.As(err, &pe) {
				return err
			}

			result, err = certmark.ReadCertHost(target, certInsecure)
			if err != nil {
				return err
			}
			isHost = true
		}

		cert := result.Cert

		if jsonOutput {
			out := certShowJSON{
				Source:    "file",
				Target:    target,
				Serial:    formatSerial(cert.SerialNumber),
				Subject:   cert.Subject.CommonName,
				Issuer:    cert.Issuer.CommonName,
				IssuerOrg: cert.Issuer.Organization,
				NotBefore: cert.NotBefore,
				NotAfter:  cert.NotAfter,
				DNSNames:  cert.DNSNames,
			}
			if isHost {
				out.Source = "host"
				out.CrtSH = fmt.Sprintf("https://crt.sh/?q=%s", target)
			}
			if result.TLSInfo != nil {
				out.TLS = &certShowTLSJSON{
					Version:     result.TLSInfo.VersionString(),
					CipherSuite: result.TLSInfo.CipherSuiteString(),
				}
			}
			return printJSON(out)
		}

		if result.TLSInfo != nil {
			fmt.Printf("** TLS Connection **\n")
			fmt.Printf("Version: %s\n", result.TLSInfo.VersionString())
			fmt.Printf("Cipher Suite: %s\n", result.TLSInfo.CipherSuiteString())
			fmt.Printf("\n")
		}

		fmt.Printf("Serial: % x\n", cert.SerialNumber)
		fmt.Printf("Subject: %s\n", cert.Subject.CommonName)
		fmt.Printf("Not Before: %s\n", cert.NotBefore)
		fmt.Printf("Not After: %s\n", cert.NotAfter)
		fmt.Printf("Issuer: %s\n", cert.Issuer.CommonName)
		fmt.Printf("Issuer Org: %s\n", cert.Issuer.Organization)
		if isHost {
			fmt.Printf("crt.sh: https://crt.sh/?q=%s\n", target)
		}

		graphType := certmark.GraphType(certGraphType)
		fmt.Printf("\n%s\n", certmark.GenerateGraphicFromCert(cert, certmark.GraphConfig{Type: graphType}))
		return nil
	},
}

func formatSerial(n *big.Int) string {
	if n == nil {
		return ""
	}
	// Match human output spacing loosely as hex pairs without requiring spaces in JSON
	hex := fmt.Sprintf("%x", n)
	if len(hex)%2 == 1 {
		hex = "0" + hex
	}
	var b strings.Builder
	for i := 0; i < len(hex); i += 2 {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(hex[i : i+2])
	}
	return b.String()
}

func init() {
	certCmd.AddCommand(certShowCmd)

	certShowCmd.Flags().StringVar(&certGraphType, "type", "identicon", "Graph type: identicon or randomart")
	certShowCmd.Flags().BoolVar(&certInsecure, "insecure", false, "Skip TLS verification when connecting to host")
}
