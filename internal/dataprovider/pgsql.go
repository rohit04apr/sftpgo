// Copyright (C) 2019-2023 Nicola Murino
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, version 3.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

//go:build !nopgsql
// +build !nopgsql

package dataprovider

import (
	"context"
	"crypto/x509"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/drakkan/sftpgo/v2/internal/logger"
	"github.com/drakkan/sftpgo/v2/internal/version"
	"github.com/drakkan/sftpgo/v2/internal/vfs"
)

const (
	pgsqlResetSQL = `DROP TABLE IF EXISTS "{{api_keys}}" CASCADE;
DROP TABLE IF EXISTS "{{folders_mapping}}" CASCADE;
DROP TABLE IF EXISTS "{{users_folders_mapping}}" CASCADE;
DROP TABLE IF EXISTS "{{users_groups_mapping}}" CASCADE;
DROP TABLE IF EXISTS "{{admins_groups_mapping}}" CASCADE;
DROP TABLE IF EXISTS "{{groups_folders_mapping}}" CASCADE;
DROP TABLE IF EXISTS "{{admins}}" CASCADE;
DROP TABLE IF EXISTS "{{folders}}" CASCADE;
DROP TABLE IF EXISTS "{{shares}}" CASCADE;
DROP TABLE IF EXISTS "{{users}}" CASCADE;
DROP TABLE IF EXISTS "{{groups}}" CASCADE;
DROP TABLE IF EXISTS "{{defender_events}}" CASCADE;
DROP TABLE IF EXISTS "{{defender_hosts}}" CASCADE;
DROP TABLE IF EXISTS "{{active_transfers}}" CASCADE;
DROP TABLE IF EXISTS "{{shared_sessions}}" CASCADE;
DROP TABLE IF EXISTS "{{rules_actions_mapping}}" CASCADE;
DROP TABLE IF EXISTS "{{events_actions}}" CASCADE;
DROP TABLE IF EXISTS "{{events_rules}}" CASCADE;
DROP TABLE IF EXISTS "{{tasks}}" CASCADE;
DROP TABLE IF EXISTS "{{nodes}}" CASCADE;
DROP TABLE IF EXISTS "{{roles}}" CASCADE;
DROP TABLE IF EXISTS "{{ip_lists}}" CASCADE;
DROP TABLE IF EXISTS "{{configs}}" CASCADE;
DROP TABLE IF EXISTS "{{schema_version}}" CASCADE;
`
	pgsqlInitial = `CREATE TABLE "{{schema_version}}" ("id" serial NOT NULL PRIMARY KEY, "version" integer NOT NULL);
CREATE TABLE "{{admins}}" ("id" serial NOT NULL PRIMARY KEY, "username" varchar(255) NOT NULL UNIQUE,
"description" varchar(512) NULL, "password" varchar(255) NOT NULL, "email" varchar(255) NULL, "status" integer NOT NULL,
"permissions" text NOT NULL, "filters" text NULL, "additional_info" text NULL, "last_login" bigint NOT NULL,
"created_at" bigint NOT NULL, "updated_at" bigint NOT NULL);
CREATE TABLE "{{active_transfers}}" ("id" bigserial NOT NULL PRIMARY KEY, "connection_id" varchar(100) NOT NULL,
"transfer_id" bigint NOT NULL, "transfer_type" integer NOT NULL, "username" varchar(255) NOT NULL,
"folder_name" varchar(255) NULL, "ip" varchar(50) NOT NULL, "truncated_size" bigint NOT NULL,
"current_ul_size" bigint NOT NULL, "current_dl_size" bigint NOT NULL, "created_at" bigint NOT NULL,
"updated_at" bigint NOT NULL);
CREATE TABLE "{{defender_hosts}}" ("id" bigserial NOT NULL PRIMARY KEY, "ip" varchar(50) NOT NULL UNIQUE,
"ban_time" bigint NOT NULL, "updated_at" bigint NOT NULL);
CREATE TABLE "{{defender_events}}" ("id" bigserial NOT NULL PRIMARY KEY, "date_time" bigint NOT NULL, "score" integer NOT NULL,
"host_id" bigint NOT NULL);
ALTER TABLE "{{defender_events}}" ADD CONSTRAINT "{{prefix}}defender_events_host_id_fk_defender_hosts_id" FOREIGN KEY
("host_id") REFERENCES "{{defender_hosts}}" ("id") MATCH SIMPLE ON UPDATE NO ACTION ON DELETE CASCADE;
CREATE TABLE "{{folders}}" ("id" serial NOT NULL PRIMARY KEY, "name" varchar(255) NOT NULL UNIQUE, "description" varchar(512) NULL,
"path" text NULL, "used_quota_size" bigint NOT NULL, "used_quota_files" integer NOT NULL, "last_quota_update" bigint NOT NULL,
"filesystem" text NULL);
CREATE TABLE "{{groups}}" ("id" serial NOT NULL PRIMARY KEY, "name" varchar(255) NOT NULL UNIQUE,
"description" varchar(512) NULL, "created_at" bigint NOT NULL, "updated_at" bigint NOT NULL, "user_settings" text NULL);
CREATE TABLE "{{shared_sessions}}" ("key" varchar(128) NOT NULL PRIMARY KEY,
"data" text NOT NULL, "type" integer NOT NULL, "timestamp" bigint NOT NULL);
CREATE TABLE "{{users}}" ("id" serial NOT NULL PRIMARY KEY, "username" varchar(255) NOT NULL UNIQUE, "status" integer NOT NULL,
"expiration_date" bigint NOT NULL, "description" varchar(512) NULL, "password" text NULL, "public_keys" text NULL,
"home_dir" text NOT NULL, "uid" bigint NOT NULL, "gid" bigint NOT NULL, "max_sessions" integer NOT NULL,
"quota_size" bigint NOT NULL, "quota_files" integer NOT NULL, "permissions" text NOT NULL, "used_quota_size" bigint NOT NULL,
"used_quota_files" integer NOT NULL, "last_quota_update" bigint NOT NULL, "upload_bandwidth" integer NOT NULL,
"download_bandwidth" integer NOT NULL, "last_login" bigint NOT NULL, "filters" text NULL, "filesystem" text NULL,
"additional_info" text NULL, "created_at" bigint NOT NULL, "updated_at" bigint NOT NULL, "email" varchar(255) NULL,
"upload_data_transfer" integer NOT NULL, "download_data_transfer" integer NOT NULL, "total_data_transfer" integer NOT NULL,
"used_upload_data_transfer" integer NOT NULL, "used_download_data_transfer" integer NOT NULL, "deleted_at" bigint NOT NULL,
"first_download" bigint NOT NULL, "first_upload" bigint NOT NULL);
CREATE TABLE "{{groups_folders_mapping}}" ("id" serial NOT NULL PRIMARY KEY, "group_id" integer NOT NULL,
"folder_id" integer NOT NULL, "virtual_path" text NOT NULL, "quota_size" bigint NOT NULL, "quota_files" integer NOT NULL);
CREATE TABLE "{{users_groups_mapping}}" ("id" serial NOT NULL PRIMARY KEY, "user_id" integer NOT NULL,
"group_id" integer NOT NULL, "group_type" integer NOT NULL);
CREATE TABLE "{{users_folders_mapping}}" ("id" serial NOT NULL PRIMARY KEY, "virtual_path" text NOT NULL,
"quota_size" bigint NOT NULL, "quota_files" integer NOT NULL, "folder_id" integer NOT NULL, "user_id" integer NOT NULL);
ALTER TABLE "{{users_folders_mapping}}" ADD CONSTRAINT "{{prefix}}unique_user_folder_mapping" UNIQUE ("user_id", "folder_id");
ALTER TABLE "{{users_folders_mapping}}" ADD CONSTRAINT "{{prefix}}users_folders_mapping_folder_id_fk_folders_id"
FOREIGN KEY ("folder_id") REFERENCES "{{folders}}" ("id") MATCH SIMPLE ON UPDATE NO ACTION ON DELETE CASCADE;
ALTER TABLE "{{users_folders_mapping}}" ADD CONSTRAINT "{{prefix}}users_folders_mapping_user_id_fk_users_id"
FOREIGN KEY ("user_id") REFERENCES "{{users}}" ("id") MATCH SIMPLE ON UPDATE NO ACTION ON DELETE CASCADE;
CREATE TABLE "{{shares}}" ("id" serial NOT NULL PRIMARY KEY,
"share_id" varchar(60) NOT NULL UNIQUE, "name" varchar(255) NOT NULL, "description" varchar(512) NULL,
"scope" integer NOT NULL, "paths" text NOT NULL, "created_at" bigint NOT NULL, "updated_at" bigint NOT NULL,
"last_use_at" bigint NOT NULL, "expires_at" bigint NOT NULL, "password" text NULL,
"max_tokens" integer NOT NULL, "used_tokens" integer NOT NULL, "allow_from" text NULL,
"user_id" integer NOT NULL);
ALTER TABLE "{{shares}}" ADD CONSTRAINT "{{prefix}}shares_user_id_fk_users_id" FOREIGN KEY ("user_id")
REFERENCES "{{users}}" ("id") MATCH SIMPLE ON UPDATE NO ACTION ON DELETE CASCADE;
CREATE TABLE "{{api_keys}}" ("id" serial NOT NULL PRIMARY KEY, "name" varchar(255) NOT NULL,
"key_id" varchar(50) NOT NULL UNIQUE, "api_key" varchar(255) NOT NULL UNIQUE, "scope" integer NOT NULL,
"created_at" bigint NOT NULL, "updated_at" bigint NOT NULL, "last_use_at" bigint NOT NULL,"expires_at" bigint NOT NULL,
"description" text NULL, "admin_id" integer NULL, "user_id" integer NULL);
ALTER TABLE "{{api_keys}}" ADD CONSTRAINT "{{prefix}}api_keys_admin_id_fk_admins_id" FOREIGN KEY ("admin_id")
REFERENCES "{{admins}}" ("id") MATCH SIMPLE ON UPDATE NO ACTION ON DELETE CASCADE;
ALTER TABLE "{{api_keys}}" ADD CONSTRAINT "{{prefix}}api_keys_user_id_fk_users_id" FOREIGN KEY ("user_id")
REFERENCES "{{users}}" ("id") MATCH SIMPLE ON UPDATE NO ACTION ON DELETE CASCADE;
ALTER TABLE "{{users_groups_mapping}}" ADD CONSTRAINT "{{prefix}}unique_user_group_mapping" UNIQUE ("user_id", "group_id");
ALTER TABLE "{{groups_folders_mapping}}" ADD CONSTRAINT "{{prefix}}unique_group_folder_mapping" UNIQUE ("group_id", "folder_id");
CREATE INDEX "{{prefix}}users_groups_mapping_group_id_idx" ON "{{users_groups_mapping}}" ("group_id");
ALTER TABLE "{{users_groups_mapping}}" ADD CONSTRAINT "{{prefix}}users_groups_mapping_group_id_fk_groups_id"
FOREIGN KEY ("group_id") REFERENCES "{{groups}}" ("id") MATCH SIMPLE ON UPDATE NO ACTION ON DELETE NO ACTION;
CREATE INDEX "{{prefix}}users_groups_mapping_user_id_idx" ON "{{users_groups_mapping}}" ("user_id");
ALTER TABLE "{{users_groups_mapping}}" ADD CONSTRAINT "{{prefix}}users_groups_mapping_user_id_fk_users_id"
FOREIGN KEY ("user_id") REFERENCES "{{users}}" ("id") MATCH SIMPLE ON UPDATE NO ACTION ON DELETE CASCADE;
CREATE INDEX "{{prefix}}groups_folders_mapping_folder_id_idx" ON "{{groups_folders_mapping}}" ("folder_id");
ALTER TABLE "{{groups_folders_mapping}}" ADD CONSTRAINT "{{prefix}}groups_folders_mapping_folder_id_fk_folders_id"
FOREIGN KEY ("folder_id") REFERENCES "{{folders}}" ("id") MATCH SIMPLE ON UPDATE NO ACTION ON DELETE CASCADE;
CREATE INDEX "{{prefix}}groups_folders_mapping_group_id_idx" ON "{{groups_folders_mapping}}" ("group_id");
ALTER TABLE "{{groups_folders_mapping}}" ADD CONSTRAINT "{{prefix}}groups_folders_mapping_group_id_fk_groups_id"
FOREIGN KEY ("group_id") REFERENCES "{{groups}}" ("id") MATCH SIMPLE ON UPDATE NO ACTION ON DELETE CASCADE;
CREATE TABLE "{{events_rules}}" ("id" serial NOT NULL PRIMARY KEY, "name" varchar(255) NOT NULL UNIQUE,
"description" varchar(512) NULL, "created_at" bigint NOT NULL, "updated_at" bigint NOT NULL, "trigger" integer NOT NULL,
"conditions" text NOT NULL, "deleted_at" bigint NOT NULL);
CREATE TABLE "{{events_actions}}" ("id" serial NOT NULL PRIMARY KEY, "name" varchar(255) NOT NULL UNIQUE,
"description" varchar(512) NULL, "type" integer NOT NULL, "options" text NOT NULL);
CREATE TABLE "{{rules_actions_mapping}}" ("id" serial NOT NULL PRIMARY KEY, "rule_id" integer NOT NULL,
"action_id" integer NOT NULL, "order" integer NOT NULL, "options" text NOT NULL);
CREATE TABLE "{{tasks}}" ("id" serial NOT NULL PRIMARY KEY, "name" varchar(255) NOT NULL UNIQUE, "updated_at" bigint NOT NULL,
"version" bigint NOT NULL);
ALTER TABLE "{{rules_actions_mapping}}" ADD CONSTRAINT "{{prefix}}unique_rule_action_mapping" UNIQUE ("rule_id", "action_id");
ALTER TABLE "{{rules_actions_mapping}}" ADD CONSTRAINT "{{prefix}}rules_actions_mapping_rule_id_fk_events_rules_id"
FOREIGN KEY ("rule_id") REFERENCES "{{events_rules}}" ("id") MATCH SIMPLE ON UPDATE NO ACTION ON DELETE CASCADE;
ALTER TABLE "{{rules_actions_mapping}}" ADD CONSTRAINT "{{prefix}}rules_actions_mapping_action_id_fk_events_targets_id"
FOREIGN KEY ("action_id") REFERENCES "{{events_actions}}" ("id") MATCH SIMPLE ON UPDATE NO ACTION ON DELETE NO ACTION;
CREATE TABLE "{{admins_groups_mapping}}" ("id" serial NOT NULL PRIMARY KEY,
"admin_id" integer NOT NULL, "group_id" integer NOT NULL, "options" text NOT NULL);
ALTER TABLE "{{admins_groups_mapping}}" ADD CONSTRAINT "{{prefix}}unique_admin_group_mapping" UNIQUE ("admin_id", "group_id");
ALTER TABLE "{{admins_groups_mapping}}" ADD CONSTRAINT "{{prefix}}admins_groups_mapping_admin_id_fk_admins_id"
FOREIGN KEY ("admin_id") REFERENCES "{{admins}}" ("id") MATCH SIMPLE ON UPDATE NO ACTION ON DELETE CASCADE;
ALTER TABLE "{{admins_groups_mapping}}" ADD CONSTRAINT "{{prefix}}admins_groups_mapping_group_id_fk_groups_id"
FOREIGN KEY ("group_id") REFERENCES "{{groups}}" ("id") MATCH SIMPLE ON UPDATE NO ACTION ON DELETE CASCADE;
CREATE TABLE "{{nodes}}" ("id" serial NOT NULL PRIMARY KEY, "name" varchar(255) NOT NULL UNIQUE,
"data" text NOT NULL, "created_at" bigint NOT NULL, "updated_at" bigint NOT NULL);
CREATE INDEX "{{prefix}}users_folders_mapping_folder_id_idx" ON "{{users_folders_mapping}}" ("folder_id");
CREATE INDEX "{{prefix}}users_folders_mapping_user_id_idx" ON "{{users_folders_mapping}}" ("user_id");
CREATE INDEX "{{prefix}}api_keys_admin_id_idx" ON "{{api_keys}}" ("admin_id");
CREATE INDEX "{{prefix}}api_keys_user_id_idx" ON "{{api_keys}}" ("user_id");
CREATE INDEX "{{prefix}}users_updated_at_idx" ON "{{users}}" ("updated_at");
CREATE INDEX "{{prefix}}users_deleted_at_idx" ON "{{users}}" ("deleted_at");
CREATE INDEX "{{prefix}}shares_user_id_idx" ON "{{shares}}" ("user_id");
CREATE INDEX "{{prefix}}defender_hosts_updated_at_idx" ON "{{defender_hosts}}" ("updated_at");
CREATE INDEX "{{prefix}}defender_hosts_ban_time_idx" ON "{{defender_hosts}}" ("ban_time");
CREATE INDEX "{{prefix}}defender_events_date_time_idx" ON "{{defender_events}}" ("date_time");
CREATE INDEX "{{prefix}}defender_events_host_id_idx" ON "{{defender_events}}" ("host_id");
CREATE INDEX "{{prefix}}active_transfers_connection_id_idx" ON "{{active_transfers}}" ("connection_id");
CREATE INDEX "{{prefix}}active_transfers_transfer_id_idx" ON "{{active_transfers}}" ("transfer_id");
CREATE INDEX "{{prefix}}active_transfers_updated_at_idx" ON "{{active_transfers}}" ("updated_at");
CREATE INDEX "{{prefix}}shared_sessions_type_idx" ON "{{shared_sessions}}" ("type");
CREATE INDEX "{{prefix}}shared_sessions_timestamp_idx" ON "{{shared_sessions}}" ("timestamp");
CREATE INDEX "{{prefix}}events_rules_updated_at_idx" ON "{{events_rules}}" ("updated_at");
CREATE INDEX "{{prefix}}events_rules_deleted_at_idx" ON "{{events_rules}}" ("deleted_at");
CREATE INDEX "{{prefix}}events_rules_trigger_idx" ON "{{events_rules}}" ("trigger");
CREATE INDEX "{{prefix}}rules_actions_mapping_rule_id_idx" ON "{{rules_actions_mapping}}" ("rule_id");
CREATE INDEX "{{prefix}}rules_actions_mapping_action_id_idx" ON "{{rules_actions_mapping}}" ("action_id");
CREATE INDEX "{{prefix}}rules_actions_mapping_order_idx" ON "{{rules_actions_mapping}}" ("order");
CREATE INDEX "{{prefix}}admins_groups_mapping_admin_id_idx" ON "{{admins_groups_mapping}}" ("admin_id");
CREATE INDEX "{{prefix}}admins_groups_mapping_group_id_idx" ON "{{admins_groups_mapping}}" ("group_id");
INSERT INTO {{schema_version}} (version) VALUES (23);
`
	pgsqlV24SQL = `CREATE TABLE "{{roles}}" ("id" serial NOT NULL PRIMARY KEY, "name" varchar(255) NOT NULL UNIQUE,
"description" varchar(512) NULL, "created_at" bigint NOT NULL, "updated_at" bigint NOT NULL);
ALTER TABLE "{{admins}}" ADD COLUMN "role_id" integer NULL CONSTRAINT "{{prefix}}admins_role_id_fk_roles_id"
REFERENCES "{{roles}}"("id") ON DELETE NO ACTION;
ALTER TABLE "{{users}}" ADD COLUMN "role_id" integer NULL CONSTRAINT "{{prefix}}users_role_id_fk_roles_id"
REFERENCES "{{roles}}"("id") ON DELETE SET NULL;
CREATE INDEX "{{prefix}}admins_role_id_idx" ON "{{admins}}" ("role_id");
CREATE INDEX "{{prefix}}users_role_id_idx" ON "{{users}}" ("role_id");
`
	pgsqlV24DownSQL = `ALTER TABLE "{{users}}" DROP COLUMN "role_id" CASCADE;
ALTER TABLE "{{admins}}" DROP COLUMN "role_id" CASCADE;
DROP TABLE "{{roles}}" CASCADE;
`
	pgsqlV25SQL = `ALTER TABLE "{{users}}" ADD COLUMN "last_password_change" bigint DEFAULT 0 NOT NULL;
ALTER TABLE "{{users}}" ALTER COLUMN "last_password_change" DROP DEFAULT;
`
	pgsqlV25DownSQL = `ALTER TABLE "{{users}}" DROP COLUMN "last_password_change" CASCADE;`
	pgsqlV26SQL     = `ALTER TABLE "{{events_rules}}" ADD COLUMN "status" integer DEFAULT 1 NOT NULL;
ALTER TABLE "{{events_rules}}" ALTER COLUMN "status" DROP DEFAULT;
`
	pgsqlV26DownSQL = `ALTER TABLE "{{events_rules}}" DROP COLUMN "status" CASCADE;`
	pgsqlV27SQL     = `CREATE TABLE "{{ip_lists}}" ("id" bigserial NOT NULL PRIMARY KEY, "type" integer NOT NULL,
"ipornet" varchar(50) NOT NULL, "mode" integer NOT NULL, "description" varchar(512) NULL, "first" inet NOT NULL,
"last" inet NOT NULL, "ip_type" integer NOT NULL, "protocols" integer NOT NULL,  "created_at" bigint NOT NULL,
"updated_at" bigint NOT NULL, "deleted_at" bigint NOT NULL);
ALTER TABLE "{{ip_lists}}" ADD CONSTRAINT "{{prefix}}unique_ipornet_type_mapping" UNIQUE ("type", "ipornet");
CREATE INDEX "{{prefix}}ip_lists_type_idx" ON "{{ip_lists}}" ("type");
CREATE INDEX "{{prefix}}ip_lists_ipornet_idx" ON "{{ip_lists}}" ("ipornet");
CREATE INDEX "{{prefix}}ip_lists_ipornet_like_idx" ON "{{ip_lists}}" ("ipornet" varchar_pattern_ops);
CREATE INDEX "{{prefix}}ip_lists_updated_at_idx" ON "{{ip_lists}}" ("updated_at");
CREATE INDEX "{{prefix}}ip_lists_deleted_at_idx" ON "{{ip_lists}}" ("deleted_at");
CREATE INDEX "{{prefix}}ip_lists_first_last_idx" ON "{{ip_lists}}" ("first", "last");
`
	pgsqlV27DownSQL = `DROP TABLE "{{ip_lists}}" CASCADE;`
	pgsqlV28SQL     = `CREATE TABLE "{{configs}}" ("id" serial NOT NULL PRIMARY KEY, "configs" text NOT NULL);
INSERT INTO {{configs}} (configs) VALUES ('{}');
`
	pgsqlV28DownSQL = `DROP TABLE "{{configs}}" CASCADE;`
)

