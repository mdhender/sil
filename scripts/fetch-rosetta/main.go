/*
 * SIL - SNOBOL Interpretation Language
 * Copyright (c) 2021, Michael D Henderson
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are met:
 *
 * 1. Redistributions of source code must retain the above copyright notice, this
 *    list of conditions and the following disclaimer.
 *
 * 2. Redistributions in binary form must reproduce the above copyright notice,
 *    this list of conditions and the following disclaimer in the documentation
 *    and/or other materials provided with the distribution.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
 * AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
 * IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 * DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
 * FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
 * DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
 * SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
 * CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
 * OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
 * OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */

// Command fetch-rosetta puts the SNOBOL4 programs named in
// internal/rosetta's manifest into testdata/rosetta.
//
//	go run ./scripts/fetch-rosetta            # everything not already there
//	go run ./scripts/fetch-rosetta -all       # everything, again
//	go run ./scripts/fetch-rosetta -task fizz # the tasks whose name has fizz in it
//
// The programs are RosettaCode's, under CC BY-SA, and are not in this
// repository; testdata/rosetta is gitignored. This is how a checkout
// gets them, the way engines/README.md's curl line is how it gets the
// SIL source.
//
// # Pinning
//
// A task with no Oldid takes whatever revision is current, and the run
// prints the revision and hash it got as a line to paste back into
// internal/rosetta/tasks.go. Read the program first: pinning is the
// record that somebody looked at what arrived and agreed the task's
// expectation still describes it.
//
// A task that is pinned is fetched by revision number, so it comes
// back byte for byte. If it does not hash the same the fetch is
// refused, because the difference is either RosettaCode rewriting
// history or this program's extraction changing, and both are things
// to look at rather than overwrite.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mdhender/sil/internal/corpus"
	"github.com/mdhender/sil/internal/rosetta"
)

// The wiki asks for a User-Agent that says who is calling and gives
// somewhere to complain to.
const agent = "sil-rosetta-fetch/1 (https://github.com/mdhender/sil; SNOBOL4 test corpus)"

// Between requests. Nothing here is in a hurry and the corpus is a
// dozen pages.
const pause = time.Second

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fetch-rosetta: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		dir   = flag.String("dir", "", "write programs here instead of "+rosetta.Dir)
		only  = flag.String("task", "", "only tasks whose name has this in it, without case")
		all   = flag.Bool("all", false, "fetch tasks that are already in the checkout")
		force = flag.Bool("force", false, "write a pinned task whose program no longer hashes the same")
		list  = flag.Bool("list", false, "print the manifest and fetch nothing")
	)
	flag.Parse()

	if bad := rosetta.Validate(rosetta.Tasks); len(bad) != 0 {
		return fmt.Errorf("the manifest has %d problems, and fetching would not fix them:\n%s",
			len(bad), strings.Join(bad, "\n"))
	}

	root, err := corpus.Root()
	if err != nil {
		return err
	}
	into := filepath.Join(root, rosetta.Dir)
	if *dir != "" {
		into = *dir
	}
	if err := os.MkdirAll(into, 0o755); err != nil {
		return err
	}

	var todo []rosetta.Task
	for _, t := range rosetta.Tasks {
		if *only != "" && !strings.Contains(strings.ToLower(t.Name), strings.ToLower(*only)) {
			continue
		}
		todo = append(todo, t)
	}
	if len(todo) == 0 {
		return fmt.Errorf("no task in the manifest has %q in its name", *only)
	}

	if *list {
		for _, t := range todo {
			pin := "unpinned"
			if t.Oldid != 0 {
				pin = fmt.Sprintf("r%d", t.Oldid)
			}
			fmt.Printf("%-28s %-26s %-10s %s\n", t.File, t.Name, pin, rosetta.Page(t))
		}
		return nil
	}

	client := &http.Client{Timeout: 30 * time.Second}
	var pins []string
	var failed int
	for i, t := range todo {
		path := filepath.Join(into, t.File)
		if !*all {
			if _, err := os.Stat(path); err == nil {
				fmt.Printf("%-28s have it\n", t.File)
				continue
			}
		}
		if i > 0 {
			time.Sleep(pause)
		}
		revid, program, err := fetch(client, t)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%-28s %v\n", t.File, err)
			failed++
			continue
		}
		sum := sha256.Sum256([]byte(program))
		hash := hex.EncodeToString(sum[:])

		if t.SHA256 != "" && t.SHA256 != hash {
			if !*force {
				fmt.Fprintf(os.Stderr,
					"%-28s r%d no longer hashes the same: have %s, pinned %s.\n"+
						"%28s Read the difference before -force overwrites it.\n",
					t.File, revid, hash[:12], t.SHA256[:12], "")
				failed++
				continue
			}
			fmt.Fprintf(os.Stderr, "%-28s forced over a pin\n", t.File)
		}
		if err := os.WriteFile(path, []byte(program), 0o644); err != nil {
			return err
		}
		fmt.Printf("%-28s r%d %s %d lines\n", t.File, revid, hash[:12],
			strings.Count(program, "\n"))
		if t.Oldid == 0 || t.SHA256 == "" {
			pins = append(pins, fmt.Sprintf("\t\t// %s\n\t\tOldid:  %d,\n\t\tSHA256: %q,",
				t.Name, revid, hash))
		}
	}

	if len(pins) != 0 {
		fmt.Printf("\nRead what arrived, then pin it in internal/rosetta/tasks.go:\n\n%s\n",
			strings.Join(pins, "\n\n"))
	}
	if failed != 0 {
		return fmt.Errorf("%d of %d tasks did not come back", failed, len(todo))
	}
	return nil
}

