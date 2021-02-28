// +build !nobolt

package dataprovider

import (
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	bolt "go.etcd.io/bbolt"

	"github.com/drakkan/sftpgo/logger"
	"github.com/drakkan/sftpgo/utils"
	"github.com/drakkan/sftpgo/version"
	"github.com/drakkan/sftpgo/vfs"
)

const (
	boltDatabaseVersion = 6
)

var (
	usersBucket     = []byte("users")
	foldersBucket   = []byte("folders")
	adminsBucket    = []byte("admins")
	dbVersionBucket = []byte("db_version")
	dbVersionKey    = []byte("version")
)

// BoltProvider auth provider for bolt key/value store
type BoltProvider struct {
	dbHandle *bolt.DB
}

func init() {
	version.AddFeature("+bolt")
}

func initializeBoltProvider(basePath string) error {
	var err error
	logSender = fmt.Sprintf("dataprovider_%v", BoltDataProviderName)
	dbPath := config.Name
	if !utils.IsFileInputValid(dbPath) {
		return fmt.Errorf("Invalid database path: %#v", dbPath)
	}
	if !filepath.IsAbs(dbPath) {
		dbPath = filepath.Join(basePath, dbPath)
	}
	dbHandle, err := bolt.Open(dbPath, 0600, &bolt.Options{
		NoGrowSync:   false,
		FreelistType: bolt.FreelistArrayType,
		Timeout:      5 * time.Second})
	if err == nil {
		providerLog(logger.LevelDebug, "bolt key store handle created")
		err = dbHandle.Update(func(tx *bolt.Tx) error {
			_, e := tx.CreateBucketIfNotExists(usersBucket)
			return e
		})
		if err != nil {
			providerLog(logger.LevelWarn, "error creating users bucket: %v", err)
			return err
		}
		if err != nil {
			providerLog(logger.LevelWarn, "error creating username idx bucket: %v", err)
			return err
		}
		err = dbHandle.Update(func(tx *bolt.Tx) error {
			_, e := tx.CreateBucketIfNotExists(foldersBucket)
			return e
		})
		if err != nil {
			providerLog(logger.LevelWarn, "error creating folders bucket: %v", err)
			return err
		}
		err = dbHandle.Update(func(tx *bolt.Tx) error {
			_, e := tx.CreateBucketIfNotExists(adminsBucket)
			return e
		})
		if err != nil {
			providerLog(logger.LevelWarn, "error creating admins bucket: %v", err)
			return err
		}
		err = dbHandle.Update(func(tx *bolt.Tx) error {
			_, e := tx.CreateBucketIfNotExists(dbVersionBucket)
			return e
		})
		if err != nil {
			providerLog(logger.LevelWarn, "error creating database version bucket: %v", err)
			return err
		}
		provider = &BoltProvider{dbHandle: dbHandle}
	} else {
		providerLog(logger.LevelWarn, "error creating bolt key/value store handler: %v", err)
	}
	return err
}

func (p *BoltProvider) checkAvailability() error {
	_, err := getBoltDatabaseVersion(p.dbHandle)
	return err
}

func (p *BoltProvider) validateUserAndTLSCert(username, protocol string, tlsCert *x509.Certificate) (User, error) {
	var user User
	if tlsCert == nil {
		return user, errors.New("TLS certificate cannot be null or empty")
	}
	user, err := p.userExists(username)
	if err != nil {
		providerLog(logger.LevelWarn, "error authenticating user %#v: %v", username, err)
		return user, err
	}
	return checkUserAndTLSCertificate(&user, protocol, tlsCert)
}

func (p *BoltProvider) validateUserAndPass(username, password, ip, protocol string) (User, error) {
	var user User
	if password == "" {
		return user, errors.New("Credentials cannot be null or empty")
	}
	user, err := p.userExists(username)
	if err != nil {
		providerLog(logger.LevelWarn, "error authenticating user %#v: %v", username, err)
		return user, err
	}
	return checkUserAndPass(&user, password, ip, protocol)
}

func (p *BoltProvider) validateAdminAndPass(username, password, ip string) (Admin, error) {
	admin, err := p.adminExists(username)
	if err != nil {
		providerLog(logger.LevelWarn, "error authenticating admin %#v: %v", username, err)
		return admin, err
	}
	err = admin.checkUserAndPass(password, ip)
	return admin, err
}

