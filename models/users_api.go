package models

type UserName struct {
	Name string `json:"name" example:"NKTKLN"`
}

type UserUsername struct {
	Username string `json:"username" example:"nktkln"`
}

type UserEmail struct {
	Email string `json:"email" example:"nktkln@example.com"`
}

type UserPassword struct {
	Password string `json:"password" example:"StRon9Pa$$w0rd"`
}

type UserTokens struct {
	AccessToken string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIzMDE5NDgxODQiLCJleHAiOjE2NTU0MTc4MDF9.ZqPxmPK3qV3VrT4D0wbwN2tMGzAhaH5kQnqr8iePTZA"`
	RefreshToken string `json:"refresh_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIzMDE5NDgxODQiLCJleHAiOjE2NTU0MjI1NTh9.S1ypUshOnPB66VJV6RL42cdgYbV8LaGk7zYgL5JlsYg"`
}

type UserLoginData struct {
	Email    string `json:"email" example:"nktkln@example.com"`
	Password string `json:"password" example:"StRon9Pa$$w0rd"`
}

type UserData struct {
	Email    string `json:"email" example:"nktkln@example.com"`
	Password string `json:"password" example:"StRon9Pa$$w0rd"`
	Name     string `json:"name" example:"NKTKLN"`
	Username string `json:"username" example:"nktkln"`
}

type ShowUserData struct {
	Id       int    `json:"id" example:"1023456789"`
	Name     string `json:"name" example:"NKTKLN"`
	Username string `json:"username" example:"nktkln"`
}
