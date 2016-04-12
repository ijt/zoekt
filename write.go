// Copyright 2016 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package codesearch

import (
	"bufio"
	"io"
	"log"
	"sort"
)

var _ = log.Println

type indexTOC struct {
	fileContents contentSection
	fileNames    contentSection
	caseBits     compoundSection
	postings     compoundSection
	newlines     compoundSection
	ngramText    simpleSection

	nameNgramText simpleSection
	namePostings  compoundSection
}

func (t *indexTOC) sections() []section {
	return []section{
		&t.fileContents,
		&t.fileNames,
		&t.newlines,
		&t.ngramText,
		&t.postings,

		&t.nameNgramText,
		&t.namePostings,
	}
}

func (w *writer) writeTOC(toc *indexTOC) {
	for _, s := range toc.sections() {
		s.write(w)
	}
}

func (b *IndexBuilder) Write(out io.Writer) error {
	buffered := bufio.NewWriterSize(out, 1<<20)
	defer buffered.Flush()

	w := &writer{w: buffered}
	toc := indexTOC{}

	toc.fileContents.writeStrings(w, b.files)

	toc.newlines.start(w)
	for _, f := range b.files {
		toc.newlines.addItem(w, toDeltas(newLinesIndices(f.data)))
	}
	toc.newlines.end(w)

	var keys []string
	for k := range b.contentPostings {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	toc.ngramText.start(w)
	for _, k := range keys {
		w.Write([]byte(k))
	}
	toc.ngramText.end(w)

	toc.postings.start(w)
	for _, k := range keys {
		toc.postings.addItem(w, toDeltas(b.contentPostings[k]))
	}
	toc.postings.end(w)

	// names.
	toc.fileNames.writeStrings(w, b.fileNames)

	keys = keys[:0]
	for k := range b.namePostings {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	toc.nameNgramText.start(w)
	for _, k := range keys {
		w.Write([]byte(k))
	}
	toc.nameNgramText.end(w)

	toc.namePostings.start(w)
	for _, k := range keys {
		toc.namePostings.addItem(w, toDeltas(b.namePostings[k]))
	}
	toc.namePostings.end(w)

	var tocSection simpleSection

	tocSection.start(w)
	w.writeTOC(&toc)

	tocSection.end(w)
	tocSection.write(w)
	return w.err
}

func newLinesIndices(in []byte) []uint32 {
	out := make([]uint32, 0, len(in)/30)
	for i, c := range in {
		if c == '\n' {
			out = append(out, uint32(i))
		}
	}
	return out
}