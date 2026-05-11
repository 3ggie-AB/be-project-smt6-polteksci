CREATE TABLE IF NOT EXISTS users (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(160) NOT NULL,
  email VARCHAR(255) NOT NULL,
  password VARCHAR(255) NOT NULL,
  role VARCHAR(32) NOT NULL DEFAULT 'USER',
  created_at TIMESTAMP NULL,
  PRIMARY KEY (id),
  UNIQUE KEY idx_users_email (email),
  KEY idx_users_role (role)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS sessions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id BIGINT UNSIGNED NOT NULL,
  token TEXT NOT NULL,
  expired_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP NULL,
  PRIMARY KEY (id),
  KEY idx_sessions_user_id (user_id),
  KEY idx_sessions_expired_at (expired_at),
  CONSTRAINT fk_sessions_user FOREIGN KEY (user_id) REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS devices (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(160) NOT NULL,
  ip VARCHAR(64) NOT NULL,
  type ENUM('AP','SERVICE') NOT NULL,
  vendor VARCHAR(120) NULL,
  location VARCHAR(180) NULL,
  status ENUM('ONLINE','OFFLINE','WARNING') NOT NULL DEFAULT 'OFFLINE',
  created_at TIMESTAMP NULL,
  PRIMARY KEY (id),
  KEY idx_devices_name (name),
  KEY idx_devices_ip (ip),
  KEY idx_devices_type (type),
  KEY idx_devices_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS device_status (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  device_id BIGINT UNSIGNED NOT NULL,
  latency DOUBLE NOT NULL DEFAULT 0,
  packet_loss DOUBLE NOT NULL DEFAULT 0,
  cpu_usage DOUBLE NOT NULL DEFAULT 0,
  memory_usage DOUBLE NOT NULL DEFAULT 0,
  last_seen TIMESTAMP NULL,
  PRIMARY KEY (id),
  KEY idx_device_status_device_id (device_id),
  KEY idx_device_status_last_seen (last_seen),
  CONSTRAINT fk_device_status_device FOREIGN KEY (device_id) REFERENCES devices(id) ON UPDATE CASCADE ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS monitoring_configs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  device_id BIGINT UNSIGNED NOT NULL,
  ping_enabled BOOLEAN NOT NULL DEFAULT TRUE,
  tcp_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  ping_interval INT NOT NULL DEFAULT 5,
  tcp_interval INT NOT NULL DEFAULT 30,
  monitored_port INT NOT NULL DEFAULT 0,
  created_at TIMESTAMP NULL,
  PRIMARY KEY (id),
  KEY idx_monitoring_configs_device_id (device_id),
  CONSTRAINT fk_monitoring_configs_device FOREIGN KEY (device_id) REFERENCES devices(id) ON UPDATE CASCADE ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS alerts (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  device_id BIGINT UNSIGNED NOT NULL,
  severity ENUM('INFO','WARNING','CRITICAL') NOT NULL,
  message TEXT NOT NULL,
  status ENUM('ACTIVE','RESOLVED') NOT NULL DEFAULT 'ACTIVE',
  created_at TIMESTAMP NULL,
  PRIMARY KEY (id),
  KEY idx_alerts_device_id (device_id),
  KEY idx_alerts_severity (severity),
  KEY idx_alerts_status (status),
  CONSTRAINT fk_alerts_device FOREIGN KEY (device_id) REFERENCES devices(id) ON UPDATE CASCADE ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS notifications (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id BIGINT UNSIGNED NOT NULL,
  alert_id BIGINT UNSIGNED NOT NULL,
  title VARCHAR(180) NOT NULL,
  message TEXT NOT NULL,
  is_read BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMP NULL,
  PRIMARY KEY (id),
  KEY idx_notifications_user_id (user_id),
  KEY idx_notifications_alert_id (alert_id),
  KEY idx_notifications_is_read (is_read),
  CONSTRAINT fk_notifications_user FOREIGN KEY (user_id) REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE,
  CONSTRAINT fk_notifications_alert FOREIGN KEY (alert_id) REFERENCES alerts(id) ON UPDATE CASCADE ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS activity_logs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id BIGINT UNSIGNED NOT NULL,
  action VARCHAR(160) NOT NULL,
  description TEXT NULL,
  created_at TIMESTAMP NULL,
  PRIMARY KEY (id),
  KEY idx_activity_logs_user_id (user_id),
  KEY idx_activity_logs_action (action),
  CONSTRAINT fk_activity_logs_user FOREIGN KEY (user_id) REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS network_topology (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  source_device_id BIGINT UNSIGNED NOT NULL,
  target_device_id BIGINT UNSIGNED NOT NULL,
  relation_type VARCHAR(120) NOT NULL,
  status VARCHAR(80) NOT NULL,
  created_at TIMESTAMP NULL,
  PRIMARY KEY (id),
  KEY idx_network_topology_source_device_id (source_device_id),
  KEY idx_network_topology_target_device_id (target_device_id),
  KEY idx_network_topology_relation_type (relation_type),
  CONSTRAINT fk_network_topology_source FOREIGN KEY (source_device_id) REFERENCES devices(id) ON UPDATE CASCADE ON DELETE CASCADE,
  CONSTRAINT fk_network_topology_target FOREIGN KEY (target_device_id) REFERENCES devices(id) ON UPDATE CASCADE ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ml_predictions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  device_id BIGINT UNSIGNED NOT NULL,
  prediction_type VARCHAR(120) NOT NULL,
  prediction_value DOUBLE NOT NULL DEFAULT 0,
  confidence_score DOUBLE NOT NULL DEFAULT 0,
  created_at TIMESTAMP NULL,
  PRIMARY KEY (id),
  KEY idx_ml_predictions_device_id (device_id),
  KEY idx_ml_predictions_prediction_type (prediction_type),
  CONSTRAINT fk_ml_predictions_device FOREIGN KEY (device_id) REFERENCES devices(id) ON UPDATE CASCADE ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ml_anomalies (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  device_id BIGINT UNSIGNED NOT NULL,
  anomaly_score DOUBLE NOT NULL DEFAULT 0,
  prediction VARCHAR(255) NOT NULL,
  severity ENUM('WARNING','CRITICAL') NOT NULL,
  created_at TIMESTAMP NULL,
  PRIMARY KEY (id),
  KEY idx_ml_anomalies_device_id (device_id),
  KEY idx_ml_anomalies_severity (severity),
  CONSTRAINT fk_ml_anomalies_device FOREIGN KEY (device_id) REFERENCES devices(id) ON UPDATE CASCADE ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
