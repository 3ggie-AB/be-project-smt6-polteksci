CREATE TABLE IF NOT EXISTS workspaces (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(160) NOT NULL,
  slug VARCHAR(160) NOT NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY idx_workspaces_name (name),
  UNIQUE KEY idx_workspaces_slug (slug)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS roles (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(32) NOT NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY idx_roles_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS users (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  workspace_id BIGINT UNSIGNED NULL,
  role_id BIGINT UNSIGNED NOT NULL,
  email VARCHAR(255) NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  name VARCHAR(160) NOT NULL,
  avatar_url VARCHAR(512) NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  last_login_at DATETIME(3) NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY idx_users_email (email),
  KEY idx_users_workspace_id (workspace_id),
  KEY idx_users_role_id (role_id),
  KEY idx_users_is_active (is_active),
  CONSTRAINT fk_users_workspace FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON UPDATE CASCADE ON DELETE SET NULL,
  CONSTRAINT fk_users_role FOREIGN KEY (role_id) REFERENCES roles(id) ON UPDATE CASCADE ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS devices (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  workspace_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(160) NOT NULL,
  ip_address VARCHAR(64) NOT NULL,
  mac_address VARCHAR(32) NULL,
  vendor VARCHAR(80) NULL,
  model VARCHAR(120) NULL,
  location VARCHAR(180) NULL,
  device_type VARCHAR(50) NOT NULL DEFAULT 'network',
  snmp_community VARCHAR(120) NULL,
  snmp_version VARCHAR(10) NULL DEFAULT 'v2c',
  ruijie_external_id VARCHAR(160) NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  last_seen_at DATETIME(3) NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  KEY idx_devices_workspace_ip (workspace_id, ip_address),
  KEY idx_devices_name (name),
  KEY idx_devices_mac_address (mac_address),
  KEY idx_devices_vendor (vendor),
  KEY idx_devices_location (location),
  KEY idx_devices_device_type (device_type),
  KEY idx_devices_ruijie_external_id (ruijie_external_id),
  KEY idx_devices_is_active (is_active),
  KEY idx_devices_last_seen_at (last_seen_at),
  CONSTRAINT fk_devices_workspace FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON UPDATE CASCADE ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS device_groups (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  workspace_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(160) NOT NULL,
  description VARCHAR(500) NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  KEY idx_device_groups_workspace_id (workspace_id),
  KEY idx_device_groups_name (name),
  CONSTRAINT fk_device_groups_workspace FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON UPDATE CASCADE ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS device_group_members (
  device_group_id BIGINT UNSIGNED NOT NULL,
  device_id BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (device_group_id, device_id),
  KEY idx_device_group_members_device_id (device_id),
  CONSTRAINT fk_dgm_group FOREIGN KEY (device_group_id) REFERENCES device_groups(id) ON DELETE CASCADE,
  CONSTRAINT fk_dgm_device FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS monitoring_targets (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  workspace_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(160) NOT NULL,
  host VARCHAR(255) NOT NULL,
  check_type VARCHAR(16) NOT NULL,
  port INT NOT NULL DEFAULT 0,
  interval_sec INT NOT NULL DEFAULT 0,
  timeout_sec INT NOT NULL DEFAULT 0,
  description VARCHAR(500) NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  last_checked_at DATETIME(3) NULL,
  last_status BOOLEAN NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  KEY idx_monitoring_targets_workspace_host (workspace_id, host),
  KEY idx_monitoring_targets_name (name),
  KEY idx_monitoring_targets_check_active (check_type, is_active),
  KEY idx_monitoring_targets_last_checked_at (last_checked_at),
  KEY idx_monitoring_targets_last_status (last_status),
  CONSTRAINT fk_monitoring_targets_workspace FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON UPDATE CASCADE ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS notifications (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  workspace_id BIGINT UNSIGNED NOT NULL,
  user_id BIGINT UNSIGNED NULL,
  device_id BIGINT UNSIGNED NULL,
  type VARCHAR(80) NOT NULL,
  severity VARCHAR(24) NOT NULL DEFAULT 'info',
  title VARCHAR(180) NOT NULL,
  message VARCHAR(1000) NULL,
  read_at DATETIME(3) NULL,
  created_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  KEY idx_notifications_workspace_id (workspace_id),
  KEY idx_notifications_user_id (user_id),
  KEY idx_notifications_device_id (device_id),
  KEY idx_notifications_type (type),
  KEY idx_notifications_severity (severity),
  KEY idx_notifications_read_at (read_at),
  KEY idx_notifications_created_at (created_at),
  CONSTRAINT fk_notifications_workspace FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON UPDATE CASCADE ON DELETE CASCADE,
  CONSTRAINT fk_notifications_user FOREIGN KEY (user_id) REFERENCES users(id) ON UPDATE CASCADE ON DELETE SET NULL,
  CONSTRAINT fk_notifications_device FOREIGN KEY (device_id) REFERENCES devices(id) ON UPDATE CASCADE ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO roles (name, created_at, updated_at)
VALUES
  ('SUPER_ADMIN', NOW(3), NOW(3)),
  ('ADMIN', NOW(3), NOW(3)),
  ('USER', NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE name = VALUES(name);

INSERT INTO workspaces (name, slug, created_at, updated_at)
VALUES ('Default Workspace', 'default', NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE name = VALUES(name);
