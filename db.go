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

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	Path string
	conn *sql.DB
}

func (db *DB) Open(dbPath string) error {
	db.Path = dbPath

	log.Printf("opening database at %s", db.Path)

	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}
	db.conn = conn

	_, err = db.conn.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		return fmt.Errorf("cannot enable foreign keys: %v", err)
	}

	return nil
}

func (db *DB) Close() {
	db.conn.Close()
}

func (db *DB) WithTx(fn func(*sql.Tx) error) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("cannot begin transaction: %v", err)
	}

	if err := fn(tx); err != nil {
		if txerr := tx.Rollback(); txerr != nil {
			log.Panicf("cannot rollback transaction: %v", err)
		}

		return err
	}

	if err := tx.Commit(); err != nil {
		log.Panicf("cannot commit transaction: %v", err)
	}

	return nil
}
