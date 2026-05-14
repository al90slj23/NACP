package service

import (
	"context"
	"fmt"

	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

const statusClientClosedRequest = 499

const contextKeyBillingSkippedRequestCanceled = "billing_skipped_request_canceled"

func RequestContextErr(c *gin.Context) error {
	if c == nil || c.Request == nil {
		return nil
	}
	return c.Request.Context().Err()
}

func NewRequestCanceledError(c *gin.Context) *types.NewAPIError {
	err := RequestContextErr(c)
	if err == nil {
		err = context.Canceled
	}
	return types.NewErrorWithStatusCode(
		fmt.Errorf("client request closed: %w", err),
		types.ErrorCodeDoRequestFailed,
		statusClientClosedRequest,
		types.ErrOptionWithSkipRetry(),
	)
}

func NewRequestCanceledErrorIfDone(c *gin.Context) *types.NewAPIError {
	if RequestContextErr(c) == nil {
		return nil
	}
	return NewRequestCanceledError(c)
}

func MarkBillingSkippedRequestCanceled(c *gin.Context) {
	if c == nil {
		return
	}
	c.Set(contextKeyBillingSkippedRequestCanceled, true)
}

func WasBillingSkippedRequestCanceled(c *gin.Context) bool {
	if c == nil {
		return false
	}
	v, ok := c.Get(contextKeyBillingSkippedRequestCanceled)
	if !ok {
		return false
	}
	skipped, ok := v.(bool)
	return ok && skipped
}
