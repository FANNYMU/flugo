package examples

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"flugo.com/auth"
	"flugo.com/cache"
	"flugo.com/dto"
	"flugo.com/response"
	"flugo.com/upload"
)

type CreateUserDTO struct {
	Name     string `json:"name" required:"true" min_length:"2" max_length:"50" alpha:"true"`
	Email    string `json:"email" required:"true" email:"true"`
	Phone    string `json:"phone,omitempty" phone:"true"`
	Age      int    `json:"age,omitempty" min:"18" max:"120"`
	Website  string `json:"website,omitempty" url:"true"`
	Password string `json:"password" required:"true" min_length:"8"`
}

type UpdateUserDTO struct {
	Name    string `json:"name,omitempty" min_length:"2" max_length:"50" alpha:"true"`
	Email   string `json:"email,omitempty" email:"true"`
	Phone   string `json:"phone,omitempty" phone:"true"`
	Age     int    `json:"age,omitempty" min:"18" max:"120"`
	Website string `json:"website,omitempty" url:"true"`
}

type LoginDTO struct {
	Email    string `json:"email" required:"true" email:"true"`
	Password string `json:"password" required:"true"`
}

type UserController struct {
	UserService *UserService `inject:"true"`
}

func NewUserController() *UserController {
	return &UserController{}
}

func (c *UserController) GetUsers(w http.ResponseWriter, r *http.Request) {
	cacheKey := "users:all"

	if cachedUsers, found := cache.Get(cacheKey); found {
		response.Success(w, cachedUsers, "Users retrieved from cache")
		return
	}

	users := c.UserService.GetAll()
	cache.Set(cacheKey, users, 5*time.Minute)

	response.Success(w, users, "Users retrieved successfully")
}

func (c *UserController) GetUsersById(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 3 {
		response.BadRequest(w, "ID parameter required")
		return
	}

	id, err := strconv.Atoi(pathParts[2])
	if err != nil {
		response.BadRequest(w, "Invalid ID parameter")
		return
	}

	cacheKey := "user:" + strconv.Itoa(id)
	if cachedUser, found := cache.Get(cacheKey); found {
		response.Success(w, cachedUser, "User retrieved from cache")
		return
	}

	user := c.UserService.GetByID(id)
	if user == nil {
		response.NotFound(w, "User not found")
		return
	}

	cache.Set(cacheKey, user, 10*time.Minute)
	response.Success(w, user, "User retrieved successfully")
}

func (c *UserController) PostUsers(w http.ResponseWriter, r *http.Request) {
	var createUserDTO CreateUserDTO

	if !dto.BindAndRespond(w, r, &createUserDTO) {
		return
	}

	user := User{
		Name:  createUserDTO.Name,
		Email: createUserDTO.Email,
	}

	createdUser := c.UserService.Create(user)
	cache.Delete("users:all")

	response.Created(w, createdUser, "User created successfully")
}

func (c *UserController) PutUsersById(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 3 {
		response.BadRequest(w, "ID parameter required")
		return
	}

	id, err := strconv.Atoi(pathParts[2])
	if err != nil {
		response.BadRequest(w, "Invalid ID parameter")
		return
	}

	var updateUserDTO UpdateUserDTO
	if !dto.BindAndRespond(w, r, &updateUserDTO) {
		return
	}

	user := c.UserService.GetByID(id)
	if user == nil {
		response.NotFound(w, "User not found")
		return
	}

	if updateUserDTO.Name != "" {
		user.Name = updateUserDTO.Name
	}
	if updateUserDTO.Email != "" {
		user.Email = updateUserDTO.Email
	}

	updatedUser := c.UserService.Update(*user)
	cache.Delete("users:all")
	cache.Delete("user:" + strconv.Itoa(id))

	response.Updated(w, updatedUser, "User updated successfully")
}

func (c *UserController) DeleteUsersById(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 3 {
		response.BadRequest(w, "ID parameter required")
		return
	}

	id, err := strconv.Atoi(pathParts[2])
	if err != nil {
		response.BadRequest(w, "Invalid ID parameter")
		return
	}

	if !c.UserService.Delete(id) {
		response.NotFound(w, "User not found")
		return
	}

	cache.Delete("users:all")
	cache.Delete("user:" + strconv.Itoa(id))

	response.Deleted(w, "User deleted successfully")
}

func (c *UserController) PostLogin(w http.ResponseWriter, r *http.Request) {
	var loginDTO LoginDTO
	if !dto.BindAndRespond(w, r, &loginDTO) {
		return
	}

	user := c.UserService.GetByEmail(loginDTO.Email)
	if user == nil {
		response.Unauthorized(w, "Invalid credentials")
		return
	}

	claims := auth.Claims{
		UserID:   user.ID,
		Username: user.Name,
		Email:    user.Email,
		Roles:    []string{"user"},
		Extra: map[string]interface{}{
			"last_login": time.Now(),
		},
	}

	token, err := auth.GenerateToken(claims)
	if err != nil {
		response.InternalError(w, "Failed to generate token")
		return
	}

	response.Success(w, token, "Login successful")
}

func (c *UserController) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetCurrentUserID(r)
	if userID == 0 {
		response.Unauthorized(w, "User not authenticated")
		return
	}

	user := c.UserService.GetByID(userID)
	if user == nil {
		response.NotFound(w, "User not found")
		return
	}

	response.Success(w, user, "Profile retrieved successfully")
}

func (c *UserController) PostUploadAvatar(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetCurrentUserID(r)
	if userID == 0 {
		response.Unauthorized(w, "User not authenticated")
		return
	}

	uploadResult, err := upload.HandleUpload(r, "avatar")
	if err != nil {
		response.BadRequest(w, "Upload failed", err.Error())
		return
	}

	response.Success(w, uploadResult, "Avatar uploaded successfully")
}