// PGSQLProvider defines the auth provider for PostgreSQL database
type PGSQLProvider struct {
	dbHandle *sql.DB
}

func init() {
	version.AddFeature("+pgsql")
}

func initializePGSQLProvider() error {
	var dbHandle *sql.DB
	if config.TargetSessionAttrs == "any" {
		pgxConfig, err := pgx.ParseConfig(getPGSQLConnectionString(false))
		if err != nil {
			providerLog(logger.LevelError, "error parsing postgres configuration, connection string: %q, error: %v",
				getPGSQLConnectionString(true), err)
			return err
		}
		dbHandle = stdlib.OpenDB(*pgxConfig, stdlib.OptionBeforeConnect(stdlib.RandomizeHostOrderFunc))
	} else {
		var err error
		dbHandle, err = sql.Open("pgx", getPGSQLConnectionString(false))
		if err != nil {
			providerLog(logger.LevelError, "error creating postgres database handler, connection string: %q, error: %v",
				getPGSQLConnectionString(true), err)
			return err
		}
	}
	providerLog(logger.LevelDebug, "postgres database handle created, connection string: %q, pool size: %d",
		getPGSQLConnectionString(true), config.PoolSize)
	dbHandle.SetMaxOpenConns(config.PoolSize)
	if config.PoolSize > 0 {
		dbHandle.SetMaxIdleConns(config.PoolSize)
	} else {
		dbHandle.SetMaxIdleConns(2)
	}
	dbHandle.SetConnMaxLifetime(240 * time.Second)
	dbHandle.SetConnMaxIdleTime(120 * time.Second)
	provider = &PGSQLProvider{dbHandle: dbHandle}

	ctx, cancel := context.WithTimeout(context.Background(), defaultSQLQueryTimeout)
	defer cancel()

	return dbHandle.PingContext(ctx)
}