// wikitext is what api.php?action=parse&prop=wikitext returns under
// formatversion=2, and the error it returns instead.
type wikitext struct {
	Parse struct {
		Title    string `json:"title"`
		RevID    int    `json:"revid"`
		Wikitext string `json:"wikitext"`
	} `json:"parse"`
	Error struct {
		Code string `json:"code"`
		Info string `json:"info"`
	} `json:"error"`
}

// fetch returns the revision it read and the SNOBOL4 program in it.
func fetch(client *http.Client, t rosetta.Task) (int, string, error) {
	req, err := http.NewRequest("GET", rosetta.API(t), nil)
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("User-Agent", agent)
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return 0, "", err
	}
	if resp.StatusCode != http.StatusOK {
		return 0, "", fmt.Errorf("%s: %s", rosetta.API(t), resp.Status)
	}
	var doc wikitext
	if err := json.Unmarshal(body, &doc); err != nil {
		return 0, "", fmt.Errorf("%s: %w", rosetta.API(t), err)
	}
	if doc.Error.Code != "" {
		return 0, "", fmt.Errorf("%s: %s: %s", t.Name, doc.Error.Code, doc.Error.Info)
	}

	section, err := snobolSection(doc.Parse.Wikitext)
	if err != nil {
		return doc.Parse.RevID, "", err
	}
	program, err := codeBlock(section, t.Block)
	if err != nil {
		return doc.Parse.RevID, "", err
	}
	return doc.Parse.RevID, program, nil
}

// header matches the line RosettaCode opens a language's section with,
// which is a template rather than a plain heading:
//
//	=={{header|SNOBOL4}}==
var header = regexp.MustCompile(`(?mi)^=+\s*\{\{\s*header\s*\|\s*SNOBOL4\s*\}\}\s*=+\s*$`)

// snobolSection returns the SNOBOL4 part of a task's wikitext: from
// its header to the next language's, subsections included. Language
// sections are level two, so a line of three or more equals signs is a
// heading inside this one and does not end it.
func snobolSection(text string) (string, error) {
	loc := header.FindStringIndex(text)
	if loc == nil {
		return "", fmt.Errorf("the page has no SNOBOL4 section")
	}
	rest := text[loc[1]:]
	for i, line := range strings.Split(rest, "\n") {
		trimmed := strings.TrimSpace(line)
		if i > 0 && strings.HasPrefix(trimmed, "==") && !strings.HasPrefix(trimmed, "===") {
			return rest[:offsetOfLine(rest, i)], nil
		}
	}
	return rest, nil
}

// offsetOfLine returns where the nth line of s begins.
func offsetOfLine(s string, n int) int {
	at := 0
	for ; n > 0; n-- {
		i := strings.IndexByte(s[at:], '\n')
		if i < 0 {
			return len(s)
		}
		at += i + 1
	}
	return at
}

// code matches the three ways a program appears in wikitext. The wiki
// moved from <lang> to <syntaxhighlight> and the old markup is still
// on plenty of pages, so both are read; <pre> is the fallback for a
// page that never used either.
var code = regexp.MustCompile(`(?is)<(syntaxhighlight|lang|pre)\b[^>]*>(.*?)</(?:syntaxhighlight|lang|pre)\s*>`)

// codeBlock returns the nth program in a section. A task with more
// than one SNOBOL4 solution gets them in the order the page has them,
// and the manifest's Block says which one the expectation was written
// against.
func codeBlock(section string, n int) (string, error) {
	blocks := code.FindAllStringSubmatch(section, -1)
	if len(blocks) == 0 {
		return "", fmt.Errorf("the SNOBOL4 section has no code in it")
	}
	if n >= len(blocks) {
		return "", fmt.Errorf("the SNOBOL4 section has %d code blocks, and the manifest asks for number %d",
			len(blocks), n)
	}

	program := unescape(blocks[n][2])
	// The markup usually opens with a newline that is not part of the
	// program. A deck is lines, so the end gets exactly one.
	program = strings.TrimLeft(program, "\n")
	program = strings.TrimRight(program, " \t\n") + "\n"
	if strings.TrimSpace(program) == "" {
		return "", fmt.Errorf("code block %d is empty", n)
	}
	return program, nil
}

// unescape undoes the five XML entities, and only those. The general
// HTML unescaping in the standard library also knows the entities that
// may be written without a semicolon -- &copy, &not, &times -- and
// SNOBOL4 keywords are written with an ampersand, so it would be free
// to make &COPY into a copyright sign. Ampersand goes last, so that
// &amp;lt; comes out as &lt; and not as a less-than sign.
func unescape(s string) string {
	for _, pair := range [][2]string{
		{"&lt;", "<"}, {"&gt;", ">"}, {"&quot;", `"`},
		{"&#39;", "'"}, {"&apos;", "'"}, {"&amp;", "&"},
	} {
		s = strings.ReplaceAll(s, pair[0], pair[1])
	}
	return s
}
