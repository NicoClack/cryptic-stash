-- +goose Up
-- create "download_sessions" table
CREATE TABLE `download_sessions` (`id` uuid NOT NULL, `created_at` datetime NOT NULL, `hashed_auth_code` blob NOT NULL, `valid_from` datetime NOT NULL, `valid_until` datetime NOT NULL, `user_agent` text NOT NULL, `ip` text NOT NULL, `user_id` uuid NOT NULL, PRIMARY KEY (`id`), CONSTRAINT `download_sessions_users_downloadSessions` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE);
-- create index "download_sessions_hashed_auth_code_key" to table: "download_sessions"
CREATE UNIQUE INDEX `download_sessions_hashed_auth_code_key` ON `download_sessions` (`hashed_auth_code`);
-- create index "downloadsession_hashed_auth_code_user_id" to table: "download_sessions"
CREATE INDEX `downloadsession_hashed_auth_code_user_id` ON `download_sessions` (`hashed_auth_code`, `user_id`);
-- create "jobs" table
CREATE TABLE `jobs` (`id` uuid NOT NULL, `created_at` datetime NOT NULL, `due_at` datetime NOT NULL, `originally_due_at` datetime NOT NULL, `started_at` datetime NULL, `type` text NOT NULL, `version` integer NOT NULL, `priority` integer NOT NULL, `weight` integer NOT NULL, `body` json NOT NULL, `status` text NOT NULL DEFAULT ('pending'), `retries` integer NOT NULL DEFAULT (0), `retried_fraction` real NOT NULL DEFAULT (0), `logged_stall_warning` bool NOT NULL DEFAULT (false), PRIMARY KEY (`id`));
-- create index "job_status_priority_due_at" to table: "jobs"
CREATE INDEX `job_status_priority_due_at` ON `jobs` (`status`, `priority`, `due_at`);
-- create index "job_due_at" to table: "jobs"
CREATE INDEX `job_due_at` ON `jobs` (`due_at`);
-- create "key_values" table
CREATE TABLE `key_values` (`id` uuid NOT NULL, `key` text NOT NULL, `value` json NOT NULL, PRIMARY KEY (`id`));
-- create index "keyvalue_key" to table: "key_values"
CREATE UNIQUE INDEX `keyvalue_key` ON `key_values` (`key`);
-- create "log_entries" table
CREATE TABLE `log_entries` (`id` uuid NOT NULL, `logged_at` datetime NOT NULL, `logged_at_known` bool NOT NULL, `level` integer NOT NULL, `message` text NOT NULL, `attributes` json NOT NULL, `source_file` text NOT NULL, `source_function` text NOT NULL, `source_line` integer NOT NULL, `public_message` text NOT NULL, `user_id` uuid NULL, PRIMARY KEY (`id`), CONSTRAINT `log_entries_users_logs` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE SET NULL);
-- create index "logentry_logged_at" to table: "log_entries"
CREATE INDEX `logentry_logged_at` ON `log_entries` (`logged_at`);
-- create "login_alerts" table
CREATE TABLE `login_alerts` (`id` uuid NOT NULL, `sent_at` datetime NOT NULL, `confirmed` bool NOT NULL, `download_session_id` uuid NOT NULL, `user_messenger_id` uuid NOT NULL, PRIMARY KEY (`id`), CONSTRAINT `login_alerts_download_sessions_loginAlerts` FOREIGN KEY (`download_session_id`) REFERENCES `download_sessions` (`id`) ON DELETE CASCADE, CONSTRAINT `login_alerts_user_messengers_loginAlerts` FOREIGN KEY (`user_messenger_id`) REFERENCES `user_messengers` (`id`) ON DELETE CASCADE);
-- create "periodic_tasks" table
CREATE TABLE `periodic_tasks` (`id` uuid NOT NULL, `name` text NOT NULL, `last_ran_at` datetime NULL, PRIMARY KEY (`id`));
-- create index "periodictask_name" to table: "periodic_tasks"
CREATE UNIQUE INDEX `periodictask_name` ON `periodic_tasks` (`name`);
-- create "signup_links" table
CREATE TABLE `signup_links` (`id` uuid NOT NULL, `created_at` datetime NOT NULL, `name` text NOT NULL, `hashed_code` blob NOT NULL, `expires_at` datetime NOT NULL, `user_agent` text NOT NULL DEFAULT (''), `ip` text NOT NULL DEFAULT (''), `user_id` uuid NULL, PRIMARY KEY (`id`), CONSTRAINT `signup_links_users_signupLink` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE);
-- create index "signup_links_hashed_code_key" to table: "signup_links"
CREATE UNIQUE INDEX `signup_links_hashed_code_key` ON `signup_links` (`hashed_code`);
-- create index "signup_links_user_id_key" to table: "signup_links"
CREATE UNIQUE INDEX `signup_links_user_id_key` ON `signup_links` (`user_id`);
-- create index "signuplink_hashed_code" to table: "signup_links"
CREATE INDEX `signuplink_hashed_code` ON `signup_links` (`hashed_code`);
-- create index "signuplink_created_at" to table: "signup_links"
CREATE INDEX `signuplink_created_at` ON `signup_links` (`created_at`);
-- create "stashes" table
CREATE TABLE `stashes` (`id` uuid NOT NULL, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `last_download_at` datetime NULL, `content` blob NOT NULL, `file_name` blob NOT NULL, `encryption_data_key` blob NOT NULL, `password_salt` blob NOT NULL, `hash_time` integer NOT NULL, `hash_memory` integer NOT NULL, `hash_threads` integer NOT NULL, `user_id` uuid NOT NULL, PRIMARY KEY (`id`), CONSTRAINT `stashes_users_stash` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE);
-- create index "stashes_user_id_key" to table: "stashes"
CREATE UNIQUE INDEX `stashes_user_id_key` ON `stashes` (`user_id`);
-- create "two_factor_actions" table
CREATE TABLE `two_factor_actions` (`id` uuid NOT NULL, `type` text NOT NULL, `version` integer NOT NULL, `body` json NOT NULL, `expires_at` datetime NOT NULL, `code` text NOT NULL, PRIMARY KEY (`id`));
-- create index "twofactoraction_code" to table: "two_factor_actions"
CREATE INDEX `twofactoraction_code` ON `two_factor_actions` (`code`);
-- create "users" table
CREATE TABLE `users` (`id` uuid NOT NULL, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `username` text NOT NULL, `locked` bool NOT NULL DEFAULT (false), `locked_until` datetime NULL, `download_sessions_valid_from` datetime NOT NULL, PRIMARY KEY (`id`));
-- create index "users_username_key" to table: "users"
CREATE UNIQUE INDEX `users_username_key` ON `users` (`username`);
-- create index "user_created_at" to table: "users"
CREATE INDEX `user_created_at` ON `users` (`created_at`);
-- create "user_messengers" table
CREATE TABLE `user_messengers` (`id` uuid NOT NULL, `type` text NOT NULL, `version` integer NOT NULL, `enabled` bool NOT NULL DEFAULT (true), `options` json NOT NULL, `user_id` uuid NOT NULL, PRIMARY KEY (`id`), CONSTRAINT `user_messengers_users_messengers` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE);
-- create index "usermessenger_type_version_user_id" to table: "user_messengers"
CREATE UNIQUE INDEX `usermessenger_type_version_user_id` ON `user_messengers` (`type`, `version`, `user_id`);