func getPGSQLHostsAndPorts(configHost string, configPort int) (string, string) {
	var hosts, ports []string
	defaultPort := strconv.Itoa(configPort)
	if defaultPort == "0" {
		defaultPort = "5432"
	}

	for _, hostport := range strings.Split(configHost, ",") {
		hostport = strings.TrimSpace(hostport)
		if hostport == "" {
			continue
		}
		host, port, err := net.SplitHostPort(hostport)
		if err == nil {
			hosts = append(hosts, host)
			ports = append(ports, port)
		} else {
			hosts = append(hosts, hostport)
			ports = append(ports, defaultPort)
		}
	}

	return strings.Join(hosts, ","), strings.Join(ports, ",")
}

func getPGSQLConnectionString(redactedPwd bool) string {
	var connectionString string
	if config.ConnectionString == "" {
		password := config.Password
		if redactedPwd && password != "" {
			password = "[redacted]"
		}
		host, port := getPGSQLHostsAndPorts(config.Host, config.Port)
		connectionString = fmt.Sprintf("host='%s' port='%s' dbname='%s' user='%s' password='%s' sslmode=%s connect_timeout=10",
			host, port, config.Name, config.Username, password, getSSLMode())
		if config.RootCert != "" {
			connectionString += fmt.Sprintf(" sslrootcert='%s'", config.RootCert)
		}
		if config.ClientCert != "" && config.ClientKey != "" {
			connectionString += fmt.Sprintf(" sslcert='%s' sslkey='%s'", config.ClientCert, config.ClientKey)
		}
		if config.DisableSNI {
			connectionString += " sslsni=0"
		}
		if config.TargetSessionAttrs != "" {
			connectionString += fmt.Sprintf(" target_session_attrs='%s'", config.TargetSessionAttrs)
		}
	} else {
		connectionString = config.ConnectionString
	}
	return connectionString
}