func (p *BoltProvider) validateUserAndPubKey(username string, pubKey []byte) (User, string, error) {
	var user User
	if len(pubKey) == 0 {
		return user, "", errors.New("Credentials cannot be null or empty")
	}
	user, err := p.userExists(username)
	if err != nil {
		providerLog(logger.LevelWarn, "error authenticating user %#v: %v", username, err)
		return user, "", err
	}
	return checkUserAndPubKey(&user, pubKey)
}

func (p *BoltProvider) updateLastLogin(username string) error {
	return p.dbHandle.Update(func(tx *bolt.Tx) error {
		bucket, err := getUsersBucket(tx)
		if err != nil {
			return err
		}
		var u []byte
		if u = bucket.Get([]byte(username)); u == nil {
			return &RecordNotFoundError{err: fmt.Sprintf("username %#v does not exist, unable to update last login", username)}
		}
		var user User
		err = json.Unmarshal(u, &user)
		if err != nil {
			return err
		}
		user.LastLogin = utils.GetTimeAsMsSinceEpoch(time.Now())
		buf, err := json.Marshal(user)
		if err != nil {
			return err
		}
		err = bucket.Put([]byte(username), buf)
		if err == nil {
			providerLog(logger.LevelDebug, "last login updated for user %#v", username)
		} else {
			providerLog(logger.LevelWarn, "error updating last login for user %#v: %v", username, err)
		}
		return err
	})
}

func (p *BoltProvider) updateQuota(username string, filesAdd int, sizeAdd int64, reset bool) error {
	return p.dbHandle.Update(func(tx *bolt.Tx) error {
		bucket, err := getUsersBucket(tx)
		if err != nil {
			return err
		}
		var u []byte
		if u = bucket.Get([]byte(username)); u == nil {
			return &RecordNotFoundError{err: fmt.Sprintf("username %#v does not exist, unable to update quota", username)}
		}
		var user User
		err = json.Unmarshal(u, &user)
		if err != nil {
			return err
		}
		if reset {
			user.UsedQuotaSize = sizeAdd
			user.UsedQuotaFiles = filesAdd
		} else {
			user.UsedQuotaSize += sizeAdd
			user.UsedQuotaFiles += filesAdd
		}
		user.LastQuotaUpdate = utils.GetTimeAsMsSinceEpoch(time.Now())
		buf, err := json.Marshal(user)
		if err != nil {
			return err
		}
		err = bucket.Put([]byte(username), buf)
		providerLog(logger.LevelDebug, "quota updated for user %#v, files increment: %v size increment: %v is reset? %v",
			username, filesAdd, sizeAdd, reset)
		return err
	})
}

func (p *BoltProvider) getUsedQuota(username string) (int, int64, error) {
	user, err := p.userExists(username)
	if err != nil {
		providerLog(logger.LevelWarn, "unable to get quota for user %v error: %v", username, err)
		return 0, 0, err
	}
	return user.UsedQuotaFiles, user.UsedQuotaSize, err
}

func (p *BoltProvider) adminExists(username string) (Admin, error) {
	var admin Admin

	err := p.dbHandle.View(func(tx *bolt.Tx) error {
		bucket, err := getAdminBucket(tx)
		if err != nil {
			return err
		}
		a := bucket.Get([]byte(username))
		if a == nil {
			return &RecordNotFoundError{err: fmt.Sprintf("admin %v does not exist", username)}
		}
		return json.Unmarshal(a, &admin)
	})

	return admin, err
}

func (p *BoltProvider) addAdmin(admin *Admin) error {
	err := admin.validate()
	if err != nil {
		return err
	}
	return p.dbHandle.Update(func(tx *bolt.Tx) error {
		bucket, err := getAdminBucket(tx)
		if err != nil {
			return err
		}
		if a := bucket.Get([]byte(admin.Username)); a != nil {
			return fmt.Errorf("admin %v already exists", admin.Username)
		}
		id, err := bucket.NextSequence()
		if err != nil {
			return err
		}
		admin.ID = int64(id)
		buf, err := json.Marshal(admin)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(admin.Username), buf)
	})
}

