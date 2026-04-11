package main

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jlucaspains/sharp-cred-manager/internal/models"
	"github.com/jlucaspains/sharp-cred-manager/internal/services"
	"github.com/spf13/cobra"
)

var (
	verbose             bool
	validityDaysWarning int
	urls                []string
)

var rootCmd = &cobra.Command{
	Use:   "sharp-cred-manager",
	Short: "A tool to check certificates.",
	Long: `A command-line tool to check and manage SSL/TLS certificates.
	
This tool connects to a list of websites, downloads their TLS certificates, and checks their validity and expiration dates.`,
}

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check certificates for a list of websites",
	Long:  `Check the SSL/TLS certificates of specified websites.`,
	RunE:  runCheck,
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	checkCmd.Flags().IntVar(&validityDaysWarning, "warning-threshold", 90, "Number of days to trigger warning for certificate validity")
	checkCmd.Flags().StringArrayVar(&urls, "url", []string{}, "URL of the website to check")

	rootCmd.AddCommand(checkCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runCheck(cmd *cobra.Command, args []string) error {
	logger := setupLogger()

	parsedUrls := map[string]string{}
	for _, parameterUrl := range urls {
		logger.Debug("Parsing URL", "url", parameterUrl)

		parsedUrl, err := url.ParseRequestURI(parameterUrl)
		if err != nil {
			return fmt.Errorf("\033[31mInvalid URL %s: %w\033[0m", parameterUrl, err)
		}
		parsedUrls[parsedUrl.Host] = parameterUrl
	}

	if validityDaysWarning < 0 {
		return fmt.Errorf("\033[31mWarning threshold must be a non-negative integer\033[0m")
	}

	logger.Debug("Starting Sharp Cred Manager...", "urls", urls, "validityDaysWarning", validityDaysWarning)

	t := table.NewWriter()
	t.AppendHeader(table.Row{"Domain", "Common Name", "Status", "Details"})

	for domain, parsedUrl := range parsedUrls {
		logger.Debug("Checking certificate for url", "domain", domain, "url", parsedUrl)

		checkStatus, err := services.CheckCertStatus(models.CheckCertItem{
			Name: domain,
			Url:  parsedUrl,
			Type: models.CertCheckURL,
		}, validityDaysWarning)

		if err != nil {
			logger.Debug("Error checking certificate", "domain", domain, "error", err)
			t.AppendRow(table.Row{domain, "", "\033[0;31mError\033[0m", fmt.Sprintf("Error: %s", err)})
			continue
		}
		if checkStatus.IsValid {
			daysLeft := int(time.Until(checkStatus.CertEndDate).Hours() / 24)
			logger.Debug("Certificate is valid", "expires", daysLeft, "date", checkStatus.CertEndDate)

			status := "\033[0;32mValid\033[0m"
			if checkStatus.ExpirationWarning {
				status = "\033[0;33mWarning\033[0m"
			}

			t.AppendRow(table.Row{domain, checkStatus.CommonName, status, fmt.Sprintf("Expires in %d days on %s", daysLeft, checkStatus.CertEndDate.Format("2006-01-02"))})
		} else {
			logger.Debug("Certificate is invalid", "issues", strings.Join(checkStatus.ValidationIssues, ", "))
			t.AppendRow(table.Row{domain, checkStatus.CommonName, "\033[0;31mInvalid\033[0m", strings.Join(checkStatus.ValidationIssues, "\n")})
		}
	}

	fmt.Println(t.Render())

	return nil
}

func setupLogger() *slog.Logger {
	opts := &slog.HandlerOptions{}

	if verbose {
		opts.Level = slog.LevelDebug
	} else {
		opts.Level = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stdout, opts)
	logger := slog.New(handler)

	return logger
}