func (p *PGSQLProvider) checkAvailability() error {
	return sqlCommonCheckAvailability(p.dbHandle)
}

func (p *PGSQLProvider) validateUserAndPass(username, password, ip, protocol string) (User, error) {
	return sqlCommonValidateUserAndPass(username, password, ip, protocol, p.dbHandle)
}

func (p *PGSQLProvider) validateUserAndTLSCert(username, protocol string, tlsCert *x509.Certificate) (User, error) {
	return sqlCommonValidateUserAndTLSCertificate(username, protocol, tlsCert, p.dbHandle)
}

func (p *PGSQLProvider) validateUserAndPubKey(username string, publicKey []byte, isSSHCert bool) (User, string, error) {
	return sqlCommonValidateUserAndPubKey(username, publicKey, isSSHCert, p.dbHandle)
}

func (p *PGSQLProvider) updateTransferQuota(username string, uploadSize, downloadSize int64, reset bool) error {
	return sqlCommonUpdateTransferQuota(username, uploadSize, downloadSize, reset, p.dbHandle)
}

func (p *PGSQLProvider) updateQuota(username string, filesAdd int, sizeAdd int64, reset bool) error {
	return sqlCommonUpdateQuota(username, filesAdd, sizeAdd, reset, p.dbHandle)
}

func (p *PGSQLProvider) getUsedQuota(username string) (int, int64, int64, int64, error) {
	return sqlCommonGetUsedQuota(username, p.dbHandle)
}

func (p *PGSQLProvider) setUpdatedAt(username string) {
	sqlCommonSetUpdatedAt(username, p.dbHandle)
}

func (p *PGSQLProvider) updateLastLogin(username string) error {
	return sqlCommonUpdateLastLogin(username, p.dbHandle)
}

func (p *PGSQLProvider) updateAdminLastLogin(username string) error {
	return sqlCommonUpdateAdminLastLogin(username, p.dbHandle)
}

func (p *PGSQLProvider) userExists(username, role string) (User, error) {
	return sqlCommonGetUserByUsername(username, role, p.dbHandle)
}

func (p *PGSQLProvider) addUser(user *User) error {
	return sqlCommonAddUser(user, p.dbHandle)
}

func (p *PGSQLProvider) updateUser(user *User) error {
	return sqlCommonUpdateUser(user, p.dbHandle)
}

