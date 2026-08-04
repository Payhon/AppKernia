package httpx

type Success[T any] struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Data      T      `json:"data"`
	RequestID string `json:"request_id"`
}

type ErrorBody struct {
	Code       string         `json:"code"`
	MessageKey string         `json:"message_key"`
	Message    string         `json:"message"`
	Details    map[string]any `json:"details,omitempty"`
}

type Error struct {
	Error     ErrorBody `json:"error"`
	RequestID string    `json:"request_id"`
}
