package retryer

import (
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/request"
)

// Retryable codes specific to ResourceDeleters
var retryableCodes = map[string]struct{}{
	"DependencyViolation": {},
}

// DeleteRetryer configures retrying for all AWS delete requests
type DeleteRetryer struct {
	NumMaxRetries int
}

// RetryRules define how a request is retried upon failure. Uses the
// client.DefaultRetryer.RetryRules function to calculate exponential backoff
func (dr DeleteRetryer) RetryRules(r *request.Request) time.Duration {
	retryer := client.DefaultRetryer{NumMaxRetries: dr.NumMaxRetries}
	return retryer.RetryRules(r)
}

// ShouldRetry returns whether a request should be retried. Requests will be
// retried if the error code is in retryableCodes or is a retryable/throttle
// code
func (dr DeleteRetryer) ShouldRetry(r *request.Request) bool {
	aerr, ok := r.Error.(awserr.Error)
	if (ok && isCodeRetryable(aerr.Code())) || r.HTTPResponse.StatusCode >= 500 {
		return true
	}

	return r.IsErrorRetryable() || r.IsErrorThrottle()
}

func isCodeRetryable(code string) bool {
	_, ok := retryableCodes[code]
	return ok
}

// MaxRetries returns the number of retries the retryer should attempt
func (dr DeleteRetryer) MaxRetries() int {
	return dr.NumMaxRetries
}
