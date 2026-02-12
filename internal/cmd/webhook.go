package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/notion"
)

func newWebhookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "webhook",
		Aliases: []string{"wh"},
		Short:   "Webhook utilities",
		Long:    "Verify and parse Notion webhook payloads.",
	}

	cmd.AddCommand(newWebhookVerifyCmd())
	cmd.AddCommand(newWebhookParseCmd())

	return cmd
}

func newWebhookVerifyCmd() *cobra.Command {
	var secret string
	var signature string
	var payloadFile string

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify or compute a webhook signature",
		Long: `Verify a Notion webhook signature against a payload.

If --signature is omitted, this command prints the computed signature for the payload.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if secret == "" {
				return fmt.Errorf("--secret is required")
			}

			payload, err := readPayload(payloadFile)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			printer := printerForContext(ctx)

			if signature == "" {
				return printer.Print(ctx, map[string]interface{}{
					"signature": notion.ComputeWebhookSignature(secret, payload),
				})
			}

			valid := notion.VerifyWebhookSignature(secret, payload, signature)
			if !valid {
				return fmt.Errorf("signature verification failed")
			}
			return printer.Print(ctx, map[string]interface{}{
				"valid": true,
			})
		},
	}

	cmd.Flags().StringVar(&secret, "secret", "", "Webhook signing secret")
	cmd.Flags().StringVar(&signature, "signature", "", "Signature from X-Notion-Signature header")
	cmd.Flags().StringVar(&payloadFile, "payload", "", "Path to payload file (defaults to stdin)")

	return cmd
}

func newWebhookParseCmd() *cobra.Command {
	var payloadFile string

	cmd := &cobra.Command{
		Use:   "parse",
		Short: "Parse a webhook payload",
		Long:  "Parse a webhook payload and print the decoded event or verification request.",
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := readPayload(payloadFile)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			printer := printerForContext(ctx)

			if notion.IsVerificationRequest(payload) {
				req, err := notion.ParseWebhookVerification(payload)
				if err != nil {
					return err
				}
				return printer.Print(ctx, map[string]interface{}{
					"type": "verification",
					"data": req,
				})
			}

			event, err := notion.ParseWebhookEvent(payload)
			if err != nil {
				return err
			}
			return printer.Print(ctx, map[string]interface{}{
				"type": "event",
				"data": event,
			})
		},
	}

	cmd.Flags().StringVar(&payloadFile, "payload", "", "Path to payload file (defaults to stdin)")

	return cmd
}

func readPayload(path string) ([]byte, error) {
	if path == "" {
		return io.ReadAll(os.Stdin)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read payload: %w", err)
	}
	return data, nil
}
