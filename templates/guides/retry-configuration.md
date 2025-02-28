---
layout: "azapi"
page_title: "Feature: Retry Configuration"
description: |-
  This guide will describe how to configure the retry behavior in the AzAPI provider. The retry configuration allows you to control how the provider handles failed requests and if it should retry them.
---

## Why retry?

Sometimes, when managing cloud infrastructure, requests to the cloud provider may fail due to transient issues such as network problems, timeouts, eventual consistency, or rate limiting. In these cases, it can be beneficial to retry the request a few times before giving up. The AzAPI provider allows you to configure the retry behavior to suit your needs.

## Retry configuration

There are two types of retry configurations available in the AzAPI provider:

1. Provider retry configuration
2. Resource-specific retry configuration

### Provider retry configuration

The provider retry configuration is a global configuration that applies to all resources managed by the provider. You can configure the provider retry behavior by setting the following provider values:

- `maximum_busy_retry_attempts`

This value controls the number of times the provider will retry a failed request. The default value is `3`.
A retry will be triggered if the request fails with HTTP 408, 429, 500, 502, 503, or 504.

In the case that the response header contains a `Retry-After` value, the provider will wait for the specified duration before retrying the request.

### Resource-specific retry configuration

In addition to the provider retry configuration, you can also configure the retry behavior for individual resources. This allows you to fine-tune the retry behavior for specific resources.

The resource-specific retry comes after the provider retry, that is to say that the provider retry will be attempted first, and if it fails or exceeds the maximum attempts, the resource-specific retry will be attempted.
Note that the resource-specific retry does not honour the `Retry-After` header and is exponential backoff based.

Resource specific retry is configured using the `retry` attribute.

If you configure a retry configuration, the maximum elapsed time for the retry will be set to the resource's timeout value for that operation (create, update, read, delete).

With `azapi_resource` and `azapi_data_plane_resource`, the provider performs a read operation after the resource has been created so that we can store the read-only values.

The schema of these retry attributes is as follows:

- `error_message_regex` - A list of regular expressions to match against error messages. If any of the regular expressions match, the request will be retried.
- `interval_seconds` - The initial number of seconds to wait before the 1st retry. The default value is `10`.
- `max_interval_seconds` - The maximum number of seconds to wait before retrying a request. The default value is `180`.
- `multiplier` - The multiplier to apply to the interval between retries. The default value is `1.5`.
- `randomization_factor` - The randomization factor to apply to the interval between retries. The default value is `0.5`. The formula for the randomized interval is: `RetryInterval * (random value in range [1 - RandomizationFactor, 1 + RandomizationFactor])`. Set to zero `0.0` for no randomization.

## Default resource-specific retry configuration

If you do not configure any retry values, the provider will use the following:

For the initial create/read/update/delete operation we will retry on HTTP 429 status codes, this is a safety net against the provider `maximum_busy_retry_attempts` being exceeded.

For the read-after-create, the provider will retry on HTTP 404 and 403 status codes. This is logical as, if we have just successfully created the resource, we should not be getting a 404 or 403 on any subsequent GET request.

Both of these configurations will have a maximum elapsed time of the resource's timeout value for that operation (create, update, read, delete).
