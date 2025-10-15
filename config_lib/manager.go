package config_lib

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	sm "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type Manager struct {
	client *sm.Client
}

func New(ctx context.Context, region string, optFns ...func(*config.LoadOptions) error) (*Manager, error) {
	loadOpts := []func(*config.LoadOptions) error{}

	if region != "" {
		loadOpts = append(loadOpts, config.WithRegion(region))
	}
	loadOpts = append(loadOpts, optFns...)

	cfg, err := config.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return nil, err
	}
	return &Manager{client: sm.NewFromConfig(cfg)}, nil
}

func (m *Manager) GetSecretString(ctx context.Context, secretID string, versionStage string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	out, err := m.client.GetSecretValue(ctx, &sm.GetSecretValueInput{
		SecretId:     aws.String(secretID),
		VersionStage: aws.String(versionStage),
	})
	if err != nil {
		return "", err
	}

	if out.SecretString == nil {
		return "", nil
	}
	return *out.SecretString, nil
}
