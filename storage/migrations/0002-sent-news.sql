-- +migrate Up

CREATE TABLE IF NOT EXISTS `sent_news`
(
    `id`         VARCHAR(255) NOT NULL,
    `created_at` datetime     NOT NULL DEFAULT current_timestamp(),
    PRIMARY KEY (`id`),
    KEY `created_at` (`created_at`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

INSERT IGNORE INTO `sent_news` (`id`)
SELECT `value`
FROM `system`
WHERE `key` = 'last_entry'
  AND `value` <> '';

DROP TABLE `system`;
