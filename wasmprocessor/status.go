package wasmprocessor

type StatusCode int32

const (
	StatusCodeSuccess StatusCode = iota
	StatusCodeError
)

var statusCodeToString = map[StatusCode]string{
	StatusCodeSuccess: "Success",
	StatusCodeError:   "Error",
}

func (s StatusCode) String() string {
	if str, ok := statusCodeToString[s]; ok {
		return str
	}
	return "Unknown"
}
