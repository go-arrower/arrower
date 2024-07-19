package application

import "github.com/go-arrower/arrower/app"

type UserApplication struct {
	RegisterUser app.Request[RegisterUserRequest, RegisterUserResponse]
	LoginUser    app.Request[LoginUserRequest, LoginUserResponse]
	ListUsers    app.Query[ListUsersQuery, ListUsersResponse]
	ShowUser     app.Query[ShowUserQuery, ShowUserResponse]
	NewUser      app.Command[NewUserCommand]
	VerifyUser   app.Command[VerifyUserCommand]
}
