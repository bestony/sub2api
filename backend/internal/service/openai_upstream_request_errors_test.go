package service

import (
	"net/http"
	"testing"
)

func TestIsOpenAIUpstreamRequestError(t *testing.T) {
	passthrough := []int{
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusRequestEntityTooLarge,
		http.StatusUnprocessableEntity,
	}
	for _, code := range passthrough {
		if !IsOpenAIUpstreamRequestError(code) {
			t.Fatalf("IsOpenAIUpstreamRequestError(%d) = false, want true", code)
		}
	}

	flattened := []int{
		http.StatusUnauthorized,
		http.StatusConflict,
		http.StatusPaymentRequired,
		http.StatusForbidden,
		http.StatusRequestTimeout,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		529,
	}
	for _, code := range flattened {
		if IsOpenAIUpstreamRequestError(code) {
			t.Fatalf("IsOpenAIUpstreamRequestError(%d) = true, want false", code)
		}
	}
}

func TestOpenAIUpstreamRequestErrorType(t *testing.T) {
	if got := OpenAIUpstreamRequestErrorType(http.StatusNotFound); got != "not_found_error" {
		t.Fatalf("OpenAIUpstreamRequestErrorType(404) = %q, want not_found_error", got)
	}
	for _, code := range []int{400, 413, 422} {
		if got := OpenAIUpstreamRequestErrorType(code); got != "invalid_request_error" {
			t.Fatalf("OpenAIUpstreamRequestErrorType(%d) = %q, want invalid_request_error", code, got)
		}
	}
}

func TestOpenAIUpstreamRequestErrorMessage(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want string
	}{
		{
			name: "error.message payload",
			body: []byte(`{"error":{"message":"Invalid schema for response_format 'agentic_plan'"}}`),
			want: "Invalid schema for response_format 'agentic_plan'",
		},
		{
			name: "detail payload",
			body: []byte(`{"detail":"The 'gpt-5.6-sol' model is not supported when using Codex with a ChatGPT account."}`),
			want: "The 'gpt-5.6-sol' model is not supported when using Codex with a ChatGPT account.",
		},
		{
			name: "empty body falls back",
			body: nil,
			want: OpenAIUpstreamRequestErrorFallbackMessage,
		},
		{
			name: "non json body falls back",
			body: []byte(`<html>bad gateway</html>`),
			want: OpenAIUpstreamRequestErrorFallbackMessage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := OpenAIUpstreamRequestErrorMessage(tt.body); got != tt.want {
				t.Fatalf("OpenAIUpstreamRequestErrorMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestShouldPassthroughOpenAIUpstreamRequestError(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   []byte
		want   bool
	}{
		{
			name:   "deterministic schema 400 passes",
			status: http.StatusBadRequest,
			body:   []byte(`{"error":{"message":"Invalid schema for response_format 'agentic_plan'"}}`),
			want:   true,
		},
		{
			name:   "transient server_is_overloaded 400 does not pass",
			status: http.StatusBadRequest,
			body:   []byte(`{"error":{"code":"server_is_overloaded","message":"The server is currently overloaded"}}`),
			want:   false,
		},
		{
			name:   "account-shaped org disabled 400 does not pass",
			status: http.StatusBadRequest,
			body:   []byte(`{"error":{"message":"Your organization has been disabled."}}`),
			want:   false,
		},
		{
			name:   "account-shaped credit balance 400 does not pass",
			status: http.StatusBadRequest,
			body:   []byte(`{"error":{"message":"Your credit balance is too low to access the API."}}`),
			want:   false,
		},
		{
			name:   "non request-shaped status does not pass",
			status: http.StatusUnauthorized,
			body:   []byte(`{"error":{"message":"bad token"}}`),
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldPassthroughOpenAIUpstreamRequestError(tt.status, tt.body); got != tt.want {
				t.Fatalf("ShouldPassthroughOpenAIUpstreamRequestError(%d) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}
