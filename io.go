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
	"fmt"
	"io"
	"os"
)

func CopyFile(ipath, opath string) error {
	r, err := os.Open(ipath)
	if err != nil {
		return fmt.Errorf("cannot open %s: %v", ipath, err)
	}
	defer r.Close()

	w, err := os.Create(opath)
	if err != nil {
		return fmt.Errorf("cannot open %s: %v", opath, err)
	}
	defer w.Close()

	_, err = io.Copy(w, r)
	if err != nil {
		return fmt.Errorf("cannot copy %s to %s: %v", ipath, opath, err)
	}

	return nil
}