func (p *BoltProvider) updateAdmin(admin *Admin) error {
	err := admin.validate()
	if err != nil {
		return err
	}
	return p.dbHandle.Update(func(tx *bolt.Tx) error {
		bucket, err := getAdminBucket(tx)
		if err != nil {
			return err
		}
		var a []byte

		if a = bucket.Get([]byte(admin.Username)); a == nil {
			return &RecordNotFoundError{err: fmt.Sprintf("admin %v does not exist", admin.Username)}
		}
		var oldAdmin Admin
		err = json.Unmarshal(a, &oldAdmin)
		if err != nil {
			return err
		}

		admin.ID = oldAdmin.ID
		buf, err := json.Marshal(admin)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(admin.Username), buf)
	})
}

func (p *BoltProvider) deleteAdmin(admin *Admin) error {
	return p.dbHandle.Update(func(tx *bolt.Tx) error {
		bucket, err := getAdminBucket(tx)
		if err != nil {
			return err
		}

		if bucket.Get([]byte(admin.Username)) == nil {
			return &RecordNotFoundError{err: fmt.Sprintf("admin %v does not exist", admin.Username)}
		}

		return bucket.Delete([]byte(admin.Username))
	})
}

func (p *BoltProvider) getAdmins(limit int, offset int, order string) ([]Admin, error) {
	admins := make([]Admin, 0, limit)

	err := p.dbHandle.View(func(tx *bolt.Tx) error {
		bucket, err := getAdminBucket(tx)
		if err != nil {
			return err
		}
		cursor := bucket.Cursor()
		itNum := 0
		if order == OrderASC {
			for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
				itNum++
				if itNum <= offset {
					continue
				}
				var admin Admin
				err = json.Unmarshal(v, &admin)
				if err != nil {
					return err
				}
				admin.HideConfidentialData()
				admins = append(admins, admin)
				if len(admins) >= limit {
					break
				}
			}
		} else {
			for k, v := cursor.Last(); k != nil; k, v = cursor.Prev() {
				itNum++
				if itNum <= offset {
					continue
				}
				var admin Admin
				err = json.Unmarshal(v, &admin)
				if err != nil {
					return err
				}
				admin.HideConfidentialData()
				admins = append(admins, admin)
				if len(admins) >= limit {
					break
				}
			}
		}
		return err
	})

	return admins, err
}

func (p *BoltProvider) dumpAdmins() ([]Admin, error) {
	admins := make([]Admin, 0, 30)
	err := p.dbHandle.View(func(tx *bolt.Tx) error {
		bucket, err := getAdminBucket(tx)
		if err != nil {
			return err
		}

		cursor := bucket.Cursor()
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			var admin Admin
			err = json.Unmarshal(v, &admin)
			if err != nil {
				return err
			}
			admins = append(admins, admin)
		}
		return err
	})

	return admins, err
}

func (p *BoltProvider) userExists(username string) (User, error) {
	var user User
	err := p.dbHandle.View(func(tx *bolt.Tx) error {
		bucket, err := getUsersBucket(tx)
		if err != nil {
			return err
		}
		u := bucket.Get([]byte(username))
		if u == nil {
			return &RecordNotFoundError{err: fmt.Sprintf("username %v does not exist", username)}
		}
		folderBucket, err := getFolderBucket(tx)
		if err != nil {
			return err
		}
		user, err = joinUserAndFolders(u, folderBucket)
		return err
	})
	return user, err
}

func (p *BoltProvider) addUser(user *User) error {
	err := ValidateUser(user)
	if err != nil {
		return err
	}
	return p.dbHandle.Update(func(tx *bolt.Tx) error {
		bucket, err := getUsersBucket(tx)
		if err != nil {
			return err
		}
		folderBucket, err := getFolderBucket(tx)
		if err != nil {
			return err
		}
		if u := bucket.Get([]byte(user.Username)); u != nil {
			return fmt.Errorf("username %v already exists", user.Username)
		}
		id, err := bucket.NextSequence()
		if err != nil {
			return err
		}
		user.ID = int64(id)
		user.LastQuotaUpdate = 0
		user.UsedQuotaSize = 0
		user.UsedQuotaFiles = 0
		user.LastLogin = 0
		for _, folder := range user.VirtualFolders {
			err = addUserToFolderMapping(folder, user, folderBucket)
			if err != nil {
				return err
			}
		}
		buf, err := json.Marshal(user)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(user.Username), buf)
	})
}

