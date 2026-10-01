package request

type CreateUserRequestDTO struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UpdateUserRequestDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	AccessToken string `json:"access_token,omitempty"`
}