-- +goose Down
-- reverse: create index "usermessenger_type_version_user_id" to table: "user_messengers"
DROP INDEX `usermessenger_type_version_user_id`;
-- reverse: create "user_messengers" table
DROP TABLE `user_messengers`;
-- reverse: create index "user_created_at" to table: "users"
DROP INDEX `user_created_at`;
-- reverse: create index "users_username_key" to table: "users"
DROP INDEX `users_username_key`;
-- reverse: create "users" table
DROP TABLE `users`;
-- reverse: create index "twofactoraction_code" to table: "two_factor_actions"
DROP INDEX `twofactoraction_code`;
-- reverse: create "two_factor_actions" table
DROP TABLE `two_factor_actions`;
-- reverse: create index "stashes_user_id_key" to table: "stashes"
DROP INDEX `stashes_user_id_key`;
-- reverse: create "stashes" table
DROP TABLE `stashes`;
-- reverse: create index "signuplink_created_at" to table: "signup_links"
DROP INDEX `signuplink_created_at`;
-- reverse: create index "signuplink_hashed_code" to table: "signup_links"
DROP INDEX `signuplink_hashed_code`;
-- reverse: create index "signup_links_user_id_key" to table: "signup_links"
DROP INDEX `signup_links_user_id_key`;
-- reverse: create index "signup_links_hashed_code_key" to table: "signup_links"
DROP INDEX `signup_links_hashed_code_key`;
-- reverse: create "signup_links" table
DROP TABLE `signup_links`;
-- reverse: create index "periodictask_name" to table: "periodic_tasks"
DROP INDEX `periodictask_name`;
-- reverse: create "periodic_tasks" table
DROP TABLE `periodic_tasks`;
-- reverse: create "login_alerts" table
DROP TABLE `login_alerts`;
-- reverse: create index "logentry_logged_at" to table: "log_entries"
DROP INDEX `logentry_logged_at`;
-- reverse: create "log_entries" table
DROP TABLE `log_entries`;
-- reverse: create index "keyvalue_key" to table: "key_values"
DROP INDEX `keyvalue_key`;
-- reverse: create "key_values" table
DROP TABLE `key_values`;
-- reverse: create index "job_due_at" to table: "jobs"
DROP INDEX `job_due_at`;
-- reverse: create index "job_status_priority_due_at" to table: "jobs"
DROP INDEX `job_status_priority_due_at`;
-- reverse: create "jobs" table
DROP TABLE `jobs`;
-- reverse: create index "downloadsession_hashed_auth_code_user_id" to table: "download_sessions"
DROP INDEX `downloadsession_hashed_auth_code_user_id`;
-- reverse: create index "download_sessions_hashed_auth_code_key" to table: "download_sessions"
DROP INDEX `download_sessions_hashed_auth_code_key`;
-- reverse: create "download_sessions" table
DROP TABLE `download_sessions`;