func (p *BoltProvider) updateUser(user *User) error {
	err := ValidateUser(user)
	if err != nil {
		return err
	}
	return p.dbHandle.Update(func(tx *bolt.Tx) error {
		bucket, err := getUsersBucket(tx)
		if err != nil {
			return err
		}
		folderBucket, err := getFolderBucket(tx)
		if err != nil {
			return err
		}
		var u []byte
		if u = bucket.Get([]byte(user.Username)); u == nil {
			return &RecordNotFoundError{err: fmt.Sprintf("username %v does not exist", user.Username)}
		}
		var oldUser User
		err = json.Unmarshal(u, &oldUser)
		if err != nil {
			return err
		}
		for _, folder := range oldUser.VirtualFolders {
			err = removeUserFromFolderMapping(folder, &oldUser, folderBucket)
			if err != nil {
				return err
			}
		}
		for _, folder := range user.VirtualFolders {
			err = addUserToFolderMapping(folder, user, folderBucket)
			if err != nil {
				return err
			}
		}
		user.ID = oldUser.ID
		user.LastQuotaUpdate = oldUser.LastQuotaUpdate
		user.UsedQuotaSize = oldUser.UsedQuotaSize
		user.UsedQuotaFiles = oldUser.UsedQuotaFiles
		user.LastLogin = oldUser.LastLogin
		buf, err := json.Marshal(user)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(user.Username), buf)
	})
}

func (p *BoltProvider) deleteUser(user *User) error {
	return p.dbHandle.Update(func(tx *bolt.Tx) error {
		bucket, err := getUsersBucket(tx)
		if err != nil {
			return err
		}
		if len(user.VirtualFolders) > 0 {
			folderBucket, err := getFolderBucket(tx)
			if err != nil {
				return err
			}
			for _, folder := range user.VirtualFolders {
				err = removeUserFromFolderMapping(folder, user, folderBucket)
				if err != nil {
					return err
				}
			}
		}
		exists := bucket.Get([]byte(user.Username))
		if exists == nil {
			return &RecordNotFoundError{err: fmt.Sprintf("user %#v does not exist", user.Username)}
		}
		return bucket.Delete([]byte(user.Username))
	})
}

func (p *BoltProvider) dumpUsers() ([]User, error) {
	users := make([]User, 0, 100)
	err := p.dbHandle.View(func(tx *bolt.Tx) error {
		bucket, err := getUsersBucket(tx)
		if err != nil {
			return err
		}
		folderBucket, err := getFolderBucket(tx)
		if err != nil {
			return err
		}
		cursor := bucket.Cursor()
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			user, err := joinUserAndFolders(v, folderBucket)
			if err != nil {
				return err
			}
			err = addCredentialsToUser(&user)
			if err != nil {
				return err
			}
			users = append(users, user)
		}
		return err
	})
	return users, err
}

func (p *BoltProvider) getUsers(limit int, offset int, order string) ([]User, error) {
	users := make([]User, 0, limit)
	var err error
	if limit <= 0 {
		return users, err
	}
	err = p.dbHandle.View(func(tx *bolt.Tx) error {
		bucket, err := getUsersBucket(tx)
		if err != nil {
			return err
		}
		folderBucket, err := getFolderBucket(tx)
		if err != nil {
			return err
		}
		cursor := bucket.Cursor()
		itNum := 0
		if order == OrderASC {
			for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
				itNum++
				if itNum <= offset {
					continue
				}
				user, err := joinUserAndFolders(v, folderBucket)
				if err == nil {
					user.HideConfidentialData()
					users = append(users, user)
				}
				if len(users) >= limit {
					break
				}
			}
		} else {
			for k, v := cursor.Last(); k != nil; k, v = cursor.Prev() {
				itNum++
				if itNum <= offset {
					continue
				}
				user, err := joinUserAndFolders(v, folderBucket)
				if err == nil {
					user.HideConfidentialData()
					users = append(users, user)
				}
				if len(users) >= limit {
					break
				}
			}
		}
		return err
	})
	return users, err
}

func (p *BoltProvider) dumpFolders() ([]vfs.BaseVirtualFolder, error) {
	folders := make([]vfs.BaseVirtualFolder, 0, 50)
	err := p.dbHandle.View(func(tx *bolt.Tx) error {
		bucket, err := getFolderBucket(tx)
		if err != nil {
			return err
		}
		cursor := bucket.Cursor()
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			var folder vfs.BaseVirtualFolder
			err = json.Unmarshal(v, &folder)
			if err != nil {
				return err
			}
			folders = append(folders, folder)
		}
		return err
	})
	return folders, err
}