func (p *PGSQLProvider) deleteUser(user User, softDelete bool) error {
	return sqlCommonDeleteUser(user, softDelete, p.dbHandle)
}

func (p *PGSQLProvider) updateUserPassword(username, password string) error {
	return sqlCommonUpdateUserPassword(username, password, p.dbHandle)
}

func (p *PGSQLProvider) dumpUsers() ([]User, error) {
	return sqlCommonDumpUsers(p.dbHandle)
}

func (p *PGSQLProvider) getRecentlyUpdatedUsers(after int64) ([]User, error) {
	return sqlCommonGetRecentlyUpdatedUsers(after, p.dbHandle)
}

func (p *PGSQLProvider) getUsers(limit int, offset int, order, role string) ([]User, error) {
	return sqlCommonGetUsers(limit, offset, order, role, p.dbHandle)
}

func (p *PGSQLProvider) getUsersForQuotaCheck(toFetch map[string]bool) ([]User, error) {
	return sqlCommonGetUsersForQuotaCheck(toFetch, p.dbHandle)
}

func (p *PGSQLProvider) dumpFolders() ([]vfs.BaseVirtualFolder, error) {
	return sqlCommonDumpFolders(p.dbHandle)
}

func (p *PGSQLProvider) getFolders(limit, offset int, order string, minimal bool) ([]vfs.BaseVirtualFolder, error) {
	return sqlCommonGetFolders(limit, offset, order, minimal, p.dbHandle)
}

func (p *PGSQLProvider) getFolderByName(name string) (vfs.BaseVirtualFolder, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultSQLQueryTimeout)
	defer cancel()
	return sqlCommonGetFolderByName(ctx, name, p.dbHandle)
}

func (p *PGSQLProvider) addFolder(folder *vfs.BaseVirtualFolder) error {
	return sqlCommonAddFolder(folder, p.dbHandle)
}

func (p *PGSQLProvider) updateFolder(folder *vfs.BaseVirtualFolder) error {
	return sqlCommonUpdateFolder(folder, p.dbHandle)
}

func (p *PGSQLProvider) deleteFolder(folder vfs.BaseVirtualFolder) error {
	return sqlCommonDeleteFolder(folder, p.dbHandle)
}

func (p *PGSQLProvider) updateFolderQuota(name string, filesAdd int, sizeAdd int64, reset bool) error {
	return sqlCommonUpdateFolderQuota(name, filesAdd, sizeAdd, reset, p.dbHandle)
}

func (p *PGSQLProvider) getUsedFolderQuota(name string) (int, int64, error) {
	return sqlCommonGetFolderUsedQuota(name, p.dbHandle)
}

func (p *PGSQLProvider) getGroups(limit, offset int, order string, minimal bool) ([]Group, error) {
	return sqlCommonGetGroups(limit, offset, order, minimal, p.dbHandle)
}

func (p *PGSQLProvider) getGroupsWithNames(names []string) ([]Group, error) {
	return sqlCommonGetGroupsWithNames(names, p.dbHandle)
}

func (p *PGSQLProvider) getUsersInGroups(names []string) ([]string, error) {
	return sqlCommonGetUsersInGroups(names, p.dbHandle)
}

func (p *PGSQLProvider) groupExists(name string) (Group, error) {
	return sqlCommonGetGroupByName(name, p.dbHandle)
}

func (p *PGSQLProvider) addGroup(group *Group) error {
	return sqlCommonAddGroup(group, p.dbHandle)
}

func (p *PGSQLProvider) updateGroup(group *Group) error {
	return sqlCommonUpdateGroup(group, p.dbHandle)
}

func (p *PGSQLProvider) deleteGroup(group Group) error {
	return sqlCommonDeleteGroup(group, p.dbHandle)
}

func (p *PGSQLProvider) dumpGroups() ([]Group, error) {
	return sqlCommonDumpGroups(p.dbHandle)
}

func (p *PGSQLProvider) adminExists(username string) (Admin, error) {
	return sqlCommonGetAdminByUsername(username, p.dbHandle)
}

func (p *PGSQLProvider) addAdmin(admin *Admin) error {
	return sqlCommonAddAdmin(admin, p.dbHandle)
}

func (p *PGSQLProvider) updateAdmin(admin *Admin) error {
	return sqlCommonUpdateAdmin(admin, p.dbHandle)
}

func (p *PGSQLProvider) deleteAdmin(admin Admin) error {
	return sqlCommonDeleteAdmin(admin, p.dbHandle)
}

func (p *PGSQLProvider) getAdmins(limit int, offset int, order string) ([]Admin, error) {
	return sqlCommonGetAdmins(limit, offset, order, p.dbHandle)
}

func (p *PGSQLProvider) dumpAdmins() ([]Admin, error) {
	return sqlCommonDumpAdmins(p.dbHandle)
}

func (p *PGSQLProvider) validateAdminAndPass(username, password, ip string) (Admin, error) {
	return sqlCommonValidateAdminAndPass(username, password, ip, p.dbHandle)
}

func (p *PGSQLProvider) apiKeyExists(keyID string) (APIKey, error) {
	return sqlCommonGetAPIKeyByID(keyID, p.dbHandle)
}

func (p *PGSQLProvider) addAPIKey(apiKey *APIKey) error {
	return sqlCommonAddAPIKey(apiKey, p.dbHandle)
}

func (p *PGSQLProvider) updateAPIKey(apiKey *APIKey) error {
	return sqlCommonUpdateAPIKey(apiKey, p.dbHandle)
}

func (p *PGSQLProvider) deleteAPIKey(apiKey APIKey) error {
	return sqlCommonDeleteAPIKey(apiKey, p.dbHandle)
}

func (p *PGSQLProvider) getAPIKeys(limit int, offset int, order string) ([]APIKey, error) {
	return sqlCommonGetAPIKeys(limit, offset, order, p.dbHandle)
}

func (p *PGSQLProvider) dumpAPIKeys() ([]APIKey, error) {
	return sqlCommonDumpAPIKeys(p.dbHandle)
}

func (p *PGSQLProvider) updateAPIKeyLastUse(keyID string) error {
	return sqlCommonUpdateAPIKeyLastUse(keyID, p.dbHandle)
}

func (p *PGSQLProvider) shareExists(shareID, username string) (Share, error) {
	return sqlCommonGetShareByID(shareID, username, p.dbHandle)
}

func (p *PGSQLProvider) addShare(share *Share) error {
	return sqlCommonAddShare(share, p.dbHandle)
}

func (p *PGSQLProvider) updateShare(share *Share) error {
	return sqlCommonUpdateShare(share, p.dbHandle)
}

func (p *PGSQLProvider) deleteShare(share Share) error {
	return sqlCommonDeleteShare(share, p.dbHandle)
}

func (p *PGSQLProvider) getShares(limit int, offset int, order, username string) ([]Share, error) {
	return sqlCommonGetShares(limit, offset, order, username, p.dbHandle)
}

func (p *PGSQLProvider) dumpShares() ([]Share, error) {
	return sqlCommonDumpShares(p.dbHandle)
}

func (p *PGSQLProvider) updateShareLastUse(shareID string, numTokens int) error {
	return sqlCommonUpdateShareLastUse(shareID, numTokens, p.dbHandle)
}

