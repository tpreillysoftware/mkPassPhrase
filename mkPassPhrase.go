/******************************************************************************
Copyright (c) 2026, Timothy Reilly
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

* Redistributions of source code must retain the above copyright notice, this
  list of conditions and the following disclaimer.

* Redistributions in binary form must reproduce the above copyright notice,
  this list of conditions and the following disclaimer in the documentation
  and/or other materials provided with the distribution.

* Neither the name of mkPassPhrase nor the names of its
  contributors may be used to endorse or promote products derived from
  this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*******************************************************************************/

package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
)

const (
	keyLength = 5
	diceMin   = '1'
	diceMax   = '8'
	wordCount = 8 * 8 * 8 * 8 * 8
)

func main() {
	wordListPath := flag.String("w", "", "path to a custom word list")
	diceKey := flag.String("d", "", "five dice digits, each in the range 1-8")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [-w <path-to-wordlist>] -d [1-8][1-8][1-8][1-8][1-8]\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	if flag.NArg() != 0 {
		exitWithUsage("unexpected positional argument(s)")
	}

	if !validDiceKey(*diceKey) {
		exitWithUsage("-d must be exactly 5 digits, each in the range 1-8")
	}

	words := defaultWordMap
	if *wordListPath != "" {
		importedWords, err := importWordList(*wordListPath)
		if err != nil {
			exitWithError("failed to import word list: %v", err)
		}
		words = importedWords
	}

	word, ok := words[*diceKey]
	if !ok {
		exitWithError("no word found for dice key %q", *diceKey)
	}

	fmt.Println(word)
}

func exitWithUsage(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	flag.Usage()
	os.Exit(2)
}

func exitWithError(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func validDiceKey(key string) bool {
	if len(key) != keyLength {
		return false
	}

	for _, digit := range key {
		if digit < diceMin || digit > diceMax {
			return false
		}
	}

	return true
}

func importWordList(filename string) (map[string]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	words := make(map[string]string, wordCount)
	scanner := bufio.NewScanner(file)

	index := 0
	for scanner.Scan() {
		key, err := keyForIndex(index)
		if err != nil {
			return nil, err
		}

		words[key] = scanner.Text()
		index++
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return words, nil
}

func keyForIndex(index int) (string, error) {
	if index < 0 || index >= wordCount {
		return "", errors.New("word list contains more than 32,768 entries")
	}

	key := [keyLength]byte{}
	for i := keyLength - 1; i >= 0; i-- {
		key[i] = byte(index%8) + diceMin
		index /= 8
	}

	return string(key[:]), nil
}
