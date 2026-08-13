CREATE TABLE IF NOT EXISTS `activitypub_key` (
  `id` int(16) NOT NULL AUTO_INCREMENT,
  `private_key` text NOT NULL,
  `public_key` text NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `activitypub_follower` (
  `id` int(16) NOT NULL AUTO_INCREMENT,
  `actor_uri` varchar(500) NOT NULL DEFAULT '',
  `inbox_url` varchar(500) NOT NULL DEFAULT '',
  `shared_inbox` varchar(500) DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_activitypub_follower_actor_uri` (`actor_uri`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `activitypub_delivery` (
  `id` int(16) NOT NULL AUTO_INCREMENT,
  `inbox_url` varchar(500) NOT NULL DEFAULT '',
  `payload` text NOT NULL,
  `status` varchar(20) NOT NULL DEFAULT 'pending',
  `attempts` int(11) NOT NULL DEFAULT 0,
  `next_attempt` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `last_error` text,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_activitypub_delivery_inbox_url` (`inbox_url`),
  KEY `idx_activitypub_delivery_status` (`status`),
  KEY `idx_activitypub_delivery_next_attempt` (`next_attempt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
