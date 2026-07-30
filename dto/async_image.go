package dto

type AsyncImageTaskResponse struct {
	ID     string               `json:"id"`
	Status string               `json:"status"`
	Output AsyncImageTaskOutput `json:"output"`
	Error  *AsyncImageTaskError `json:"error,omitempty"`
}

type AsyncImageTaskOutput struct {
	Availability string                 `json:"availability"`
	ExpiresAt    int64                  `json:"expires_at,omitempty"`
	Data         []AsyncImageOutputData `json:"data"`
	Error        *AsyncImageTaskError   `json:"error,omitempty"`
}

type AsyncImageOutputData struct {
	Index int    `json:"index"`
	URL   string `json:"url"`
}

type AsyncImageTaskError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
