package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"golang.org/x/oauth2"
	oauth2google "golang.org/x/oauth2/google"
	gcpopt "google.golang.org/api/option"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
	grpc "google.golang.org/grpc"
)

// We are going to override default endpoint if usePrivateEndpoint=true.
// This is needed if you want to route all requests to private Google API subnet
// BGP routed via your VPN tunnels.
const GooglePrivateEndpoint = "private.googleapis.com:443"

// Oauth2 scope for Secret Manager is documented here:
// https://cloud.google.com/secret-manager/docs/reference/rest/v1/projects.secrets/get
const oauth2scope = "https://www.googleapis.com/auth/cloud-platform"

// GCP SecretManager client implementation
type GCPVaultClient struct {
	ProjectID string

	client *secretmanager.Client
}

// Returns instance of client
func NewGCPVaultClient(ctx context.Context, projectID string, usePrivateEndpoint bool) (*GCPVaultClient, error) {

	// https://pkg.go.dev/google.golang.org/api/option#ClientOption
	clientOptions := []gcpopt.ClientOption{}

	// Rewrite Dialers to use Private GCP endpoint.
	if usePrivateEndpoint == true {
		// In case of Secret manager client we need to forward following endpoints:
		// * secretmanager.googleapis.com:443 -> private.googleapis.com:443
		// * oauth2.googleapis.com:443 -> private.googleapis.com:443
		// Here is the workaround with dialers - all remains the same in the GCP client,
		// except the Dialers, which create the underlying TCP connections.
		// See the alternative DNS workaround - https://cloud.google.com/vpc/docs/configure-private-google-access

		// Create new gRPC DialContext function: it replaces any address with Google private API Endpoint
		gRPCDialFunc := func(ctx context.Context, address string) (net.Conn, error) {
			return dialContextPrivateAPI(ctx, "tcp", address)
		}

		// Add Secret Manager client option to use custom gRPC dialer.
		clientOptions = append(clientOptions, gcpopt.WithGRPCDialOption(grpc.WithContextDialer(gRPCDialFunc)))

		// Create special HTTP client, which will be used by oauth2 flow.
		oauth2httpTransport := http.DefaultTransport.(*http.Transport)
		// Override the DialContext method here to forward everything to pivate GCP endpoint.
		oauth2httpTransport.DialContext = dialContextPrivateAPI
		oauth2httpClient := &http.Client{Transport: oauth2httpTransport, Timeout: 30 * time.Second}

		// Add the new HTTP client to oauth2 context as Value.
		// See https://stackoverflow.com/questions/38150891/how-to-pass-custom-client-to-golang-oauth2-exchange
		oauth2ctx := context.WithValue(ctx, oauth2.HTTPClient, oauth2httpClient)
		oauth2tokenSource, err := oauth2google.DefaultTokenSource(oauth2ctx, oauth2scope)
		if err != nil {
			return nil, err
		}

		// Add Secret Manager client option to use custom token source.
		clientOptions = append(clientOptions, gcpopt.WithTokenSource(oauth2tokenSource))
	}

	// create client
	smclient, err := secretmanager.NewClient(ctx, clientOptions...)

	client := &GCPVaultClient{
		ProjectID: projectID,
		client:    smclient,
	}

	return client, err
}

// Returns latest version of secret.
func (c *GCPVaultClient) GetSecret(ctx context.Context, name string) ([]byte, error) {

	// Сomplete id of the secret.
	secretId := fmt.Sprintf(`projects/%s/secrets/%s/versions/latest`, c.ProjectID, name)

	// Build request body.
	accessRequest := &secretmanagerpb.AccessSecretVersionRequest{Name: secretId}

	// Do request.
	result, err := c.client.AccessSecretVersion(ctx, accessRequest)
	if err != nil {
		return nil, err
	}

	return result.Payload.Data, nil
}

// Close connections and stop background jobs.
func (c *GCPVaultClient) Close() error {
	return c.client.Close()
}

// This is a DialContext implementation which replaces any address with GCP Private API Endpoint.
func dialContextPrivateAPI(ctx context.Context, network, address string) (net.Conn, error) {
	dialer := &net.Dialer{
		Timeout:   60 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}
	return dialer.DialContext(ctx, network, GooglePrivateEndpoint)
}
