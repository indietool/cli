package cmd

import (
	"errors"
	"fmt"
	"io/fs"

	"indietool/cli/certmark"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var (
	certGraphType string
	certInsecure  bool
)

var certShowCmd = &cobra.Command{
	Use:   "show <cert-file|hostname>",
	Short: "Show certificate details and visual fingerprint",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		target := args[0]

		var result *certmark.CertResult
		var isHost bool

		result, err := certmark.ReadCertFile(target)
		if err != nil {
			log.Debugf("%s", err)
			var pe *fs.PathError
			if !errors.As(err, &pe) {
				log.Fatalf("%s", err)
			}

			result, err = certmark.ReadCertHost(target, certInsecure)
			if err != nil {
				log.Fatalf("%s", err)
			}
			isHost = true
		}

		cert := result.Cert

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
	},
}

func init() {
	certCmd.AddCommand(certShowCmd)

	certShowCmd.Flags().StringVar(&certGraphType, "type", "identicon", "Graph type: identicon or randomart")
	certShowCmd.Flags().BoolVar(&certInsecure, "insecure", false, "Skip TLS verification when connecting to host")
}
