package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"sync"
)

var ErrAlreadyExists = errors.New("word already exists")

type JSONWrapper struct {
	Words []string
}

type ReadOnlyDictionary struct {
	Storage    Storage
	dictionary []string
	sync.Mutex
}

func (dict *ReadOnlyDictionary) List() []string {
	dict.Lock()
	defer dict.Unlock()

	l := make([]string, len(dict.dictionary))
	copy(l, dict.dictionary)
	return l
}

func (dict *ReadOnlyDictionary) Combination(min, max int) []string {
	rounds := max - min
	length := rand.Intn(rounds)
	l := make([]string, length)

	dict.Lock()
	defer dict.Unlock()

	for i := 0; i < rounds; i++ {
		p := rand.Intn(len(dict.dictionary) - 1)
		l = append(l, dict.dictionary[p])
	}

	return l
}

func (dict *ReadOnlyDictionary) Load() {
	data, err := dict.Storage.Load()
	if err != nil {
		return
	}

	wrapper := &JSONWrapper{}
	_ = json.Unmarshal(data, wrapper)
	sortWrapper := sort.StringSlice(wrapper.Words)
	sortWrapper.Sort()

	dict.Lock()
	defer dict.Unlock()

	dict.dictionary = wrapper.Words
}

func (dict *ReadOnlyDictionary) Add(word string) error {
	dict.Lock()
	defer dict.Unlock()

	pos := sort.SearchStrings(dict.dictionary, word)
	if pos >= len(dict.dictionary) || dict.dictionary[pos] == word {
		return ErrAlreadyExists
	}

	// insert in sorted order
	dict.dictionary = append(dict.dictionary, "")
	copy(dict.dictionary[pos+1:], dict.dictionary[pos:])
	dict.dictionary[pos] = word

	data, err := json.Marshal(&JSONWrapper{dict.dictionary})
	if err != nil {
		return fmt.Errorf("unable to marshal dictionary: %w", err)
	} else if data != nil {
		if err := dict.Storage.Save(data); err != nil {
			return fmt.Errorf("unable to save dictionary: %w", err)
		}
	} else {
		return errors.New("nil data")
	}

	return nil
}
