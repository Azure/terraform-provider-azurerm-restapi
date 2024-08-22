package clients

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	armpolicy "github.com/Azure/azure-sdk-for-go/sdk/azcore/arm/policy"
	armruntime "github.com/Azure/azure-sdk-for-go/sdk/azcore/arm/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/terraform-provider-azapi/internal/services/parse"
	"github.com/cenkalti/backoff/v4"
)

type DataPlaneClient struct {
	credential      azcore.TokenCredential
	clientOptions   *arm.ClientOptions
	cachedPipelines map[string]runtime.Pipeline
	syncMux         sync.Mutex
}

type DataPlaneClientRetryableErrors struct {
	client  DataPlaneRequester          // client is a DataPlaneRequester interface to allow mocking
	backoff *backoff.ExponentialBackOff // backoff is the backoff configuration for retrying
	errors  []regexp.Regexp             // errors is the list of errors regexp to retry on
}

type DataPlaneRequester interface {
	CreateOrUpdateThenPoll(ctx context.Context, id parse.DataPlaneResourceId, body interface{}) (interface{}, error)
	Get(ctx context.Context, id parse.DataPlaneResourceId) (interface{}, error)
	DeleteThenPoll(ctx context.Context, id parse.DataPlaneResourceId) (interface{}, error)
	Action(ctx context.Context, resourceID string, action string, apiVersion string, method string, body interface{}) (interface{}, error)
}

var (
	_ DataPlaneRequester = &DataPlaneClient{}
	_ DataPlaneRequester = &DataPlaneClientRetryableErrors{}
)

// NewDataPlaneClientRetryableErrors creates a new ResourceClientRetryableErrors.
func NewDataPlaneClientRetryableErrors(client DataPlaneRequester, bkof *backoff.ExponentialBackOff, errRegExps []regexp.Regexp) *DataPlaneClientRetryableErrors {
	rcre := &DataPlaneClientRetryableErrors{
		client:  client,
		backoff: bkof,
		errors:  errRegExps,
	}
	rcre.backoff.Reset()
	return rcre
}

func NewDataPlaneClient(credential azcore.TokenCredential, opt *arm.ClientOptions) (*DataPlaneClient, error) {
	if opt == nil {
		opt = &arm.ClientOptions{}
	}
	return &DataPlaneClient{
		credential:      credential,
		clientOptions:   opt,
		cachedPipelines: make(map[string]runtime.Pipeline),
		syncMux:         sync.Mutex{},
	}, nil
}

// WithRetry configures the retryable errors for the client.
func (client *DataPlaneClient) WithRetry(bkof *backoff.ExponentialBackOff, errRegExps []regexp.Regexp) *DataPlaneClientRetryableErrors {
	rcre := &DataPlaneClientRetryableErrors{
		client:  client,
		backoff: bkof,
		errors:  errRegExps,
	}
	rcre.backoff.Reset()
	return rcre
}

func (client *DataPlaneClient) cachedPipeline(rawUrl string) (runtime.Pipeline, error) {
	client.syncMux.Lock()
	defer client.syncMux.Unlock()

	parsedUrl, err := url.Parse(rawUrl)
	if err != nil {
		return runtime.Pipeline{}, err
	}
	serviceName := cloud.ResourceManager
	cloud := client.clientOptions.Cloud
	host := parsedUrl.Host
	for name, serviceConfiguration := range cloud.Services {
		if strings.HasSuffix(host, strings.TrimPrefix(serviceConfiguration.Endpoint, "https://")) {
			serviceName = name
			break
		}
	}

	if pipeline, ok := client.cachedPipelines[string(serviceName)]; ok {
		return pipeline, nil
	}

	plOpt := runtime.PipelineOptions{}
	plOpt.APIVersion.Name = "api-version"
	authPolicy := armruntime.NewBearerTokenPolicy(client.credential, &armpolicy.BearerTokenOptions{Scopes: []string{cloud.Services[serviceName].Audience + "/.default"}})
	plOpt.PerRetry = append(plOpt.PerRetry, authPolicy)
	pl := runtime.NewPipeline(moduleName, moduleVersion, plOpt, &client.clientOptions.ClientOptions)

	client.cachedPipelines[string(serviceName)] = pl
	return pl, nil
}

