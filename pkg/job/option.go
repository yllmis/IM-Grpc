package job

import "time"

type (
	RetryOption func(opts *retryOptions)

	retryOptions struct {
		timeout     time.Duration
		retryNums   int
		isRetryFunc IsRetryFunc
		retryJetLag RetryJetLagFunc
	}
)

func newOptions(opts ...RetryOption) *retryOptions {
	opt := &retryOptions{
		timeout:     DefaultRetryTimeout,
		retryNums:   DefaultRetryNums,
		isRetryFunc: RetryAlways,
		retryJetLag: RetryJetLagAlways,
	}

	for _, option := range opts {
		option(opt)
	}

	return opt
}

func WithRetryNums(retryNums int) RetryOption {
	return func(opts *retryOptions) {
		opts.retryNums = 1
		if retryNums > 1 {
			opts.retryNums = retryNums
		}
	}
}

func WithRetryTime(timeout time.Duration) RetryOption {
	return func(opts *retryOptions) {
		if timeout > 0 {
			opts.timeout = timeout
		}
	}
}

func WithIsRetryFunc(isRetryFunc IsRetryFunc) RetryOption {
	return func(opts *retryOptions) {
		if isRetryFunc != nil {
			opts.isRetryFunc = isRetryFunc
		}
	}
}

func WithRetryJetLagFunc(retryJetLagFunc RetryJetLagFunc) RetryOption {
	return func(opts *retryOptions) {
		if retryJetLagFunc != nil {
			opts.retryJetLag = retryJetLagFunc
		}
	}
}
