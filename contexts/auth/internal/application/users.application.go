package application

import "github.com/go-arrower/arrower/app"

type UserApplication struct {
	ListUsers app.Query[ListUsersQuery, ListUsersResponse]
}
