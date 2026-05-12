package service

import (
	"errors"

	"github.com/govnoeby/67-Ui/v3/database"
	"github.com/govnoeby/67-Ui/v3/database/model"
	"github.com/govnoeby/67-Ui/v3/logger"
	"github.com/govnoeby/67-Ui/v3/util/crypto"
	ldaputil "github.com/govnoeby/67-Ui/v3/util/ldap"
	"github.com/xlzd/gotp"
	"gorm.io/gorm"
)

// UserService provides business logic for user management and authentication.
type UserService struct {
	settingService SettingService
}

func (s *UserService) GetUsers() ([]*model.User, error) {
	db := database.GetDB()
	var users []*model.User
	err := db.Model(model.User{}).Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (s *UserService) GetUser(id int) (*model.User, error) {
	db := database.GetDB()
	user := &model.User{}
	err := db.Model(model.User{}).First(user, id).Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) CreateUser(username, password, email string, role model.Role) (*model.User, error) {
	if username == "" {
		return nil, errors.New("username can not be empty")
	}
	if password == "" {
		return nil, errors.New("password can not be empty")
	}
	if !role.IsValid() {
		return nil, errors.New("invalid role: " + string(role))
	}

	db := database.GetDB()
	var count int64
	db.Model(model.User{}).Where("username = ?", username).Count(&count)
	if count > 0 {
		return nil, errors.New("username already exists")
	}

	hashedPassword, err := crypto.HashPasswordAsBcrypt(password)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Username: username,
		Password: hashedPassword,
		Email:    email,
		Role:     role,
		IsActive: true,
	}
	err = db.Create(user).Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) UpdateUser(id int, username, password, email string, role model.Role, isActive bool) error {
	db := database.GetDB()

	updates := map[string]any{}
	if username != "" {
		updates["username"] = username
	}
	if password != "" {
		hashedPassword, err := crypto.HashPasswordAsBcrypt(password)
		if err != nil {
			return err
		}
		updates["password"] = hashedPassword
	}
	if email != "" {
		updates["email"] = email
	}
	if role != "" {
		if !role.IsValid() {
			return errors.New("invalid role: " + string(role))
		}
		updates["role"] = role
	}
	updates["is_active"] = isActive

	return db.Model(model.User{}).Where("id = ?", id).Updates(updates).Error
}

func (s *UserService) DeleteUser(id int) error {
	db := database.GetDB()
	// Prevent deleting the last admin
	var adminCount int64
	db.Model(model.User{}).Where("role = ?", model.RoleAdmin).Count(&adminCount)
	user, _ := s.GetUser(id)
	if user != nil && user.Role == model.RoleAdmin && adminCount <= 1 {
		return errors.New("cannot delete the last admin user")
	}
	return db.Delete(model.User{}, id).Error
}

func (s *UserService) GetFirstUser() (*model.User, error) {
	db := database.GetDB()

	user := &model.User{}
	err := db.Model(model.User{}).
		Order("id asc").
		First(user).
		Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) CheckUser(username string, password string, twoFactorCode string) (*model.User, error) {
	db := database.GetDB()

	user := &model.User{}

	err := db.Model(model.User{}).
		Where("username = ?", username).
		First(user).
		Error
	if err == gorm.ErrRecordNotFound {
		return nil, errors.New("invalid credentials")
	} else if err != nil {
		logger.Warning("check user err:", err)
		return nil, err
	}

	if !user.IsActive {
		return nil, errors.New("account is disabled")
	}

	if !crypto.CheckPasswordHash(user.Password, password) {
		ldapEnabled, _ := s.settingService.GetLdapEnable()
		if !ldapEnabled {
			return nil, errors.New("invalid credentials")
		}

		host, _ := s.settingService.GetLdapHost()
		port, _ := s.settingService.GetLdapPort()
		useTLS, _ := s.settingService.GetLdapUseTLS()
		bindDN, _ := s.settingService.GetLdapBindDN()
		ldapPass, _ := s.settingService.GetLdapPassword()
		baseDN, _ := s.settingService.GetLdapBaseDN()
		userFilter, _ := s.settingService.GetLdapUserFilter()
		userAttr, _ := s.settingService.GetLdapUserAttr()

		cfg := ldaputil.Config{
			Host:       host,
			Port:       port,
			UseTLS:     useTLS,
			BindDN:     bindDN,
			Password:   ldapPass,
			BaseDN:     baseDN,
			UserFilter: userFilter,
			UserAttr:   userAttr,
		}
		ok, err := ldaputil.AuthenticateUser(cfg, username, password)
		if err != nil || !ok {
			return nil, errors.New("invalid credentials")
		}
	}

	twoFactorEnable, err := s.settingService.GetTwoFactorEnable()
	if err != nil {
		logger.Warning("check two factor err:", err)
		return nil, err
	}

	if twoFactorEnable {
		twoFactorToken, err := s.settingService.GetTwoFactorToken()

		if err != nil {
			logger.Warning("check two factor token err:", err)
			return nil, err
		}

		if gotp.NewDefaultTOTP(twoFactorToken).Now() != twoFactorCode {
			return nil, errors.New("invalid 2fa code")
		}
	}

	return user, nil
}

func (s *UserService) UpdateUserCredentials(id int, username string, password string) error {
	db := database.GetDB()
	hashedPassword, err := crypto.HashPasswordAsBcrypt(password)

	if err != nil {
		return err
	}

	twoFactorEnable, err := s.settingService.GetTwoFactorEnable()
	if err != nil {
		return err
	}

	if twoFactorEnable {
		s.settingService.SetTwoFactorEnable(false)
		s.settingService.SetTwoFactorToken("")
	}

	return db.Model(model.User{}).
		Where("id = ?", id).
		Updates(map[string]any{"username": username, "password": hashedPassword}).
		Error
}

func (s *UserService) UpdateFirstUser(username string, password string) error {
	if username == "" {
		return errors.New("username can not be empty")
	} else if password == "" {
		return errors.New("password can not be empty")
	}
	hashedPassword, er := crypto.HashPasswordAsBcrypt(password)

	if er != nil {
		return er
	}

	db := database.GetDB()
	user := &model.User{}
	err := db.Model(model.User{}).First(user).Error
	if database.IsNotFound(err) {
		user.Username = username
		user.Password = hashedPassword
		return db.Model(model.User{}).Create(user).Error
	} else if err != nil {
		return err
	}
	user.Username = username
	user.Password = hashedPassword
	return db.Save(user).Error
}
