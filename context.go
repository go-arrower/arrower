package arrower

// CTXKey is the type used by all keys put in a context.
// As recommended by the package context, Arrower defines and uses its own data type for keys in the use of WithValue.
type CTXKey string

const (
	// CtxAuthUserID is a definition from the auth Context. It is here, so the postgres handler can access it, until
	// the Context gets finalised and moved to this repo. TODO .
	CtxAuthUserID CTXKey = "auth.user_id"
)
