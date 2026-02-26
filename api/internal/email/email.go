package email

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	sesv2 "github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

//go:embed templates/welcome.html
var welcomeTemplate string

type Service struct {
	client    *sesv2.Client
	from      string
	configSet string
	logger    *slog.Logger
}

func NewService(client *sesv2.Client, from, configSet string, logger *slog.Logger) *Service {
	return &Service{
		client:    client,
		from:      from,
		configSet: configSet,
		logger:    logger,
	}
}

func (s *Service) SendWelcome(ctx context.Context, to, firstName string) error {
	greeting := "Hi there,"
	if firstName != "" {
		greeting = fmt.Sprintf("Hi %s,", firstName)
	}

	body := strings.ReplaceAll(welcomeTemplate, "{{.Greeting}}", greeting)

	input := &sesv2.SendEmailInput{
		FromEmailAddress:     aws.String(s.from),
		ConfigurationSetName: aws.String(s.configSet),
		Destination: &types.Destination{
			ToAddresses: []string{to},
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{
					Data:    aws.String("You're in — start exploring federal opportunities"),
					Charset: aws.String("UTF-8"),
				},
				Body: &types.Body{
					Html: &types.Content{
						Data:    aws.String(body),
						Charset: aws.String("UTF-8"),
					},
				},
			},
		},
	}

	_, err := s.client.SendEmail(ctx, input)
	if err != nil {
		return fmt.Errorf("sending welcome email to %s: %w", to, err)
	}

	s.logger.Info("welcome email sent", "to", to)
	return nil
}
