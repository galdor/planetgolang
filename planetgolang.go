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
	"log"
	"os"
	"path"

	"github.com/galdor/go-cmdline"
)

func main() {
	cmdline := cmdline.New()

	cmdline.AddOption("d", "db", "file", "load a sqlite database")
	if Production {
		cmdline.SetOptionDefault("db",
			path.Join(DbDir, "/planetgolang.db"))
	} else {
		cmdline.SetOptionDefault("db", "./planetgolang.db")
	}

	cmdline.AddCommand("help", "print help and exit")
	cmdline.AddCommand("add-feed", "add a new feed")
	cmdline.AddCommand("update", "update all feeds")
	cmdline.AddCommand("generate", "generate the website")

	cmdline.Parse(os.Args)

	cmd := cmdline.CommandName()
	args := cmdline.CommandArgumentsValues()

	var fun func([]string, *DB)

	switch cmd {
	case "help":
		cmdline.PrintUsage(os.Stdout)
		os.Exit(0)
	case "add-feed":
		fun = CLICmdAddFeed
	case "update":
		fun = CLICmdUpdate
	case "generate":
		fun = CLICmdGenerate
	}

	log.SetFlags(log.Ltime)

	dbPath := cmdline.OptionValue("db")

	db := &DB{}
	if err := db.Open(dbPath); err != nil {
		log.Fatalf("cannot open database: %v", err)
	}

	arg0 := fmt.Sprintf("%s %s", os.Args[0], cmd)
	fun(append([]string{arg0}, args...), db)

	db.Close()
}

func CLICmdAddFeed(args []string, db *DB) {
	// Options
	cmdline := cmdline.New()

	cmdline.AddOption("a", "author", "name", "the author of the feed")

	cmdline.AddArgument("url", "the url of the feed")

	cmdline.Parse(args)

	// Create the feed
	url := cmdline.ArgumentValue("url")
	author := cmdline.OptionValue("author")

	feed := &Feed{
		URL:     url,
		Author:  author,
		Enabled: true,
	}

	if err := feed.Download(); err != nil {
		log.Fatalf("%v", err)
	}

	feed.ExtractMetadata()

	if feed.Title == "" {
		log.Fatalf("missing feed title")
	}
	if feed.Author == "" {
		log.Fatalf("missing feed author")
	}
	if feed.WebsiteURL == "" {
		log.Fatalf("missing feed website url")
	}

	if err := db.WithTx(feed.Insert); err != nil {
		log.Fatalf("%v", err)
	}
}

func CLICmdUpdate(args []string, db *DB) {
	var feeds FeedList
	if err := db.WithTx(feeds.LoadEnabled); err != nil {
		log.Fatalf("%v", err)
	}

	log.Printf("%d feeds loaded", len(feeds))

	// TODO parallelize
	for _, feed := range feeds {
		log.Printf("updating feed %s", feed.URL)

		if err := feed.Download(); err != nil {
			log.Printf("error: %v", err)
			continue
		}

		// Update feed metadata
		feed.ExtractMetadata()

		if err := db.WithTx(feed.Update); err != nil {
			log.Printf("error: %v", err)
		}

		// Load posts and merge new ones
		var posts PostList
		err := db.WithTx(func(tx *sql.Tx) error {
			return posts.LoadByFeed(tx, feed.Id)
		})
		if err != nil {
			log.Printf("error: %v", err)
			continue
		}

		newPosts := feed.ExtractPosts()
		oldNbPosts := len(posts)
		posts.Merge(newPosts)

		// Update posts
		err = db.WithTx(func(tx *sql.Tx) error {
			if err := posts.DeleteByFeed(tx, feed.Id); err != nil {
				return err
			}

			for _, post := range posts {
				if err := post.Insert(tx); err != nil {
					return err
				}
			}

			return nil
		})
		if err != nil {
			log.Printf("error: %v", err)
			continue
		}

		log.Printf("%d new posts", len(posts)-oldNbPosts)
	}
}

func CLICmdGenerate(args []string, db *DB) {
	// Options
	cmdline := cmdline.New()

	cmdline.AddOption("", "analytics-id", "id",
		"the google analytics identifier")
	cmdline.AddOption("", "share-dir", "path",
		"the directory containing data files")
	if Production {
		cmdline.SetOptionDefault("share-dir", ShareDir)
	} else {
		cmdline.SetOptionDefault("share-dir", ".")
	}

	cmdline.AddArgument("output", "the output directory")

	cmdline.Parse(args)

	// Generate the website
	outputDirPath := cmdline.ArgumentValue("output")
	log.Printf("generating website in %s", outputDirPath)

	gen := NewGenerator()
	gen.AnalyticsId = cmdline.OptionValue("analytics-id")
	gen.ShareDirPath = cmdline.OptionValue("share-dir")
	gen.OutputDirPath = outputDirPath

	err := db.WithTx(func(tx *sql.Tx) error {
		return gen.Generate(tx)
	})
	if err != nil {
		log.Fatalf("%v", err)
	}

}