func (p *BoltProvider) getFolders(limit, offset int, order string) ([]vfs.BaseVirtualFolder, error) {
	folders := make([]vfs.BaseVirtualFolder, 0, limit)
	var err error
	if limit <= 0 {
		return folders, err
	}
	err = p.dbHandle.View(func(tx *bolt.Tx) error {
		bucket, err := getFolderBucket(tx)
		if err != nil {
			return err
		}
		cursor := bucket.Cursor()
		itNum := 0
		if order == OrderASC {
			for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
				itNum++
				if itNum <= offset {
					continue
				}
				var folder vfs.BaseVirtualFolder
				err = json.Unmarshal(v, &folder)
				if err != nil {
					return err
				}
				folders = append(folders, folder)
				if len(folders) >= limit {
					break
				}
			}
		} else {
			for k, v := cursor.Last(); k != nil; k, v = cursor.Prev() {
				itNum++
				if itNum <= offset {
					continue
				}
				var folder vfs.BaseVirtualFolder
				err = json.Unmarshal(v, &folder)
				if err != nil {
					return err
				}
				folders = append(folders, folder)
				if len(folders) >= limit {
					break
				}
			}
		}
		return err
	})
	return folders, err
}

func (p *BoltProvider) getFolderByName(name string) (vfs.BaseVirtualFolder, error) {
	var folder vfs.BaseVirtualFolder
	err := p.dbHandle.View(func(tx *bolt.Tx) error {
		bucket, err := getFolderBucket(tx)
		if err != nil {
			return err
		}
		folder, err = folderExistsInternal(name, bucket)
		return err
	})
	return folder, err
}

func (p *BoltProvider) addFolder(folder *vfs.BaseVirtualFolder) error {
	err := ValidateFolder(folder)
	if err != nil {
		return err
	}
	return p.dbHandle.Update(func(tx *bolt.Tx) error {
		bucket, err := getFolderBucket(tx)
		if err != nil {
			return err
		}
		if f := bucket.Get([]byte(folder.Name)); f != nil {
			return fmt.Errorf("folder %v already exists", folder.Name)
		}
		folder.Users = nil
		_, err = addFolderInternal(*folder, bucket)
		return err
	})
}

func (p *BoltProvider) updateFolder(folder *vfs.BaseVirtualFolder) error {
	err := ValidateFolder(folder)
	if err != nil {
		return err
	}
	return p.dbHandle.Update(func(tx *bolt.Tx) error {
		bucket, err := getFolderBucket(tx)
		if err != nil {
			return err
		}
		var f []byte

		if f = bucket.Get([]byte(folder.Name)); f == nil {
			return &RecordNotFoundError{err: fmt.Sprintf("folder %v does not exist", folder.Name)}
		}
		var oldFolder vfs.BaseVirtualFolder
		err = json.Unmarshal(f, &oldFolder)
		if err != nil {
			return err
		}

		folder.ID = oldFolder.ID
		folder.LastQuotaUpdate = oldFolder.LastQuotaUpdate
		folder.UsedQuotaFiles = oldFolder.UsedQuotaFiles
		folder.UsedQuotaSize = oldFolder.UsedQuotaSize
		folder.Users = oldFolder.Users
		buf, err := json.Marshal(folder)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(folder.Name), buf)
	})
}

func (p *BoltProvider) deleteFolder(folder *vfs.BaseVirtualFolder) error {
	return p.dbHandle.Update(func(tx *bolt.Tx) error {
		bucket, err := getFolderBucket(tx)
		if err != nil {
			return err
		}
		usersBucket, err := getUsersBucket(tx)
		if err != nil {
			return err
		}
		var f []byte
		if f = bucket.Get([]byte(folder.Name)); f == nil {
			return &RecordNotFoundError{err: fmt.Sprintf("folder %v does not exist", folder.Name)}
		}
		var folder vfs.BaseVirtualFolder
		err = json.Unmarshal(f, &folder)
		if err != nil {
			return err
		}
		for _, username := range folder.Users {
			var u []byte
			if u = usersBucket.Get([]byte(username)); u == nil {
				continue
			}
			var user User
			err = json.Unmarshal(u, &user)
			if err != nil {
				return err
			}
			var folders []vfs.VirtualFolder
			for _, userFolder := range user.VirtualFolders {
				if folder.Name != userFolder.Name {
					folders = append(folders, userFolder)
				}
			}
			user.VirtualFolders = folders
			buf, err := json.Marshal(user)
			if err != nil {
				return err
			}
			err = usersBucket.Put([]byte(user.Username), buf)
			if err != nil {
				return err
			}
		}

		return bucket.Delete([]byte(folder.Name))
	})
}

