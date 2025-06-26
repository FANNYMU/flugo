package examples

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type UserService struct {
	users []User
}

func NewUserService() *UserService {
	return &UserService{
		users: []User{
			{ID: 1, Name: "John Doe", Email: "john@example.com"},
			{ID: 2, Name: "Jane Smith", Email: "jane@example.com"},
		},
	}
}

func (s *UserService) GetAll() []User {
	return s.users
}

func (s *UserService) GetByID(id int) *User {
	for _, user := range s.users {
		if user.ID == id {
			return &user
		}
	}
	return nil
}

func (s *UserService) Create(user User) User {
	user.ID = len(s.users) + 1
	s.users = append(s.users, user)
	return user
}

func (s *UserService) Update(user User) User {
	for i, u := range s.users {
		if u.ID == user.ID {
			s.users[i] = user
			return user
		}
	}
	return user
}

func (s *UserService) Delete(id int) bool {
	for i, user := range s.users {
		if user.ID == id {
			s.users = append(s.users[:i], s.users[i+1:]...)
			return true
		}
	}
	return false
}

func (s *UserService) GetByEmail(email string) *User {
	for _, user := range s.users {
		if user.Email == email {
			return &user
		}
	}
	return nil
}
