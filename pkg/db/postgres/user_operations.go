package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/NKTKLN/todo-api/models"
	"github.com/NKTKLN/todo-api/pkg/db"
)

func (d *PDB) CrateUser(model models.Users) (userId int, err error) {
	// Creating hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(model.Password), bcrypt.DefaultCost)
	if err != nil {
		return
	}

	// Generating user Id
	userId = int(uuid.New().ID())
	for !d.checkUserId(userId) {
		userId = int(uuid.New().ID())
	}

	// Creating new user
	err = d.DB.Create(&models.Users{Id: userId, Email: model.Email, Password: string(hashedPassword), Name: model.Name, Username: model.Username}).Error
	return
}

func (d *PDB) checkUserId(id int) bool {
	var userData models.Users
	result := d.DB.Table("users").Where("id = ?", id).First(&userData).Error
	return errors.Is(result, gorm.ErrRecordNotFound)
}

func (d *PDB) CheckUserEmail(email string) bool {
	var userData models.Users
	result := d.DB.Table("users").Where("email = ?", email).First(&userData).Error
	return errors.Is(result, gorm.ErrRecordNotFound)
}

func (d *PDB) CheckUserUsername(username string) bool {
	var userData models.Users
	result := d.DB.Table("users").Where("username = ?", username).First(&userData).Error
	return errors.Is(result, gorm.ErrRecordNotFound)
}

func (d *PDB) GetUser(model models.Users) (userData models.Users) {
	d.DB.Where(&model).Take(&userData)
	return
}

func (d *PDB) GetUserByEmail(email string) (userData models.Users) {
	d.DB.Where("email = ?", email).Take(&userData)
	return
}

func (d *PDB) GetUserById(id int) (userData models.Users) {
	d.DB.Where("id = ?", id).Take(&userData)
	return
}

func (d *PDB) UpdateUser(where, model models.Users) error {
	return d.DB.Where(&where).Updates(&model).Error
}

func (d *PDB) UpdateUserPassword(password, email string) (err error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return
	}
	return d.DB.Where("email = ?", email).Updates(models.Users{Password: string(hashedPassword)}).Error
}

func (d *PDB) UpdateUserIcon(id int, icon string) error {
	return d.DB.Table("users").Where("id = ?", id).Update("icon", icon).Error
}

func (d *PDB) CheckUserPassword(password, email string) (err error) {
	var passwordHash string
	err = d.DB.Table("users").Select("password").Where("email = ?", email).Take(&passwordHash).Error
	if err != nil {
		return
	}
	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	return
}

func (d *PDB) DeleteUser(storage db.MinIOClient, ctx context.Context, model models.Users) error {
	// Deleting all user lists
	for _, list := range d.GetAllUserLists(model.Id) {
		if err := d.DeleteList(list.Id); err != nil {
			return err
		}
	}

	// Deleting a user icon
	if model.Icon != "" {
		if err := storage.DeleteFile(ctx, model.Icon); err != nil {
			return err
		}
	}

	// Deleting a user account
	return d.DB.Delete(&models.Users{}, model.Id).Error
}
