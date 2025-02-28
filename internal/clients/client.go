package clients

import (
	"context"
	"log"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	azlog "github.com/Azure/azure-sdk-for-go/sdk/azcore/log"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/terraform-provider-azapi/internal/features"
)

type Client struct {
	// StopContext is used for propagating control from Terraform Core (e.g. Ctrl/Cmd+C)
	StopContext context.Context

	Features features.UserFeatures

	ResourceClient  *ResourceClient
	DataPlaneClient *DataPlaneClient

	Account ResourceManagerAccount
	Option  *Option
}

type Option struct {
	Cred                        azcore.TokenCredential
	ApplicationUserAgent        string
	Features                    features.UserFeatures
	SkipProviderRegistration    bool
	DisableCorrelationRequestID bool
	CloudCfg                    cloud.Configuration
	CustomCorrelationRequestID  string
	SubscriptionId              string
	TenantId                    string
	MaxGoSdkRetries             int32
}

// NOTE: it should be possible for this method to become Private once the top level Client's removed

func (client *Client) Build(ctx context.Context, o *Option) error {
	client.StopContext = ctx
	client.Features = o.Features
	client.Option = o

	azlog.SetListener(func(cls azlog.Event, msg string) {
		log.Printf("[DEBUG] %s %s: %s\n", time.Now().Format(time.StampMicro), cls, msg)
	})

	perCallPolicies := make([]policy.Policy, 0)
	perCallPolicies = append(perCallPolicies, withUserAgent(o.ApplicationUserAgent))
	if !o.DisableCorrelationRequestID {
		id := o.CustomCorrelationRequestID
		if id == "" {
			id = correlationRequestID()
		}
		perCallPolicies = append(perCallPolicies, withCorrelationRequestID(id))
	}
	perRetryPolicies := make([]policy.Policy, 0)
	perRetryPolicies = append(perRetryPolicies, NewLiveTrafficLogPolicy())

	allowedHeaders := []string{
		"Access-Control-Allow-Methods",
		"Access-Control-Allow-Origin",
		"Elapsed-Time",
		"Location",
		"Metadata",
		"Ocp-Automation-Accountid",
		"P3p",
		"Strict-Transport-Security",
		"Vary",
		"X-Content-Type-Options",
		"X-Frame-Options",
		"X-Ms-Correlation-Request-Id",
		"X-Ms-Ests-Server",
		"X-Ms-Failure-Cause",
		"X-Ms-Ratelimit-Remaining-Subscription-Reads",
		"X-Ms-Ratelimit-Remaining-Subscription-Writes",
		"X-Ms-Ratelimit-Remaining-Tenant-Reads",
		"X-Ms-Ratelimit-Remaining-Tenant-Writes",
		"X-Ms-Request-Id",
		"X-Ms-Routing-Request-Id",
		"X-Xss-Protection",
	}
	allowedQueryParams := []string{
		"api-version",
		"$skipToken",
	}

	resourceClient, err := NewResourceClient(o.Cred, &arm.ClientOptions{
		ClientOptions: policy.ClientOptions{
			Cloud: o.CloudCfg,
			// Disable the default telemetry policy, because it has a length limitation for user agent
			Telemetry: policy.TelemetryOptions{
				Disabled: true,
			},
			Logging: policy.LogOptions{
				IncludeBody:        true,
				AllowedHeaders:     allowedHeaders,
				AllowedQueryParams: allowedQueryParams,
			},
			PerCallPolicies:  perCallPolicies,
			PerRetryPolicies: perRetryPolicies,
			Retry:            policy.RetryOptions{MaxRetries: o.MaxGoSdkRetries},
		},
		DisableRPRegistration: o.SkipProviderRegistration,
	})
	if err != nil {
		return err
	}
	client.ResourceClient = resourceClient

	dataPlaneClient, err := NewDataPlaneClient(o.Cred, &arm.ClientOptions{
		ClientOptions: policy.ClientOptions{
			Cloud: o.CloudCfg,
			// Disable the default telemetry policy, because it has a length limitation for user agent
			Telemetry: policy.TelemetryOptions{
				Disabled: true,
			},
			Logging: policy.LogOptions{
				IncludeBody:        true,
				AllowedHeaders:     allowedHeaders,
				AllowedQueryParams: allowedQueryParams,
			},
			PerCallPolicies:  perCallPolicies,
			PerRetryPolicies: perRetryPolicies,
			Retry:            policy.RetryOptions{MaxRetries: o.MaxGoSdkRetries},
		},
		DisableRPRegistration: o.SkipProviderRegistration,
	})
	if err != nil {
		return err
	}
	client.DataPlaneClient = dataPlaneClient

	client.Account = NewResourceManagerAccount(o.TenantId, o.SubscriptionId, ParsedTokenClaimsObjectIDProvider(o.Cred, o.CloudCfg))

	return nil
}