func (p *PGSQLProvider) getDefenderHosts(from int64, limit int) ([]DefenderEntry, error) {
	return sqlCommonGetDefenderHosts(from, limit, p.dbHandle)
}

func (p *PGSQLProvider) getDefenderHostByIP(ip string, from int64) (DefenderEntry, error) {
	return sqlCommonGetDefenderHostByIP(ip, from, p.dbHandle)
}

func (p *PGSQLProvider) isDefenderHostBanned(ip string) (DefenderEntry, error) {
	return sqlCommonIsDefenderHostBanned(ip, p.dbHandle)
}

func (p *PGSQLProvider) updateDefenderBanTime(ip string, minutes int) error {
	return sqlCommonDefenderIncrementBanTime(ip, minutes, p.dbHandle)
}

func (p *PGSQLProvider) deleteDefenderHost(ip string) error {
	return sqlCommonDeleteDefenderHost(ip, p.dbHandle)
}

func (p *PGSQLProvider) addDefenderEvent(ip string, score int) error {
	return sqlCommonAddDefenderHostAndEvent(ip, score, p.dbHandle)
}

func (p *PGSQLProvider) setDefenderBanTime(ip string, banTime int64) error {
	return sqlCommonSetDefenderBanTime(ip, banTime, p.dbHandle)
}

func (p *PGSQLProvider) cleanupDefender(from int64) error {
	return sqlCommonDefenderCleanup(from, p.dbHandle)
}

func (p *PGSQLProvider) addActiveTransfer(transfer ActiveTransfer) error {
	return sqlCommonAddActiveTransfer(transfer, p.dbHandle)
}

func (p *PGSQLProvider) updateActiveTransferSizes(ulSize, dlSize, transferID int64, connectionID string) error {
	return sqlCommonUpdateActiveTransferSizes(ulSize, dlSize, transferID, connectionID, p.dbHandle)
}

func (p *PGSQLProvider) removeActiveTransfer(transferID int64, connectionID string) error {
	return sqlCommonRemoveActiveTransfer(transferID, connectionID, p.dbHandle)
}

func (p *PGSQLProvider) cleanupActiveTransfers(before time.Time) error {
	return sqlCommonCleanupActiveTransfers(before, p.dbHandle)
}

func (p *PGSQLProvider) getActiveTransfers(from time.Time) ([]ActiveTransfer, error) {
	return sqlCommonGetActiveTransfers(from, p.dbHandle)
}

func (p *PGSQLProvider) addSharedSession(session Session) error {
	return sqlCommonAddSession(session, p.dbHandle)
}

func (p *PGSQLProvider) deleteSharedSession(key string) error {
	return sqlCommonDeleteSession(key, p.dbHandle)
}

func (p *PGSQLProvider) getSharedSession(key string) (Session, error) {
	return sqlCommonGetSession(key, p.dbHandle)
}

func (p *PGSQLProvider) cleanupSharedSessions(sessionType SessionType, before int64) error {
	return sqlCommonCleanupSessions(sessionType, before, p.dbHandle)
}

func (p *PGSQLProvider) getEventActions(limit, offset int, order string, minimal bool) ([]BaseEventAction, error) {
	return sqlCommonGetEventActions(limit, offset, order, minimal, p.dbHandle)
}

func (p *PGSQLProvider) dumpEventActions() ([]BaseEventAction, error) {
	return sqlCommonDumpEventActions(p.dbHandle)
}

func (p *PGSQLProvider) eventActionExists(name string) (BaseEventAction, error) {
	return sqlCommonGetEventActionByName(name, p.dbHandle)
}

func (p *PGSQLProvider) addEventAction(action *BaseEventAction) error {
	return sqlCommonAddEventAction(action, p.dbHandle)
}

func (p *PGSQLProvider) updateEventAction(action *BaseEventAction) error {
	return sqlCommonUpdateEventAction(action, p.dbHandle)
}

func (p *PGSQLProvider) deleteEventAction(action BaseEventAction) error {
	return sqlCommonDeleteEventAction(action, p.dbHandle)
}

func (p *PGSQLProvider) getEventRules(limit, offset int, order string) ([]EventRule, error) {
	return sqlCommonGetEventRules(limit, offset, order, p.dbHandle)
}

func (p *PGSQLProvider) dumpEventRules() ([]EventRule, error) {
	return sqlCommonDumpEventRules(p.dbHandle)
}

func (p *PGSQLProvider) getRecentlyUpdatedRules(after int64) ([]EventRule, error) {
	return sqlCommonGetRecentlyUpdatedRules(after, p.dbHandle)
}

func (p *PGSQLProvider) eventRuleExists(name string) (EventRule, error) {
	return sqlCommonGetEventRuleByName(name, p.dbHandle)
}

func (p *PGSQLProvider) addEventRule(rule *EventRule) error {
	return sqlCommonAddEventRule(rule, p.dbHandle)
}

func (p *PGSQLProvider) updateEventRule(rule *EventRule) error {
	return sqlCommonUpdateEventRule(rule, p.dbHandle)
}

func (p *PGSQLProvider) deleteEventRule(rule EventRule, softDelete bool) error {
	return sqlCommonDeleteEventRule(rule, softDelete, p.dbHandle)
}

func (p *PGSQLProvider) getTaskByName(name string) (Task, error) {
	return sqlCommonGetTaskByName(name, p.dbHandle)
}

func (p *PGSQLProvider) addTask(name string) error {
	return sqlCommonAddTask(name, p.dbHandle)
}

func (p *PGSQLProvider) updateTask(name string, version int64) error {
	return sqlCommonUpdateTask(name, version, p.dbHandle)
}

func (p *PGSQLProvider) updateTaskTimestamp(name string) error {
	return sqlCommonUpdateTaskTimestamp(name, p.dbHandle)
}

func (p *PGSQLProvider) addNode() error {
	return sqlCommonAddNode(p.dbHandle)
}

func (p *PGSQLProvider) getNodeByName(name string) (Node, error) {
	return sqlCommonGetNodeByName(name, p.dbHandle)
}

func (p *PGSQLProvider) getNodes() ([]Node, error) {
	return sqlCommonGetNodes(p.dbHandle)
}

func (p *PGSQLProvider) updateNodeTimestamp() error {
	return sqlCommonUpdateNodeTimestamp(p.dbHandle)
}

func (p *PGSQLProvider) cleanupNodes() error {
	return sqlCommonCleanupNodes(p.dbHandle)
}

func (p *PGSQLProvider) roleExists(name string) (Role, error) {
	return sqlCommonGetRoleByName(name, p.dbHandle)
}

func (p *PGSQLProvider) addRole(role *Role) error {
	return sqlCommonAddRole(role, p.dbHandle)
}

func (p *PGSQLProvider) updateRole(role *Role) error {
	return sqlCommonUpdateRole(role, p.dbHandle)
}

func (p *PGSQLProvider) deleteRole(role Role) error {
	return sqlCommonDeleteRole(role, p.dbHandle)
}

func (p *PGSQLProvider) getRoles(limit int, offset int, order string, minimal bool) ([]Role, error) {
	return sqlCommonGetRoles(limit, offset, order, minimal, p.dbHandle)
}

