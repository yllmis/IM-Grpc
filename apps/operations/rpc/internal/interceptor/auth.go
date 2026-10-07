package interceptor

import "github.com/IM_System/pkg/serviceauth"

// Compatibility aliases keep the existing OperationsQuery auth contract.
const MetadataServiceToken = serviceauth.MetadataServiceToken

type Auth = serviceauth.Auth

var NewAuth = serviceauth.NewAuth
