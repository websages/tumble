CREATE TABLE IF NOT EXISTS `tags` (
  `id` int(16) NOT NULL AUTO_INCREMENT,
  `tag` varchar(255) NOT NULL DEFAULT '',
  `resource_type` varchar(50) NOT NULL DEFAULT '',
  `resource_id` int(16) NOT NULL DEFAULT 0,
  `created_by` varchar(255) NOT NULL DEFAULT '',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_tags_tag` (`tag`),
  KEY `idx_tags_resource` (`resource_type`, `resource_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