func (p *PGSQLProvider) dumpRoles() ([]Role, error) {
	return sqlCommonDumpRoles(p.dbHandle)
}

func (p *PGSQLProvider) ipListEntryExists(ipOrNet string, listType IPListType) (IPListEntry, error) {
	return sqlCommonGetIPListEntry(ipOrNet, listType, p.dbHandle)
}

func (p *PGSQLProvider) addIPListEntry(entry *IPListEntry) error {
	return sqlCommonAddIPListEntry(entry, p.dbHandle)
}

func (p *PGSQLProvider) updateIPListEntry(entry *IPListEntry) error {
	return sqlCommonUpdateIPListEntry(entry, p.dbHandle)
}

func (p *PGSQLProvider) deleteIPListEntry(entry IPListEntry, softDelete bool) error {
	return sqlCommonDeleteIPListEntry(entry, softDelete, p.dbHandle)
}

func (p *PGSQLProvider) getIPListEntries(listType IPListType, filter, from, order string, limit int) ([]IPListEntry, error) {
	return sqlCommonGetIPListEntries(listType, filter, from, order, limit, p.dbHandle)
}

func (p *PGSQLProvider) getRecentlyUpdatedIPListEntries(after int64) ([]IPListEntry, error) {
	return sqlCommonGetRecentlyUpdatedIPListEntries(after, p.dbHandle)
}

func (p *PGSQLProvider) dumpIPListEntries() ([]IPListEntry, error) {
	return sqlCommonDumpIPListEntries(p.dbHandle)
}

func (p *PGSQLProvider) countIPListEntries(listType IPListType) (int64, error) {
	return sqlCommonCountIPListEntries(listType, p.dbHandle)
}

func (p *PGSQLProvider) getListEntriesForIP(ip string, listType IPListType) ([]IPListEntry, error) {
	return sqlCommonGetListEntriesForIP(ip, listType, p.dbHandle)
}

func (p *PGSQLProvider) getConfigs() (Configs, error) {
	return sqlCommonGetConfigs(p.dbHandle)
}

func (p *PGSQLProvider) setConfigs(configs *Configs) error {
	return sqlCommonSetConfigs(configs, p.dbHandle)
}

func (p *PGSQLProvider) setFirstDownloadTimestamp(username string) error {
	return sqlCommonSetFirstDownloadTimestamp(username, p.dbHandle)
}

func (p *PGSQLProvider) setFirstUploadTimestamp(username string) error {
	return sqlCommonSetFirstUploadTimestamp(username, p.dbHandle)
}

func (p *PGSQLProvider) close() error {
	return p.dbHandle.Close()
}

func (p *PGSQLProvider) reloadConfig() error {
	return nil
}

// initializeDatabase creates the initial database structure
func (p *PGSQLProvider) initializeDatabase() error {
	dbVersion, err := sqlCommonGetDatabaseVersion(p.dbHandle, false)
	if err == nil && dbVersion.Version > 0 {
		return ErrNoInitRequired
	}
	if errors.Is(err, sql.ErrNoRows) {
		return errSchemaVersionEmpty
	}
	logger.InfoToConsole("creating initial database schema, version 23")
	providerLog(logger.LevelInfo, "creating initial database schema, version 23")
	initialSQL := sqlReplaceAll(pgsqlInitial)

	return sqlCommonExecSQLAndUpdateDBVersion(p.dbHandle, []string{initialSQL}, 23, true)
}

func (p *PGSQLProvider) migrateDatabase() error { //nolint:dupl
	dbVersion, err := sqlCommonGetDatabaseVersion(p.dbHandle, true)
	if err != nil {
		return err
	}

	switch version := dbVersion.Version; {
	case version == sqlDatabaseVersion:
		providerLog(logger.LevelDebug, "sql database is up to date, current version: %d", version)
		return ErrNoInitRequired
	case version < 23:
		err = fmt.Errorf("database schema version %d is too old, please see the upgrading docs", version)
		providerLog(logger.LevelError, "%v", err)
		logger.ErrorToConsole("%v", err)
		return err
	case version == 23:
		return updatePgSQLDatabaseFromV23(p.dbHandle)
	case version == 24:
		return updatePgSQLDatabaseFromV24(p.dbHandle)
	case version == 25:
		return updatePgSQLDatabaseFromV25(p.dbHandle)
	case version == 26:
		return updatePgSQLDatabaseFromV26(p.dbHandle)
	case version == 27:
		return updatePgSQLDatabaseFromV27(p.dbHandle)
	default:
		if version > sqlDatabaseVersion {
			providerLog(logger.LevelError, "database schema version %d is newer than the supported one: %d", version,
				sqlDatabaseVersion)
			logger.WarnToConsole("database schema version %d is newer than the supported one: %d", version,
				sqlDatabaseVersion)
			return nil
		}
		return fmt.Errorf("database schema version not handled: %d", version)
	}
}

func (p *PGSQLProvider) revertDatabase(targetVersion int) error {
	dbVersion, err := sqlCommonGetDatabaseVersion(p.dbHandle, true)
	if err != nil {
		return err
	}
	if dbVersion.Version == targetVersion {
		return errors.New("current version match target version, nothing to do")
	}

	switch dbVersion.Version {
	case 24:
		return downgradePgSQLDatabaseFromV24(p.dbHandle)
	case 25:
		return downgradePgSQLDatabaseFromV25(p.dbHandle)
	case 26:
		return downgradePgSQLDatabaseFromV26(p.dbHandle)
	case 27:
		return downgradePgSQLDatabaseFromV27(p.dbHandle)
	case 28:
		return downgradePgSQLDatabaseFromV28(p.dbHandle)
	default:
		return fmt.Errorf("database schema version not handled: %d", dbVersion.Version)
	}
}

func (p *PGSQLProvider) resetDatabase() error {
	sql := sqlReplaceAll(pgsqlResetSQL)
	return sqlCommonExecSQLAndUpdateDBVersion(p.dbHandle, []string{sql}, 0, false)
}

func updatePgSQLDatabaseFromV23(dbHandle *sql.DB) error {
	if err := updatePgSQLDatabaseFrom23To24(dbHandle); err != nil {
		return err
	}
	return updatePgSQLDatabaseFromV24(dbHandle)
}

func updatePgSQLDatabaseFromV24(dbHandle *sql.DB) error {
	if err := updatePgSQLDatabaseFrom24To25(dbHandle); err != nil {
		return err
	}
	return updatePgSQLDatabaseFromV25(dbHandle)
}

func updatePgSQLDatabaseFromV25(dbHandle *sql.DB) error {
	if err := updatePgSQLDatabaseFrom25To26(dbHandle); err != nil {
		return err
	}
	return updatePgSQLDatabaseFromV26(dbHandle)
}

func updatePgSQLDatabaseFromV26(dbHandle *sql.DB) error {
	if err := updatePgSQLDatabaseFrom26To27(dbHandle); err != nil {
		return err
	}
	return updatePgSQLDatabaseFromV27(dbHandle)
}

func updatePgSQLDatabaseFromV27(dbHandle *sql.DB) error {
	return updatePgSQLDatabaseFrom27To28(dbHandle)
}

func downgradePgSQLDatabaseFromV24(dbHandle *sql.DB) error {
	return downgradePgSQLDatabaseFrom24To23(dbHandle)
}

