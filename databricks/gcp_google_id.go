package databricks

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/databricks/sdk-go/databricks/internal"
	"golang.org/x/oauth2"
	"google.golang.org/api/impersonate"
	"google.golang.org/api/option"
)

type GoogleDefaultCredentials struct {
	GoogleServiceAccount string `name:"google_service_account" env:"DATABRICKS_GOOGLE_SERVICE_ACCOUNT"`

	// options used to enable unit testing mode for OIDC
	opts []option.ClientOption
}

func (c GoogleDefaultCredentials) Name() string {
	return "google-id"
}

func (c GoogleDefaultCredentials) Configure(ctx context.Context, cfg *Config) (func(*http.Request) error, error) {
	if c.GoogleServiceAccount == "" || !cfg.IsGcp() {
		return nil, nil
	}
	inner, err := c.idTokenSource(ctx, cfg.Host, c.GoogleServiceAccount, c.opts...)
	if err != nil {
		return nil, err
	}
	if !cfg.IsAccountsClient() {
		log.Printf("[INFO] Using Google Default Application Credentials for Workspace")
		return internal.RefreshableVisitor(inner), nil
	}
	// source for generateAccessToken
	platform, err := impersonate.CredentialsTokenSource(ctx, impersonate.CredentialsConfig{
		TargetPrincipal: c.GoogleServiceAccount,
		Scopes: []string{
			"https://www.googleapis.com/auth/cloud-platform",
			"https://www.googleapis.com/auth/compute",
		},
	}, c.opts...)
	if err != nil {
		return nil, err
	}
	log.Printf("[INFO] Using Google Default Application Credentials for Accounts API")
	return internal.ServiceToServiceVisitor(inner, platform, "X-Databricks-GCP-SA-Access-Token"), nil
}

func (c GoogleDefaultCredentials) idTokenSource(ctx context.Context, host, serviceAccount string,
	opts ...option.ClientOption) (oauth2.TokenSource, error) {
	ts, err := impersonate.IDTokenSource(ctx, impersonate.IDTokenConfig{
		Audience:        host,
		TargetPrincipal: serviceAccount,
		IncludeEmail:    true,
	}, opts...)
	if err != nil {
		err = fmt.Errorf("could not obtain OIDC token. %w Running 'gcloud auth application-default login' may help", err)
		return nil, err
	}
	return ts, err
}