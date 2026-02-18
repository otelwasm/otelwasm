package wasmplugin

import "errors"

var (
	ErrRequiredFunctionNotExported = errors.New("required function not exported")
	ErrABIVersionMarkerNotExported = errors.New("required ABI version marker not exported")
)
