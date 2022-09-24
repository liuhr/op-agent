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
	`
		CREATE TABLE IF NOT EXISTS variables (
			variable char(100) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
			value varchar(1000) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
			enable int NOT NULL DEFAULT '1' COMMENT '0: disable 1: enable',
			PRIMARY KEY (variable)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`,
	`
		CREATE TABLE IF NOT EXISTS package_info (
  			id int NOT NULL AUTO_INCREMENT,
  			package_name varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '0',
			deploydir char(100) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '/data/op-agent/src/' COMMENT 'Package deployment directory',
  			package_owner varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '0',
  			md5sum varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  			package_version int NOT NULL DEFAULT '0',
  			package_content longblob,
  			package_size int NOT NULL DEFAULT '0' COMMENT 'pacakge size',
  			package_desc varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '0',
  			ctime bigint NOT NULL DEFAULT '0',
  			utime bigint NOT NULL DEFAULT '0',
  			_timestamp timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  			PRIMARY KEY (id),
			KEY idx_package_name_package_version (package_name,package_version)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`,
	`
		CREATE TABLE IF NOT EXISTS agent_package_info (
  			id int NOT NULL AUTO_INCREMENT,
  			hostname varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  			token varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  			agent_ips varchar(500) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '0',
  			package_name varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '0',
  			package_version int NOT NULL DEFAULT '0',
  			deploydir char(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '/data/op-agent/src/' COMMENT 'Package deployment directory',
  			status tinyint NOT NULL DEFAULT '0' COMMENT '0: Init, 1: running，3: failed',
  			fail_reason varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  			ctime timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  			package_schedule_time datetime NOT NULL DEFAULT '1972-01-01 00:00:00',
  			PRIMARY KEY (id),
  			UNIQUE KEY idx_token_version (token,package_name),
  			KEY idx_agent_ips (agent_ips),
  			KEY idx_hostname (hostname)
		) ENGINE=InnoDB  DEFAULT CHARSET=utf8mb4
	`,
	`
		CREATE TABLE IF NOT EXISTS agent_package_task (
  			id int NOT NULL AUTO_INCREMENT,
  			hostname varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  			token varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  			agent_ips varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '0',
  			package_name varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '0',
  			package_version int NOT NULL DEFAULT '0',
  			deploydir char(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '/data/my-agent/src/' COMMENT 'Package deployment directory',
  			status tinyint NOT NULL DEFAULT '0' COMMENT '0: init, 1: readyToRun, 2: running, 3: succeeded, 4: failed',
  			fail_reason text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  			ctime timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  			package_schedule_time datetime NOT NULL DEFAULT '1972-01-01 00:00:00',
  			PRIMARY KEY (id),
  			UNIQUE KEY udx_token_version (token,package_name),
  			KEY idx_status (status),
  			KEY idx_agent_ips_status (agent_ips(255),status)
		) ENGINE=InnoDB  DEFAULT CHARSET=utf8mb4
	`,
	`
		CREATE TABLE IF NOT EXISTS agent_package_blacklist (
  			id int NOT NULL AUTO_INCREMENT,
  			hostname varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  			agent_ips varchar(500) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '0',
  			ctime timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  			note varchar(1000) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  			PRIMARY KEY (id),
  			KEY idx_hostname (hostname),
  			KEY idx_agent_ips (agent_ips)
		) ENGINE=InnoDB  DEFAULT CHARSET=utf8mb4
	`,
	`
		CREATE TABLE IF NOT EXISTS jobs (
  			id bigint unsigned NOT NULL AUTO_INCREMENT,
  			jobname varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'job name',
  			command varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'shell or python or other command',
  			cronexpr char(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'cron expression',
  			oncejob int NOT NULL DEFAULT '0' COMMENT '0: not once job, 1: once job',
  			timeout int NOT NULL DEFAULT '0' COMMENT 'command timeout 0: unlimited',
  			synflag smallint NOT NULL DEFAULT '0' COMMENT '1:synchronous 0:asynchronous',
  			whiteips text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT 'which machines are running on',
  			blackips text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT 'which machines are not running on',
  			killFlag int NOT NULL DEFAULT '0' COMMENT '1: will kill, 0: will not kill',
  			cpushares int NOT NULL DEFAULT '128' COMMENT 'cgroup cpu-shares',
  			cpuquotaus int NOT NULL DEFAULT '0' COMMENT 'cgroup cpu.cfs_quota_us',
  			memorylimit int NOT NULL DEFAULT '0' COMMENT 'cgroup memory.limit_in_bytes',
  			memoryswlimit int NOT NULL DEFAULT '0' COMMENT 'cgroup memory.memsw.limit_in_bytes',
  			ioreadlimit int NOT NULL DEFAULT '0' COMMENT 'cgroup blkio.throttle.read_bps_device',
  			iowritelimit int NOT NULL DEFAULT '0' COMMENT 'cgroup blkio.throttle.write_bps_device',
  			iolimitdevice char(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '/dev/sdb' COMMENT 'Devices that restrict disk IO',
  			add_time datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'add time',
  			_timestamp timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			enabled int NOT NULL DEFAULT '1' COMMENT '1: enabled  0:uenabled',
  			PRIMARY KEY (id),
  			UNIQUE KEY name (jobname)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`,
	`
		CREATE TABLE IF NOT EXISTS oncejobtask (
  			id bigint unsigned NOT NULL AUTO_INCREMENT,
  			hostname varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  			token varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  			ip char(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  			jobname varchar(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'job name',
  			command varchar(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'shell or python or other command',
  			version int NOT NULL DEFAULT '0' COMMENT 'job version',
  			hasrun int NOT NULL DEFAULT '0' COMMENT '0: once job has not been running, 1: once job has been running',
  			status tinyint NOT NULL DEFAULT '0' COMMENT '0: Init, 1: running, 2: done',
  			starttime datetime DEFAULT NULL COMMENT 'cmd start time',
  			endtime datetime DEFAULT NULL COMMENT 'cmd end time',
  			output text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT 'cmd result',
  			err varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'errors',
  			add_time datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'add time',
  			_timestamp timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  			enabled int NOT NULL DEFAULT '1' COMMENT '1: enabled  0:uenabled',
  			PRIMARY KEY (id),
  			UNIQUE KEY name (token,jobname,version),
  			KEY idx_hostname (hostname),
  			KEY idx_ip (ip)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`,
	`
		CREATE TABLE IF NOT EXISTS joblogs (
  			id bigint unsigned NOT NULL AUTO_INCREMENT,
  			hostname varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  			token varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  			ip char(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  			jobname varchar(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'job name',
  			command varchar(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'shell or python or other command',
  			version int NOT NULL DEFAULT '0' COMMENT 'job version',
  			plantime datetime DEFAULT NULL COMMENT 'plan time',
  			scheduletime datetime DEFAULT NULL COMMENT 'schedule time',
  			starttime datetime DEFAULT NULL COMMENT 'cmd start time',
  			endtime datetime DEFAULT NULL COMMENT 'cmd end time',
  			output text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT 'cmd result',
  			err varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'errors',
  			_timestamp timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  			PRIMARY KEY (id),
  			KEY idx_jobname (jobname),
  			KEY idx_command (command),
  			KEY idx_timestamp (_timestamp),
  			KEY idx_hostname (hostname),
  			KEY idx_ip (ip),
  			KEY idx_token (token)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`,
	`
		CREATE TABLE IF NOT EXISTS node_health (
  			hostname varchar(128) CHARACTER SET ascii COLLATE ascii_general_ci NOT NULL,
  			token varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  			ip varchar(2000) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  			http_port int NOT NULL,
  			raft_port int NOT NULL,
  			last_seen_active timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  			extra_info varchar(128) CHARACTER SET utf8 COLLATE utf8_general_ci NOT NULL,
  			command varchar(128) CHARACTER SET utf8 COLLATE utf8_general_ci NOT NULL,
  			app_version varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  			first_seen_active timestamp NOT NULL DEFAULT '1971-01-01 00:00:00',
  			active_flag int NOT NULL DEFAULT '0' COMMENT '0: Inactive, 1: Active',
  			db_backend varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  			incrementing_indicator bigint NOT NULL DEFAULT '0',
  			err varchar(500) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  			PRIMARY KEY (token),
  			KEY idx_ip_port (ip(255),http_port),
  			KEY last_seen_active_idx (last_seen_active),
  			KEY hostname (hostname)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`,
	`
		CREATE TABLE IF NOT EXISTS node_health_history (
  			history_id bigint unsigned NOT NULL AUTO_INCREMENT,
  			hostname varchar(128) CHARACTER SET ascii COLLATE ascii_general_ci NOT NULL,
  			token varchar(128) NOT NULL,
  			first_seen_active timestamp NOT NULL,
  			extra_info varchar(128) CHARACTER SET utf8 COLLATE utf8_general_ci NOT NULL,
  			command varchar(128) CHARACTER SET utf8 COLLATE utf8_general_ci NOT NULL,
  			app_version varchar(64) CHARACTER SET ascii COLLATE ascii_general_ci NOT NULL DEFAULT '',
  			PRIMARY KEY (history_id),
  			UNIQUE KEY hostname_token_idx_node_health_history (hostname,token),
  			KEY first_seen_active_idx_node_health_history (first_seen_active)
		) ENGINE=InnoDB  DEFAULT CHARSET=ascii;
	`,
}
