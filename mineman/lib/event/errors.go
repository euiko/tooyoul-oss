package event

import "errors"

var (
	ErrEventModuleTypeInvalid  = errors.New("event module has an invalid type")
	ErrEventHookNotInitialized = errors.New("event hook not yet initialized")
)