func (client *DataPlaneClient) CreateOrUpdateThenPoll(ctx context.Context, id parse.DataPlaneResourceId, body interface{}) (interface{}, error) {
	// build request
	urlPath := fmt.Sprintf("https://%s", id.AzureResourceId)
	req, err := runtime.NewRequest(ctx, http.MethodPut, urlPath)
	if err != nil {
		return nil, err
	}
	reqQP := req.Raw().URL.Query()
	reqQP.Set("api-version", id.ApiVersion)
	req.Raw().URL.RawQuery = reqQP.Encode()
	req.Raw().Header.Set("Accept", "application/json")
	err = runtime.MarshalAsJSON(req, body)
	if err != nil {
		return nil, err
	}

	// send request
	pipeline, err := client.cachedPipeline(urlPath)
	if err != nil {
		return nil, err
	}
	resp, err := pipeline.Do(req)
	if err != nil {
		return nil, err
	}
	if !runtime.HasStatusCode(resp, http.StatusOK, http.StatusCreated, http.StatusAccepted) {
		return nil, runtime.NewResponseError(resp)
	}

	// poll until done
	pt, err := runtime.NewPoller[interface{}](resp, pipeline, nil)
	if err == nil {
		resp, err := pt.PollUntilDone(ctx, &runtime.PollUntilDoneOptions{
			Frequency: 10 * time.Second,
		})
		return resp, err
	}

	// unmarshal response
	var responseBody interface{}
	if err := runtime.UnmarshalAsJSON(resp, &responseBody); err != nil {
		return nil, err
	}
	return responseBody, nil
}

func (client *DataPlaneClient) Get(ctx context.Context, id parse.DataPlaneResourceId) (interface{}, error) {
	// build request
	urlPath := fmt.Sprintf("https://%s", id.AzureResourceId)
	req, err := runtime.NewRequest(ctx, http.MethodGet, urlPath)
	if err != nil {
		return nil, err
	}
	reqQP := req.Raw().URL.Query()
	reqQP.Set("api-version", id.ApiVersion)
	req.Raw().URL.RawQuery = reqQP.Encode()
	req.Raw().Header.Set("Accept", "application/json")

	// send request
	pipeline, err := client.cachedPipeline(urlPath)
	if err != nil {
		return nil, err
	}
	resp, err := pipeline.Do(req)
	if err != nil {
		return nil, err
	}
	if !runtime.HasStatusCode(resp, http.StatusOK) {
		return nil, runtime.NewResponseError(resp)
	}

	// unmarshal response
	var responseBody interface{}
	if err := runtime.UnmarshalAsJSON(resp, &responseBody); err != nil {
		return nil, err
	}
	return responseBody, nil
}

func (client *DataPlaneClient) DeleteThenPoll(ctx context.Context, id parse.DataPlaneResourceId) (interface{}, error) {
	// build request
	urlPath := fmt.Sprintf("https://%s", id.AzureResourceId)
	req, err := runtime.NewRequest(ctx, http.MethodDelete, urlPath)
	if err != nil {
		return nil, err
	}
	reqQP := req.Raw().URL.Query()
	reqQP.Set("api-version", id.ApiVersion)
	req.Raw().URL.RawQuery = reqQP.Encode()
	req.Raw().Header.Set("Accept", "application/json")

	// send request
	pipeline, err := client.cachedPipeline(urlPath)
	if err != nil {
		return nil, err
	}
	resp, err := pipeline.Do(req)
	if err != nil {
		return nil, err
	}
	if !runtime.HasStatusCode(resp, http.StatusOK, http.StatusAccepted, http.StatusNoContent) {
		return nil, runtime.NewResponseError(resp)
	}

	// poll until done
	pt, err := runtime.NewPoller[interface{}](resp, pipeline, nil)
	if err == nil {
		resp, err := pt.PollUntilDone(ctx, &runtime.PollUntilDoneOptions{
			Frequency: 10 * time.Second,
		})
		return resp, err
	}

	// unmarshal response
	var responseBody interface{}
	if err := runtime.UnmarshalAsJSON(resp, &responseBody); err != nil {
		return nil, err
	}
	return responseBody, nil
}