func (p *BoltProvider) updateFolderQuota(name string, filesAdd int, sizeAdd int64, reset bool) error {
	return p.dbHandle.Update(func(tx *bolt.Tx) error {
		bucket, err := getFolderBucket(tx)
		if err != nil {
			return err
		}
		var f []byte
		if f = bucket.Get([]byte(name)); f == nil {
			return &RecordNotFoundError{err: fmt.Sprintf("folder %#v does not exist, unable to update quota", name)}
		}
		var folder vfs.BaseVirtualFolder
		err = json.Unmarshal(f, &folder)
		if err != nil {
			return err
		}
		if reset {
			folder.UsedQuotaSize = sizeAdd
			folder.UsedQuotaFiles = filesAdd
		} else {
			folder.UsedQuotaSize += sizeAdd
			folder.UsedQuotaFiles += filesAdd
		}
		folder.LastQuotaUpdate = utils.GetTimeAsMsSinceEpoch(time.Now())
		buf, err := json.Marshal(folder)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(folder.Name), buf)
	})
}

func (p *BoltProvider) getUsedFolderQuota(name string) (int, int64, error) {
	folder, err := p.getFolderByName(name)
	if err != nil {
		providerLog(logger.LevelWarn, "unable to get quota for folder %#v error: %v", name, err)
		return 0, 0, err
	}
	return folder.UsedQuotaFiles, folder.UsedQuotaSize, err
}

func (p *BoltProvider) close() error {
	return p.dbHandle.Close()
}

func (p *BoltProvider) reloadConfig() error {
	return nil
}

// initializeDatabase does nothing, no initilization is needed for bolt provider
func (p *BoltProvider) initializeDatabase() error {
	return ErrNoInitRequired
}

func (p *BoltProvider) migrateDatabase() error {
	dbVersion, err := getBoltDatabaseVersion(p.dbHandle)
	if err != nil {
		return err
	}
	switch version := dbVersion.Version; {
	case version == boltDatabaseVersion:
		providerLog(logger.LevelDebug, "bolt database is up to date, current version: %v", version)
		return ErrNoInitRequired
	case version < 6:
		err = fmt.Errorf("database version %v is too old, please see the upgrading docs", version)
		providerLog(logger.LevelError, "%v", err)
		logger.ErrorToConsole("%v", err)
		return err
	default:
		if version > boltDatabaseVersion {
			providerLog(logger.LevelWarn, "database version %v is newer than the supported one: %v", version,
				boltDatabaseVersion)
			logger.WarnToConsole("database version %v is newer than the supported one: %v", version,
				boltDatabaseVersion)
			return nil
		}
		return fmt.Errorf("Database version not handled: %v", version)
	}
}

func (p *BoltProvider) revertDatabase(targetVersion int) error {
	dbVersion, err := getBoltDatabaseVersion(p.dbHandle)
	if err != nil {
		return err
	}
	if targetVersion >= 8 {
		targetVersion = 6
	}
	if dbVersion.Version == targetVersion {
		return errors.New("current version match target version, nothing to do")
	}
	return errors.New("the current version cannot be reverted")
}

func joinUserAndFolders(u []byte, foldersBucket *bolt.Bucket) (User, error) {
	var user User
	err := json.Unmarshal(u, &user)
	if err != nil {
		return user, err
	}
	if len(user.VirtualFolders) > 0 {
		var folders []vfs.VirtualFolder
		for _, folder := range user.VirtualFolders {
			baseFolder, err := folderExistsInternal(folder.Name, foldersBucket)
			if err != nil {
				continue
			}
			folder.MappedPath = baseFolder.MappedPath
			folder.UsedQuotaFiles = baseFolder.UsedQuotaFiles
			folder.UsedQuotaSize = baseFolder.UsedQuotaSize
			folder.LastQuotaUpdate = baseFolder.LastQuotaUpdate
			folder.ID = baseFolder.ID
			folders = append(folders, folder)
		}
		user.VirtualFolders = folders
	}
	user.SetEmptySecretsIfNil()
	return user, err
}

