package service

import (
	"net/http"
	"strings"
)

// OpenAIUpstreamRequestErrorFallbackMessage is returned to the client when a
// request-shaped upstream 4xx carries no extractable message.
const OpenAIUpstreamRequestErrorFallbackMessage = "Upstream rejected the request"

// IsOpenAIUpstreamRequestError reports whether an upstream status code
// describes a problem with the request itself, as opposed to gateway account
// credentials (401/402/403), upstream throttling (429/529) or upstream
// availability (5xx). These rejections are deterministic — replaying the same
// request cannot succeed — so the gateway must pass the status through instead
// of flattening it into 502: clients such as Codex CLI auto-retry 5xx with
// backoff, which turns one deterministic 4xx into a sustained retry storm and
// hides the actionable upstream message (invalid schema, context window
// exceeded, plan-gated model, unknown previous_response_id, ...).
func IsOpenAIUpstreamRequestError(statusCode int) bool {
	// 409 is deliberately excluded: OpenAI uses it for time-dependent
	// conflicts (e.g. a response still in progress) where retrying later can
	// succeed, so it stays on the retryable 502 mapping.
	switch statusCode {
	case http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusRequestEntityTooLarge,
		http.StatusUnprocessableEntity:
		return true
	}
	return false
}

// OpenAIUpstreamRequestErrorType maps a request-shaped upstream status to the
// error type expected by strict OpenAI clients, mirroring the Chat Completions
// compat path (handleCompatErrorResponse).
func OpenAIUpstreamRequestErrorType(statusCode int) string {
	if statusCode == http.StatusNotFound {
		return "not_found_error"
	}
	return "invalid_request_error"
}

// OpenAIUpstreamRequestErrorMessage extracts the sanitized upstream message of
// a request-shaped 4xx so the client can see why its request was rejected,
// falling back to a generic message when the body has none.
func OpenAIUpstreamRequestErrorMessage(body []byte) string {
	msg := strings.TrimSpace(extractUpstreamErrorMessage(body))
	msg = sanitizeUpstreamErrorMessage(msg)
	if msg == "" {
		return OpenAIUpstreamRequestErrorFallbackMessage
	}
	return msg
}

// openAIAccountShapedBadRequestMarkers match 400 bodies that describe the
// gateway account's state (org disabled, billing, KYC) rather than the
// client's request. See RateLimitService.HandleUpstreamError, which disables
// the account on the same markers.
var openAIAccountShapedBadRequestMarkers = []string{
	"organization has been disabled",
	"credit balance",
	"identity verification is required",
}

// ShouldPassthroughOpenAIUpstreamRequestError decides whether an upstream
// error response should be passed through to the client as a request-shaped
// 4xx. Two classes must stay on the opaque, retryable 502 mapping even though
// they arrive with a request-shaped status:
//   - transient processing failures ("server_is_overloaded", "selected model
//     is at capacity", ...), which OpenAI ships on 400/503 and which a client
//     retry can genuinely resolve;
//   - account-shaped 400s (org disabled, billing, KYC), which describe the
//     gateway's account, not the client's request, and must not leak.
func ShouldPassthroughOpenAIUpstreamRequestError(statusCode int, body []byte) bool {
	if !IsOpenAIUpstreamRequestError(statusCode) {
		return false
	}
	msg := strings.TrimSpace(extractUpstreamErrorMessage(body))
	if isOpenAITransientProcessingError(statusCode, msg, body) {
		return false
	}
	lower := strings.ToLower(msg)
	for _, marker := range openAIAccountShapedBadRequestMarkers {
		if strings.Contains(lower, marker) {
			return false
		}
	}
	return true
}
