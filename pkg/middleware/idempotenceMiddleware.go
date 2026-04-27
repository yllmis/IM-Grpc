package middleware

import (
	"net/http"

	"github.com/IM_System/pkg/interceptor"
)

type IdempotenceMiddleware struct{}

func NewIdempotenceMiddleware() *IdempotenceMiddleware {
	return &IdempotenceMiddleware{}
}

func (m *IdempotenceMiddleware) Handler(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(interceptor.ContextWithVal(r.Context()))

		next(w, r)
	}
}