func folderExistsInternal(name string, bucket *bolt.Bucket) (vfs.BaseVirtualFolder, error) {
	var folder vfs.BaseVirtualFolder
	f := bucket.Get([]byte(name))
	if f == nil {
		err := &RecordNotFoundError{err: fmt.Sprintf("folder %v does not exist", name)}
		return folder, err
	}
	err := json.Unmarshal(f, &folder)
	return folder, err
}

func addFolderInternal(folder vfs.BaseVirtualFolder, bucket *bolt.Bucket) (vfs.BaseVirtualFolder, error) {
	id, err := bucket.NextSequence()
	if err != nil {
		return folder, err
	}
	folder.ID = int64(id)
	buf, err := json.Marshal(folder)
	if err != nil {
		return folder, err
	}
	err = bucket.Put([]byte(folder.Name), buf)
	return folder, err
}

func addUserToFolderMapping(folder vfs.VirtualFolder, user *User, bucket *bolt.Bucket) error {
	var baseFolder vfs.BaseVirtualFolder
	var err error
	if f := bucket.Get([]byte(folder.Name)); f == nil {
		// folder does not exists, try to create
		folder.LastQuotaUpdate = 0
		folder.UsedQuotaFiles = 0
		folder.UsedQuotaSize = 0
		baseFolder, err = addFolderInternal(folder.BaseVirtualFolder, bucket)
	} else {
		err = json.Unmarshal(f, &baseFolder)
	}
	if err != nil {
		return err
	}
	if !utils.IsStringInSlice(user.Username, baseFolder.Users) {
		baseFolder.Users = append(baseFolder.Users, user.Username)
		buf, err := json.Marshal(baseFolder)
		if err != nil {
			return err
		}
		err = bucket.Put([]byte(folder.Name), buf)
		if err != nil {
			return err
		}
	}
	return err
}

func removeUserFromFolderMapping(folder vfs.VirtualFolder, user *User, bucket *bolt.Bucket) error {
	var f []byte
	if f = bucket.Get([]byte(folder.Name)); f == nil {
		// the folder does not exists so there is no associated user
		return nil
	}
	var baseFolder vfs.BaseVirtualFolder
	err := json.Unmarshal(f, &baseFolder)
	if err != nil {
		return err
	}
	if utils.IsStringInSlice(user.Username, baseFolder.Users) {
		var newUserMapping []string
		for _, u := range baseFolder.Users {
			if u != user.Username {
				newUserMapping = append(newUserMapping, u)
			}
		}
		baseFolder.Users = newUserMapping
		buf, err := json.Marshal(baseFolder)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(folder.Name), buf)
	}
	return err
}

func getAdminBucket(tx *bolt.Tx) (*bolt.Bucket, error) {
	var err error

	bucket := tx.Bucket(adminsBucket)
	if bucket == nil {
		err = errors.New("unable to find admin bucket, bolt database structure not correcly defined")
	}
	return bucket, err
}

func getUsersBucket(tx *bolt.Tx) (*bolt.Bucket, error) {
	var err error
	bucket := tx.Bucket(usersBucket)
	if bucket == nil {
		err = errors.New("unable to find required buckets, bolt database structure not correcly defined")
	}
	return bucket, err
}

func getFolderBucket(tx *bolt.Tx) (*bolt.Bucket, error) {
	var err error
	bucket := tx.Bucket(foldersBucket)
	if bucket == nil {
		err = fmt.Errorf("unable to find required buckets, bolt database structure not correcly defined")
	}
	return bucket, err
}

func getBoltDatabaseVersion(dbHandle *bolt.DB) (schemaVersion, error) {
	var dbVersion schemaVersion
	err := dbHandle.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(dbVersionBucket)
		if bucket == nil {
			return fmt.Errorf("unable to find database version bucket")
		}
		v := bucket.Get(dbVersionKey)
		if v == nil {
			dbVersion = schemaVersion{
				Version: 6,
			}
			return nil
		}
		return json.Unmarshal(v, &dbVersion)
	})
	return dbVersion, err
}

/*func updateBoltDatabaseVersion(dbHandle *bolt.DB, version int) error {
	err := dbHandle.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(dbVersionBucket)
		if bucket == nil {
			return fmt.Errorf("unable to find database version bucket")
		}
		newDbVersion := schemaVersion{
			Version: version,
		}
		buf, err := json.Marshal(newDbVersion)
		if err != nil {
			return err
		}
		return bucket.Put(dbVersionKey, buf)
	})
	return err
}*/
