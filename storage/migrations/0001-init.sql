-- +migrate Up

CREATE TABLE IF NOT EXISTS `subscribers`
(
    `id`         bigint(20) NOT NULL,
    `created_at` datetime   NOT NULL DEFAULT current_timestamp(),
    PRIMARY KEY (`id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE IF NOT EXISTS `system`
(
    `key`   VARCHAR(20)  NOT NULL,
    `value` VARCHAR(255) NOT NULL,
    PRIMARY KEY (`key`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

INSERT INTO `system` (`key`, `value`)
VALUES ('last_entry', '');
