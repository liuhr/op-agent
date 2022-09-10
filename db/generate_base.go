package db

// generateSQLBase is lists of SQL statements required to build the  backend db

var generateSQLBase = []string{
	`
		CREATE TABLE IF NOT EXISTS server_health (
		  hostname varchar(128) CHARACTER SET ascii NOT NULL,
		  token varchar(128) NOT NULL,
		  ip char(200) NOT NULL,
		  http_port int(11) NOT NULL,
		  raft_port int(11) NOT NULL,
		  last_seen_active timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
		  extra_info varchar(128) CHARACTER SET utf8 NOT NULL,
		  command varchar(128) CHARACTER SET utf8 NOT NULL,
		  app_version varchar(64) NOT NULL DEFAULT '',
		  first_seen_active timestamp NOT NULL DEFAULT '1971-01-01 00:00:00',
		  db_backend varchar(255) NOT NULL DEFAULT '',
		  incrementing_indicator bigint(20) NOT NULL DEFAULT '0',
		  PRIMARY KEY (hostname, http_port),
		  KEY last_seen_active_idx (last_seen_active)
		) ENGINE=InnoDB DEFAULT CHARSET=ascii
	`,
	`
		 CREATE TABLE IF NOT EXISTS server_health_history (
  			history_id bigint unsigned NOT NULL AUTO_INCREMENT,
			hostname varchar(128) CHARACTER SET ascii COLLATE ascii_general_ci NOT NULL,
  			token varchar(128) NOT NULL,
  			first_seen_active timestamp NOT NULL,
  			extra_info varchar(128) CHARACTER SET utf8 COLLATE utf8_general_ci NOT NULL,
  			command varchar(128) CHARACTER SET utf8 COLLATE utf8_general_ci NOT NULL,
  			app_version varchar(64) CHARACTER SET ascii COLLATE ascii_general_ci NOT NULL DEFAULT '',
  			PRIMARY KEY (history_id),
  			UNIQUE KEY hostname_token_idx_server_health_history (hostname,token),
  			KEY first_seen_active_idx_server_health_history (first_seen_active)
		) ENGINE=InnoDB  DEFAULT CHARSET=ascii
	`,
}
