// Copyright (c) 2016 Nicolas Martyanoff <khaelin@gmail.com>
//
// Permission to use, copy, modify, and distribute this software for any
// purpose with or without fee is hereby granted, provided that the above
// copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
// WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
// ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
// WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
// ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
// OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

package main

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/mmcdole/gofeed"
)

type Post struct {
	Id      int64
	GUID    string
	URL     string
	FeedId  int64
	Date    time.Time
	Title   string
	Author  string
	Content string
	Enabled bool
}

type PostList []*Post

func (p *Post) Key() string {
	if p.GUID != "" {
		return p.GUID
	}

	return p.URL
}

func (p *Post) Insert(tx *sql.Tx) error {
	var date int64
	if p.Date.IsZero() {
		date = 0
	} else {
		date = p.Date.UTC().Unix()
	}

	res, err := tx.Exec(
		`INSERT INTO posts (guid, url, feed, date, title, author,
		                    content, enabled)
		   VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		p.GUID, p.URL, p.FeedId, date, p.Title, p.Author,
		p.Content, p.Enabled)
	if err != nil {
		return fmt.Errorf("cannot insert post: %v", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("cannot retrieve post id: %v", err)
	}

	p.Id = id
	return nil
}

func (p *Post) Update(tx *sql.Tx) error {
	var date int64
	if p.Date.IsZero() {
		date = 0
	} else {
		date = p.Date.UTC().Unix()
	}

	_, err := tx.Exec(
		`UPDATE posts SET
		     guid = ?,
		     url = ?,
		     feed = ?,
		     date = ?,
		     title = ?,
		     author = ?,
		     content = ?,
		     enabled = ?
		   WHERE id = ?`,
		p.GUID, p.URL, p.FeedId, date, p.Title, p.Author,
		p.Content, p.Enabled, p.Id)
	if err != nil {
		return fmt.Errorf("cannot update post: %v", err)
	}

	return nil
}

func (p *Post) ReadFromGofeedItem(item *gofeed.Item) {
	p.GUID = item.GUID
	p.URL = item.Link

	if item.PublishedParsed != nil {
		p.Date = item.PublishedParsed.UTC()
	} else if item.UpdatedParsed != nil {
		p.Date = item.UpdatedParsed.UTC()
	}

	p.Title = item.Title

	if item.Author != nil {
		p.Author = item.Author.Name
	}

	if item.Content != "" {
		p.Content = item.Content
	} else {
		p.Content = item.Description
	}
}

func (p *Post) ReadFromRow(row *sql.Rows) error {
	var date int64

	err := row.Scan(&p.Id, &p.GUID, &p.URL, &p.FeedId, &date, &p.Title,
		&p.Author, &p.Content, &p.Enabled)
	if err != nil {
		return err
	}

	p.Date = time.Unix(date, 0).UTC()

	return nil
}

func (pl *PostList) LoadRange(tx *sql.Tx, count int, offset int) error {
	rows, err := tx.Query(
		`SELECT p.id, p.guid, p.url, p.feed, p.date, p.title, p.author,
		        p.content, p.enabled
		   FROM posts AS p
		   INNER JOIN feeds AS f ON f.id = p.feed
		   WHERE f.enabled = 1 AND p.enabled = 1
		   ORDER BY date DESC
		   LIMIT ? OFFSET ?`, count, offset)
	if err != nil {
		return fmt.Errorf("cannot load posts: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		p := &Post{}
		if err := p.ReadFromRow(rows); err != nil {
			return fmt.Errorf("invalid post: %v", err)
		}

		*pl = append(*pl, p)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("cannot load posts: %v", err)
	}

	return nil
}

func (pl *PostList) LoadByFeed(tx *sql.Tx, feedId int64) error {
	rows, err := tx.Query(
		`SELECT id, guid, url, feed, date, title, author, content,
		        enabled
		   FROM posts
		   WHERE feed = ?`, feedId)
	if err != nil {
		return fmt.Errorf("cannot load posts: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		p := &Post{}
		if err := p.ReadFromRow(rows); err != nil {
			return fmt.Errorf("invalid post: %v", err)
		}

		*pl = append(*pl, p)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("cannot load posts: %v", err)
	}

	return nil
}

func (pl *PostList) DeleteByFeed(tx *sql.Tx, feedId int64) error {
	_, err := tx.Exec(`DELETE FROM posts WHERE feed = ?`, feedId)
	if err != nil {
		return fmt.Errorf("cannot delete posts: %v", err)
	}

	return nil
}

func (pl *PostList) Diff(newPosts PostList) (PostList, PostList) {
	table := make(map[string]*Post)
	for _, p := range *pl {
		table[p.Key()] = p
	}

	var new, updated PostList

	for _, newPost := range newPosts {
		p, found := table[newPost.Key()]
		if !found {
			// Add new post
			new = append(new, newPost)
			continue
		}

		// Update existing post
		diff := false
		diff = diff || (p.GUID != newPost.GUID)
		diff = diff || (p.URL != newPost.URL)
		diff = diff || (p.Date != newPost.Date)
		diff = diff || (p.Title != newPost.Title)
		diff = diff || (p.Author != newPost.Author)
		diff = diff || (p.Content != newPost.Content)

		p.GUID = newPost.GUID
		p.URL = newPost.URL
		p.Date = newPost.Date
		p.Title = newPost.Title
		p.Author = newPost.Author
		p.Content = newPost.Content

		if diff {
			updated = append(updated, p)
		}
	}

	return new, updated
}

func CountPosts(tx *sql.Tx) (int, error) {
	row := tx.QueryRow(
		`SELECT count(*)
		   FROM posts AS p
		   LEFT JOIN feeds AS f ON f.id = p.feed
		   WHERE f.enabled = 1 AND p.enabled = 1`)

	var count int
	if err := row.Scan(&count); err != nil {
		return -1, fmt.Errorf("cannot count posts: %v", err)
	}

	return count, nil
}
