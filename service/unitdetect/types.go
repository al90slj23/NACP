package unitdetect

type Result struct {
	OK      bool   `json:"ok"`
	Type    string `json:"type"`
	BaseURL string `json:"base_url"`
	Message string `json:"message"`
}

type httpResult struct {
	Status  int
	Headers string
	Body    []byte
}

func detected(platformType, baseURL, message string) Result {
	return Result{
		OK:      true,
		Type:    platformType,
		BaseURL: baseURL,
		Message: message,
	}
}

func failed(platformType, baseURL, message string) Result {
	return Result{
		OK:      false,
		Type:    platformType,
		BaseURL: baseURL,
		Message: message,
	}
}
