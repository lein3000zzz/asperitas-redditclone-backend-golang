SET NAMES utf8;
SET time_zone = '+00:00';
SET foreign_key_checks = 0;
SET sql_mode = 'NO_AUTO_VALUE_ON_ZERO';

DROP TABLE IF EXISTS `items`;
DROP TABLE IF EXISTS `users`;
CREATE TABLE `users` (
  `id` varchar(24) NOT NULL,
  `username` varchar(255) NOT NULL,
  `password` varchar(255) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

INSERT INTO `users` (`id`, `username`, `password`) VALUES
("ds32dd31dd33ds32dd31dd33",	'dadadada',	'bebebebe'),
("ds42dd41dd44ds42dd41dd44",	'bebebebe',	'dadadada');
