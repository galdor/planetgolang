
BEGIN;

CREATE TABLE feeds(
    id INTEGER PRIMARY KEY,
    url TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    author TEXT NOT NULL,
    website_url TEXT NOT NULL,
    enabled BOOLEAN NOT NULL
);

CREATE TABLE posts(
    id INTEGER PRIMARY KEY,
    guid TEXT NOT NULL,
    url TEXT NOT NULL UNIQUE,
    feed INTEGER NOT NULL REFERENCES feeds(id),
    date INTEGER NOT NULL, -- unix timestamp
    title TEXT NOT NULL,
    author TEXT NOT NULL, -- overrides feeds.author when not empty
    content TEXT NOT NULL,
    enabled BOOLEAN NOT NULL
);

CREATE VIEW v_posts AS
    SELECT id, feed, date(date, "unixepoch") AS datestr, url, title, enabled
      FROM posts
      ORDER BY date DESC;

COMMIT;
