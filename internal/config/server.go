package config

import (
	"context"
	"fmt"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/go-github/v59/github"
	"github.com/kelseyhightower/envconfig"
	"golang.org/x/oauth2"
)

type ServerConfig struct {
	Stage                       string `envconfig:"STAGE" default:"dev"`
	ProjectID                   string `envconfig:"GOOGLE_CLOUD_PROJECT_ID" default:"go-semantic-release"`
	Port                        string `envconfig:"PORT" default:"8080"`
	BindAddress                 string `envconfig:"BIND_ADDRESS"`
	GitHubToken                 string `envconfig:"GITHUB_TOKEN" required:"true"`
	AdminAccessToken            string `envconfig:"ADMIN_ACCESS_TOKEN"`
	CloudflareR2Bucket          string `envconfig:"CLOUDFLARE_R2_BUCKET" required:"true"`
	CloudflareR2AccessKeyID     string `envconfig:"CLOUDFLARE_R2_ACCESS_KEY_ID" required:"true"`
	CloudflareR2SecretAccessKey string `envconfig:"CLOUDFLARE_R2_SECRET_ACCESS_KEY" required:"true"`
	CloudflareAccountID         string `envconfig:"CLOUDFLARE_ACCOUNT_ID" required:"true"`
	PluginCacheHost             string `envconfig:"PLUGIN_CACHE_HOST" required:"true"`
	DisableRequestCache         bool   `envconfig:"DISABLE_REQUEST_CACHE"`
	Version                     string
	DisableMetrics              bool `envconfig:"DISABLE_METRICS"`
}

func NewServerConfigFromEnv() (*ServerConfig, error) {
	var sCfg ServerConfig
	err := envconfig.Process("", &sCfg)
	if err != nil {
		return nil, err
	}
	return &sCfg, nil
}

func (s *ServerConfig) GetServerAddr() string {
	return s.BindAddress + ":" + s.Port
}

func (s *ServerConfig) CreateGitHubClient() *github.Client {
	oauthClient := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{AccessToken: s.GitHubToken}))
	return github.NewClient(oauthClient)
}

func (s *ServerConfig) r2CloudflareEndpointResolver(_, _ string, _ ...interface{}) (aws.Endpoint, error) {
	return aws.Endpoint{
		URL: fmt.Sprintf("https://%s.r2.cloudflarestorage.com", s.CloudflareAccountID),
	}, nil
}

func (s *ServerConfig) CreateS3Client() (*s3.Client, error) {
	staticCredentialsProvider := credentials.NewStaticCredentialsProvider(
		s.CloudflareR2AccessKeyID,
		s.CloudflareR2SecretAccessKey,
		"",
	)
	s3Cfg, err := awsConfig.LoadDefaultConfig(context.TODO(),
		awsConfig.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(s.r2CloudflareEndpointResolver)),
		awsConfig.WithCredentialsProvider(staticCredentialsProvider),
	)
	if err != nil {
		return nil, err
	}
	return s3.NewFromConfig(s3Cfg), nil
}

func (s *ServerConfig) GetBucket() *string {
	return &s.CloudflareR2Bucket
}

func (s *ServerConfig) GetPublicPluginCacheDownloadURL(path string) string {
	pPath, err := url.JoinPath(s.PluginCacheHost, path)
	if err != nil {
		panic(err)
	}
	return pPath
}