func downgradePgSQLDatabaseFromV25(dbHandle *sql.DB) error {
	if err := downgradePgSQLDatabaseFrom25To24(dbHandle); err != nil {
		return err
	}
	return downgradePgSQLDatabaseFromV24(dbHandle)
}

func downgradePgSQLDatabaseFromV26(dbHandle *sql.DB) error {
	if err := downgradePgSQLDatabaseFrom26To25(dbHandle); err != nil {
		return err
	}
	return downgradePgSQLDatabaseFromV25(dbHandle)
}

func downgradePgSQLDatabaseFromV27(dbHandle *sql.DB) error {
	if err := downgradePgSQLDatabaseFrom27To26(dbHandle); err != nil {
		return err
	}
	return downgradePgSQLDatabaseFromV26(dbHandle)
}

func downgradePgSQLDatabaseFromV28(dbHandle *sql.DB) error {
	if err := downgradePgSQLDatabaseFrom28To27(dbHandle); err != nil {
		return err
	}
	return downgradePgSQLDatabaseFromV27(dbHandle)
}

func updatePgSQLDatabaseFrom23To24(dbHandle *sql.DB) error {
	logger.InfoToConsole("updating database schema version: 23 -> 24")
	providerLog(logger.LevelInfo, "updating database schema version: 23 -> 24")
	sql := strings.ReplaceAll(pgsqlV24SQL, "{{roles}}", sqlTableRoles)
	sql = strings.ReplaceAll(sql, "{{admins}}", sqlTableAdmins)
	sql = strings.ReplaceAll(sql, "{{users}}", sqlTableUsers)
	sql = strings.ReplaceAll(sql, "{{prefix}}", config.SQLTablesPrefix)
	return sqlCommonExecSQLAndUpdateDBVersion(dbHandle, []string{sql}, 24, true)
}

func updatePgSQLDatabaseFrom24To25(dbHandle *sql.DB) error {
	logger.InfoToConsole("updating database schema version: 24 -> 25")
	providerLog(logger.LevelInfo, "updating database schema version: 24 -> 25")
	sql := pgsqlV25SQL
	if config.Driver == CockroachDataProviderName {
		sql = strings.ReplaceAll(sql, `ALTER TABLE "{{users}}" ALTER COLUMN "last_password_change" DROP DEFAULT;`, "")
	}
	sql = strings.ReplaceAll(sql, "{{users}}", sqlTableUsers)
	return sqlCommonExecSQLAndUpdateDBVersion(dbHandle, []string{sql}, 25, true)
}

func updatePgSQLDatabaseFrom25To26(dbHandle *sql.DB) error {
	logger.InfoToConsole("updating database schema version: 25 -> 26")
	providerLog(logger.LevelInfo, "updating database schema version: 25 -> 26")
	sql := pgsqlV26SQL
	if config.Driver == CockroachDataProviderName {
		sql = strings.ReplaceAll(sql, `ALTER TABLE "{{events_rules}}" ALTER COLUMN "status" DROP DEFAULT;`, "")
	}
	sql = strings.ReplaceAll(sql, "{{events_rules}}", sqlTableEventsRules)
	return sqlCommonExecSQLAndUpdateDBVersion(dbHandle, []string{sql}, 26, true)
}

func updatePgSQLDatabaseFrom26To27(dbHandle *sql.DB) error {
	logger.InfoToConsole("updating database schema version: 26 -> 27")
	providerLog(logger.LevelInfo, "updating database schema version: 26 -> 27")
	sql := pgsqlV27SQL
	if config.Driver == CockroachDataProviderName {
		sql = strings.ReplaceAll(sql, `CREATE INDEX "{{prefix}}ip_lists_ipornet_like_idx" ON "{{ip_lists}}" ("ipornet" varchar_pattern_ops);`, "")
	}
	sql = strings.ReplaceAll(sql, "{{ip_lists}}", sqlTableIPLists)
	sql = strings.ReplaceAll(sql, "{{prefix}}", config.SQLTablesPrefix)
	return sqlCommonExecSQLAndUpdateDBVersion(dbHandle, []string{sql}, 27, true)
}

func updatePgSQLDatabaseFrom27To28(dbHandle *sql.DB) error {
	logger.InfoToConsole("updating database schema version: 27 -> 28")
	providerLog(logger.LevelInfo, "updating database schema version: 27 -> 28")
	sql := strings.ReplaceAll(pgsqlV28SQL, "{{configs}}", sqlTableConfigs)
	return sqlCommonExecSQLAndUpdateDBVersion(dbHandle, []string{sql}, 28, true)
}

func downgradePgSQLDatabaseFrom24To23(dbHandle *sql.DB) error {
	logger.InfoToConsole("downgrading database schema version: 24 -> 23")
	providerLog(logger.LevelInfo, "downgrading database schema version: 24 -> 23")
	sql := strings.ReplaceAll(pgsqlV24DownSQL, "{{roles}}", sqlTableRoles)
	sql = strings.ReplaceAll(sql, "{{admins}}", sqlTableAdmins)
	sql = strings.ReplaceAll(sql, "{{users}}", sqlTableUsers)
	sql = strings.ReplaceAll(sql, "{{prefix}}", config.SQLTablesPrefix)
	return sqlCommonExecSQLAndUpdateDBVersion(dbHandle, []string{sql}, 23, false)
}

func downgradePgSQLDatabaseFrom25To24(dbHandle *sql.DB) error {
	logger.InfoToConsole("downgrading database schema version: 25 -> 24")
	providerLog(logger.LevelInfo, "downgrading database schema version: 25 -> 24")
	sql := strings.ReplaceAll(pgsqlV25DownSQL, "{{users}}", sqlTableUsers)
	return sqlCommonExecSQLAndUpdateDBVersion(dbHandle, []string{sql}, 24, false)
}

func downgradePgSQLDatabaseFrom26To25(dbHandle *sql.DB) error {
	logger.InfoToConsole("downgrading database schema version: 26 -> 25")
	providerLog(logger.LevelInfo, "downgrading database schema version: 26 -> 25")
	sql := strings.ReplaceAll(pgsqlV26DownSQL, "{{events_rules}}", sqlTableEventsRules)
	return sqlCommonExecSQLAndUpdateDBVersion(dbHandle, []string{sql}, 25, false)
}

func downgradePgSQLDatabaseFrom27To26(dbHandle *sql.DB) error {
	logger.InfoToConsole("downgrading database schema version: 27 -> 26")
	providerLog(logger.LevelInfo, "downgrading database schema version: 27 -> 26")
	sql := strings.ReplaceAll(pgsqlV27DownSQL, "{{ip_lists}}", sqlTableIPLists)
	return sqlCommonExecSQLAndUpdateDBVersion(dbHandle, []string{sql}, 26, false)
}

func downgradePgSQLDatabaseFrom28To27(dbHandle *sql.DB) error {
	logger.InfoToConsole("downgrading database schema version: 28 -> 27")
	providerLog(logger.LevelInfo, "downgrading database schema version: 28 -> 27")
	sql := strings.ReplaceAll(pgsqlV28DownSQL, "{{configs}}", sqlTableConfigs)
	return sqlCommonExecSQLAndUpdateDBVersion(dbHandle, []string{sql}, 27, false)
}
