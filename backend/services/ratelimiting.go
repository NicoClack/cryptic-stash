package services

import (
	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/ratelimiting"
	"github.com/NicoClack/cryptic-stash/backend/ratelimiting/definitions"
)

type RateLimiter struct {
	App     *common.App
	Limiter *ratelimiting.Limiter
}

func NewRateLimiter(app *common.App) *RateLimiter {
	limiter := ratelimiting.NewLimiter(app)
	definitions.Register(limiter.Group(""))
	return &RateLimiter{
		App:     app,
		Limiter: limiter,
	}
}

func (service *RateLimiter) RequestSession(
	eventName string, amount int, identifier string,
) (common.LimiterSession, common.WrappedError) {
	session, wrappedErr := service.Limiter.RequestSession(eventName, amount, identifier)
	if session == nil { // Avoid wrapping nil sessions in a non-nil interface
		return nil, wrappedErr
	}
	return session, wrappedErr
}
func (service *RateLimiter) DeleteInactiveUsers() {
	service.Limiter.DeleteInactiveUsers()
}
