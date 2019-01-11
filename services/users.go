package services

import (
	"errors"
	"github.com/jinzhu/gorm"
	"github.com/pufferpanel/pufferpanel/database"
	"github.com/pufferpanel/pufferpanel/models"
	"golang.org/x/crypto/bcrypt"
	"strconv"
	"strings"
)

type UserService interface {
	Get(username string) (*models.User, bool, error)

	GetByEmail(email string) (*models.User, bool, error)

	Update(model *models.User) error

	Delete(username string) error

	Create(user *models.User) error

	ChangePassword(username string, newPass string) error

	Search(usernameFilter, emailFilter string, pageSize, page uint) (*models.Users, error)

	Login(email string, password string) (sessionToken string, err error)
}

type userService struct {
	db *gorm.DB
}

func GetUserService() (UserService, error) {
	db, err := database.GetConnection()
	if err != nil {
		return nil, err
	}

	service := &userService{
		db: db,
	}

	return service, nil
}

func (us *userService) Get(username string) (*models.User, bool, error) {
	model := &models.User{
		Username: username,
	}

	res := us.db.Where(model).FirstOrInit(model)

	return model, model.ID != 0, res.Error
}

func (us *userService) Login(email string, password string) (sessionToken string, err error) {
	oauth2, err := GetOAuthService()

	if err != nil {
		return
	}

	model := &models.User{
		Email: email,
	}

	err = us.db.Where(model).First(model).Error

	if err != nil && !gorm.IsRecordNotFoundError(err) {
		return
	}

	if model.ID == 0 || gorm.IsRecordNotFoundError(err) {
		err = errors.New("invalid credentials")
		return
	}

	providedPw := []byte(password)
	correctPw := []byte(model.HashedPassword)

	if bcrypt.CompareHashAndPassword(correctPw, providedPw) != nil {
		err = errors.New("invalid credentials")
		return
	}

	sessionToken, err = oauth2.CreateSession(model)
	return
}

func (us *userService) GetByEmail(email string) (*models.User, bool, error) {
	model := &models.User{
		Email: email,
	}

	res := us.db.Where(model).FirstOrInit(model)

	return model, model.ID != 0, res.Error
}

func (us *userService) Update(model *models.User) error {
	res := us.db.Save(model)
	return res.Error
}

func (us *userService) Delete(username string) error {
	model := &models.User{
		Username: username,
	}

	res := us.db.Delete(model)
	return res.Error
}

func (us *userService) Create(user *models.User) error {
	oauth, err := GetOAuthService()

	if err != nil {
		return err
	}

	res := us.db.Create(user)
	if res.Error != nil {
		return res.Error
	}

	name := ".internal_" + strconv.Itoa(int(user.ID))

	_, err = oauth.Create(user, nil, name, true, "login")

	if err != nil {
		us.db.Delete(user)
	}

	return err
}

func (us *userService) ChangePassword(username string, newPass string) error {
	user, exists, err := us.Get(username)

	if err != nil {
		return err
	}

	if !exists {
		return errors.New("no such user")
	}

	err = user.SetPassword(newPass)
	if err != nil {
		return err
	}
	return us.Update(user)
}

func (us *userService) Search(usernameFilter, emailFilter string, pageSize, page uint) (*models.Users, error) {
	users := &models.Users{}

	query := us.db.Offset((page - 1) * pageSize).Limit(pageSize)

	usernameFilter = strings.Replace(usernameFilter, "*", "%", -1)
	emailFilter = strings.Replace(emailFilter, "*", "%", -1)

	if usernameFilter != "" && usernameFilter != "%" {
		query = query.Where("username LIKE ?", usernameFilter)
	}

	if emailFilter != "" && emailFilter != "%" {
		query = query.Where("email LIKE ?", emailFilter)
	}

	res := query.Find(users)

	return users, res.Error
}