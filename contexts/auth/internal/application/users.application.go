package application

import "github.com/go-arrower/arrower/app"

type UserApplication struct {
	LoginUser app.Request[LoginUserRequest, LoginUserResponse]
	ListUsers app.Query[ListUsersQuery, ListUsersResponse]
}
