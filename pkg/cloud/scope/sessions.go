package scope

import (
	"context"

	"github.com/pkg/errors"
	"github.com/vpsie/govpsie"
	"golang.org/x/oauth2"
)

// TokenSource ...
type TokenSource struct {
	AccessToken string
}

// Token return the oauth token.
func (t *TokenSource) Token() (*oauth2.Token, error) {
	token := &oauth2.Token{
		AccessToken: t.AccessToken,
	}
	return token, nil
}

func (c *VpsieClients) Session() (*govpsie.Client, error) {
	// accessToken := os.Getenv("VPSIE_ACCESS_TOKEN")
	accessToken := ""
	if accessToken == "" {
		return nil, errors.New("env var VPSIE_ACCESS_TOKEN is required, set in os env")
	}

	oc := oauth2.NewClient(context.Background(), &TokenSource{
		AccessToken: accessToken,
	})

	client := govpsie.NewClient(oc)

	client.SetUserAgent("cluster-api-provider-vpsie")
	client.SetRequestHeaders(map[string]string{
		"Vpsie-Auth": accessToken,
	})

	return client, nil
}