func (client *DataPlaneClient) Action(ctx context.Context, resourceID string, action string, apiVersion string, method string, body interface{}) (interface{}, error) {
	// build request
	urlPath := fmt.Sprintf("https://%s", resourceID)
	if action != "" {
		urlPath = fmt.Sprintf("%s/%s", resourceID, action)
	}
	req, err := runtime.NewRequest(ctx, method, urlPath)
	if err != nil {
		return nil, err
	}
	reqQP := req.Raw().URL.Query()
	reqQP.Set("api-version", apiVersion)
	req.Raw().URL.RawQuery = reqQP.Encode()
	req.Raw().Header.Set("Accept", "application/json")
	if method != "GET" && body != nil {
		err = runtime.MarshalAsJSON(req, body)
	}
	if err != nil {
		return nil, err
	}

	// send request
	pipeline, err := client.cachedPipeline(urlPath)
	if err != nil {
		return nil, err
	}
	resp, err := pipeline.Do(req)
	if err != nil {
		return nil, err
	}
	if !runtime.HasStatusCode(resp, http.StatusOK, http.StatusCreated, http.StatusAccepted) {
		return nil, runtime.NewResponseError(resp)
	}

	// poll until done
	pt, err := runtime.NewPoller[interface{}](resp, pipeline, nil)
	if err == nil {
		resp, err := pt.PollUntilDone(ctx, &runtime.PollUntilDoneOptions{
			Frequency: 10 * time.Second,
		})
		return resp, err
	}

	// unmarshal response
	var responseBody interface{}
	contentType := resp.Header.Get("Content-Type")
	switch {
	case strings.Contains(contentType, "text/plain"):
		payload, err := runtime.Payload(resp)
		if err != nil {
			return nil, err
		}
		responseBody = string(payload)
	case strings.Contains(contentType, "application/json"):
		if err := runtime.UnmarshalAsJSON(resp, &responseBody); err != nil {
			return nil, err
		}
	default:
	}
	return responseBody, nil
}

func (retryclient *DataPlaneClientRetryableErrors) CreateOrUpdateThenPoll(ctx context.Context, id parse.DataPlaneResourceId, body interface{}) (interface{}, error) {
	if retryclient.backoff == nil || len(retryclient.errors) == 0 {
		return nil, fmt.Errorf("retry is not configured, please call WithRetry() first")
	}
	op := backoff.OperationWithData[interface{}](
		func() (interface{}, error) {
			data, err := retryclient.client.CreateOrUpdateThenPoll(ctx, id, body)
			if err != nil {
				for _, e := range retryclient.errors {
					if e.MatchString(err.Error()) {
						return data, err
					}
				}
				return nil, &backoff.PermanentError{Err: err}
			}
			return data, err
		})
	exbo := backoff.WithContext(retryclient.backoff, ctx)
	return backoff.RetryWithData[interface{}](op, exbo)
}

func (retryclient *DataPlaneClientRetryableErrors) Get(ctx context.Context, id parse.DataPlaneResourceId) (interface{}, error) {
	if retryclient.backoff == nil || len(retryclient.errors) == 0 {
		return nil, fmt.Errorf("retry is not configured, please call WithRetry() first")
	}
	op := backoff.OperationWithData[interface{}](
		func() (interface{}, error) {
			data, err := retryclient.client.Get(ctx, id)
			if err != nil {
				for _, e := range retryclient.errors {
					if e.MatchString(err.Error()) {
						return data, err
					}
				}
				return nil, &backoff.PermanentError{Err: err}
			}
			return data, err
		})
	exbo := backoff.WithContext(retryclient.backoff, ctx)
	return backoff.RetryWithData[interface{}](op, exbo)
}

func (retryclient *DataPlaneClientRetryableErrors) DeleteThenPoll(ctx context.Context, id parse.DataPlaneResourceId) (interface{}, error) {
	if retryclient.backoff == nil || len(retryclient.errors) == 0 {
		return nil, fmt.Errorf("retry is not configured, please call WithRetry() first")
	}
	op := backoff.OperationWithData[interface{}](
		func() (interface{}, error) {
			data, err := retryclient.client.DeleteThenPoll(ctx, id)
			if err != nil {
				for _, e := range retryclient.errors {
					if e.MatchString(err.Error()) {
						return data, err
					}
				}
				return nil, &backoff.PermanentError{Err: err}
			}
			return data, err
		})
	exbo := backoff.WithContext(retryclient.backoff, ctx)
	return backoff.RetryWithData[interface{}](op, exbo)
}

func (retryclient *DataPlaneClientRetryableErrors) Action(ctx context.Context, resourceID string, action string, apiVersion string, method string, body interface{}) (interface{}, error) {
	if retryclient.backoff == nil || len(retryclient.errors) == 0 {
		return nil, fmt.Errorf("retry is not configured, please call WithRetry() first")
	}
	op := backoff.OperationWithData[interface{}](
		func() (interface{}, error) {
			data, err := retryclient.client.Action(ctx, resourceID, action, apiVersion, method, body)
			if err != nil {
				for _, e := range retryclient.errors {
					if e.MatchString(err.Error()) {
						return data, err
					}
				}
				return nil, &backoff.PermanentError{Err: err}
			}
			return data, err
		})
	exbo := backoff.WithContext(retryclient.backoff, ctx)
	return backoff.RetryWithData[interface{}](op, exbo)
}
