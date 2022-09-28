SET search_path TO webhookrss, public;

DROP INDEX IF EXISTS feed_idx;
DROP TABLE IF EXISTS items;
