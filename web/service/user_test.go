package service

import (
	"errors"
	"sort"
	"strings"
	"testing"

	"github.com/superaddmin/SuperXray-gui/v2/database/model"
	"github.com/superaddmin/SuperXray-gui/v2/util/crypto"
	"gorm.io/gorm"
)

func TestUserServiceUsesRepositoryBoundary(t *testing.T) {
	passwordHash, err := crypto.HashPasswordAsBcrypt("secret")
	if err != nil {
		t.Fatalf("HashPasswordAsBcrypt failed: %v", err)
	}

	userRepo := newFakeUserRepository()
	userRepo.users[1] = &model.User{
		Id:       1,
		Username: "admin",
		Password: passwordHash,
	}

	settingRepo := newFakeSettingRepository()
	settingRepo.settings["twoFactorEnable"] = &model.Setting{Key: "twoFactorEnable", Value: "false"}
	userSvc := NewUserService(userRepo, *NewSettingService(settingRepo))

	firstUser, err := userSvc.GetFirstUser()
	if err != nil {
		t.Fatalf("GetFirstUser through repository returned error: %v", err)
	}
	if firstUser.Username != "admin" {
		t.Fatalf("GetFirstUser Username = %q, want admin", firstUser.Username)
	}

	checkedUser, err := userSvc.CheckUser("admin", "secret", "")
	if err != nil {
		t.Fatalf("CheckUser through repository returned error: %v", err)
	}
	if checkedUser.Id != 1 {
		t.Fatalf("CheckUser returned user id = %d, want 1", checkedUser.Id)
	}

	if err := userSvc.UpdateUser(1, "root", "changed"); err != nil {
		t.Fatalf("UpdateUser through repository returned error: %v", err)
	}
	if !userRepo.updateCredentialsCalled {
		t.Fatal("UpdateUser did not call repository UpdateCredentials")
	}
	if userRepo.users[1].Username != "root" {
		t.Fatalf("UpdateUser persisted username = %q, want root", userRepo.users[1].Username)
	}
	if !crypto.CheckPasswordHash(userRepo.users[1].Password, "changed") {
		t.Fatal("UpdateUser did not persist a bcrypt hash for the new password")
	}

	if err := userSvc.UpdateFirstUser("owner", "new-secret"); err != nil {
		t.Fatalf("UpdateFirstUser through repository returned error: %v", err)
	}
	if !userRepo.saveCalled {
		t.Fatal("UpdateFirstUser did not call repository Save")
	}
	if userRepo.users[1].Username != "owner" {
		t.Fatalf("UpdateFirstUser persisted username = %q, want owner", userRepo.users[1].Username)
	}
	if !crypto.CheckPasswordHash(userRepo.users[1].Password, "new-secret") {
		t.Fatal("UpdateFirstUser did not persist a bcrypt hash for the new password")
	}
}

func TestAccountWritesValidateUsernameBeforeChangingCredentialsOrTOTP(t *testing.T) {
	users := newFakeUserRepository()
	settings := NewSettingService(newFakeSettingRepository())
	svc := NewUserService(users, *settings)
	if err := settings.SetTwoFactorEnable(true); err != nil {
		t.Fatal(err)
	}
	limitName := strings.Repeat("é", MaxUsernameBytes/2)
	if err := svc.UpdateFirstUser(limitName, "fixture"); err != nil {
		t.Fatal(err)
	}
	for _, update := range []func(string, string) error{
		svc.UpdateFirstUser,
		func(name, password string) error { return svc.UpdateUser(1, name, password) },
	} {
		if err := update(limitName+"a", "changed"); err == nil {
			t.Fatal("accepted an oversized username")
		}
		if users.users[1].Username != limitName || !crypto.CheckPasswordHash(users.users[1].Password, "fixture") {
			t.Fatal("rejected update changed credentials")
		}
		if enabled, err := settings.GetTwoFactorEnable(); err != nil || !enabled {
			t.Fatal("rejected update changed two-factor settings")
		}
	}
}

type fakeUserRepository struct {
	users                   map[int]*model.User
	saveCalled              bool
	updateCredentialsCalled bool
}

func newFakeUserRepository() *fakeUserRepository {
	return &fakeUserRepository{
		users: make(map[int]*model.User),
	}
}

func (r *fakeUserRepository) First() (*model.User, error) {
	if len(r.users) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	ids := make([]int, 0, len(r.users))
	for id := range r.users {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return copyUser(r.users[ids[0]]), nil
}

func (r *fakeUserRepository) FindByUsername(username string) (*model.User, error) {
	for _, user := range r.users {
		if user.Username == username {
			return copyUser(user), nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeUserRepository) Save(user *model.User) error {
	if user == nil {
		return errors.New("user is nil")
	}
	r.saveCalled = true
	if user.Id == 0 {
		user.Id = len(r.users) + 1
	}
	r.users[user.Id] = copyUser(user)
	return nil
}

func (r *fakeUserRepository) UpdateCredentials(id int, username string, password string) error {
	user, ok := r.users[id]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	r.updateCredentialsCalled = true
	user.Username = username
	user.Password = password
	return nil
}

func copyUser(user *model.User) *model.User {
	copy := *user
	return &copy
}
