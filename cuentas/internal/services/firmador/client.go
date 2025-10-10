package firmador

import "github.com/hashicorp/go-retryablehttp"

type Client struct {
	baseURL    string
	httpClient *retryablehttp.Client
}
