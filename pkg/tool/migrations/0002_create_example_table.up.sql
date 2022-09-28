SET search_path TO webhookrss, public;

CREATE TABLE IF NOT EXISTS items (
  id SERIAL NOT NULL PRIMARY KEY,

  feed text CONSTRAINT feed_present CHECK ((feed != '') IS TRUE),

  title text CONSTRAINT title_present CHECK ((title != '') IS TRUE),
  url TEXT NOT NULL,
  body TEXT NOT NULL,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS feed_idx ON items(feed);
