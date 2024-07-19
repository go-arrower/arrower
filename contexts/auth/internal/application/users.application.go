package application

import "github.com/go-arrower/arrower/app"

type UserApplication struct {
	RegisterUser app.Request[RegisterUserRequest, RegisterUserResponse]
	LoginUser    app.Request[LoginUserRequest, LoginUserResponse]
	ListUsers    app.Query[ListUsersQuery, ListUsersResponse]
	ShowUser     app.Query[ShowUserQuery, ShowUserResponse]
	NewUser      app.Command[NewUserCommand]
	VerifyUser   app.Command[VerifyUserCommand]
	BlockUser    app.Request[BlockUserRequest, BlockUserResponse]     // todo refactor to a command
	UnblockUser  app.Request[UnblockUserRequest, UnblockUserResponse] // todo refactor to a command
}
