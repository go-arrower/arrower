package init

type PWConfirmation struct {
	Active  bool
	Timeout int // how long the token is valid
}

type Config struct {
	Mailer                     any // smtp <> local etc.
	UserProvider               any // future music
	PWConfirmation             PWConfirmation
	PwHashCost                 int
	LoginThrottle              int // time in sec until a new login attempt can be made
	InsecureAllowAnyPWStrength bool
	RegisterAllowed            bool // enabled | disabled
	RegisterAdminRoutes        bool
}
