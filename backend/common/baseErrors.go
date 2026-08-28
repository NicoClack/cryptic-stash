// Errors that are usually wrapped into WrappedErrors
// These are similar to the errors normally returned by 3rd party libraries,
// they just sometimes need some abstraction or aren't defined
package common

import (
	"fmt"
	"net/http"
)

const (
	ErrTypeREST = "REST"
)

// Note: NewWrappedRESTError should normally be used instead
var ErrWrapperREST = NewErrorWrapper(ErrTypeREST, ErrTypeAPI)

type RESTError struct {
	Response *http.Response
}

func (restError *RESTError) Error() string {
	return fmt.Sprintf(
		"%s %s failed with status %s",
		restError.Response.Request.Method,
		restError.Response.Request.URL.String(),
		restError.Response.Status,
	)
}

func NewWrappedRESTError(response *http.Response) WrappedError {
	return ErrWrapperREST.Wrap(&RESTError{Response: response})
}
