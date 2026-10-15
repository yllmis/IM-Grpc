package serviceauth

import (
	"context"
	"fmt"
	"os"
	"strings"

	"google.golang.org/grpc"
)

// BusinessTokenEnvironment is for trusted IM backend applications, not users
// or read-only query clients. Keep it out of public configuration centers.
const BusinessTokenEnvironment = "IM_BUSINESS_RPC_TOKEN"

func BusinessConfig() (Config, error) {
	token := strings.TrimSpace(os.Getenv(BusinessTokenEnvironment))
	if token == "" {
		return Config{}, fmt.Errorf("%s is required for business RPC access", BusinessTokenEnvironment)
	}
	return Config{Enable: true, Token: token}, nil
}

// RequireSeparateCredentials prevents a read-only query credential from also
// granting access to the legacy business RPCs (which include writes).
func RequireSeparateCredentials(business, query Config) error {
	if !query.Enable {
		return nil
	}
	for _, q := range append([]string{query.Token}, query.ExtraTokens...) {
		for _, b := range append([]string{business.Token}, business.ExtraTokens...) {
			if strings.TrimSpace(q) != "" && strings.TrimSpace(q) == strings.TrimSpace(b) {
				return fmt.Errorf("business and read-query credentials must be different")
			}
		}
	}
	return nil
}

func MustBusinessClientInterceptor() grpc.UnaryClientInterceptor {
	c, err := BusinessConfig()
	if err != nil {
		panic(err)
	}
	return BusinessClientInterceptor(c.Token)
}

// Only the original backend RPC services receive this privileged credential.
// Never attach it to a query, third-party or unknown RPC method.
func BusinessClientInterceptor(token string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		for _, service := range []string{"im.Im", "user.User", "social.social"} {
			if strings.HasPrefix(method, "/"+service+"/") {
				ctx = Outgoing(ctx, token)
				break
			}
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
