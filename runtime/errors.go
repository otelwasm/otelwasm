package runtime

import "errors"

// Common errors used across runtime implementations
var (
	ErrRuntimeNotFound           = errors.New("runtime not found")
	ErrModuleCompileFailed       = errors.New("module compilation failed")
	ErrModuleInstantiateFailed   = errors.New("module instantiation failed")
	ErrFunctionNotExported       = errors.New("function not exported")
	ErrInvalidConfiguration      = errors.New("invalid configuration")
	ErrMemoryExportNotFound      = errors.New("memory export not found")
	ErrHostFunctionNotFound      = errors.New("host function not found")
	ErrUnsupportedRuntimeType    = errors.New("unsupported runtime type")
)