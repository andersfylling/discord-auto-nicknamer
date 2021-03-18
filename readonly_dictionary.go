package nicknamer

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"sync"
)

var ErrAlreadyExists = errors.New("word already exists")
var ErrOutOfNames = errors.New("no more pre-defined names")

type JSONWrapper struct {
	Words []string
	Names []string
}

type ReadOnlyDictionary struct {
	Storage    Storage
	dictionary []string
	names      []string
	sync.Mutex
}

func (dict *ReadOnlyDictionary) ListWords() []string {
	dict.Lock()
	defer dict.Unlock()

	l := make([]string, len(dict.dictionary))
	copy(l, dict.dictionary)
	return l
}

func (dict *ReadOnlyDictionary) ListNames() []string {
	dict.Lock()
	defer dict.Unlock()

	l := make([]string, len(dict.names))
	copy(l, dict.names)
	return l
}

func (dict *ReadOnlyDictionary) RandWords(min, max int) []string {
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

func (dict *ReadOnlyDictionary) PopName() (string, error) {
	dict.Lock()
	defer dict.Unlock()

	if len(dict.names) == 0 {
		return "", ErrOutOfNames
	}

	name := dict.names[0]
	dict.names = dict.names[1:]
	_ = dict.SaveUnsafe()

	return name, nil
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
	sortWrapper2 := sort.StringSlice(wrapper.Names)
	sortWrapper2.Sort()

	dict.Lock()
	defer dict.Unlock()

	dict.dictionary = wrapper.Words
	dict.names = wrapper.Names
}

func (dict *ReadOnlyDictionary) AddGenericUnsafe(list []string, entry string) ([]string, error) {
	pos := sort.SearchStrings(list, entry)
	if pos == len(list) {
		list = append(list, entry)
	} else if list[pos] == entry {
		return nil, ErrAlreadyExists
	} else {
		// insert in sorted order
		list = append(list, "")
		copy(list[pos+1:], list[pos:])
		list[pos] = entry
	}
	return list, nil
}

func (dict *ReadOnlyDictionary) RemoveGenericUnsafe(list *[]string, entry string) {
	remove := func(s []string, i int) []string {
		return append(s[:i], s[i+1:]...)
	}
	pos := sort.SearchStrings(*list, entry)
	if (*list)[pos] == entry {
		*list = remove(*list, pos)
	}
}

func (dict *ReadOnlyDictionary) SaveUnsafe() error {
	data, err := json.Marshal(&JSONWrapper{Words: dict.dictionary, Names: dict.names})
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

func (dict *ReadOnlyDictionary) RemoveWord(word string) error {
	dict.Lock()
	defer dict.Unlock()
	dict.RemoveGenericUnsafe(&dict.dictionary, word)
	return dict.SaveUnsafe()
}

func (dict *ReadOnlyDictionary) RemoveName(name string) error {
	dict.Lock()
	defer dict.Unlock()
	dict.RemoveGenericUnsafe(&dict.names, name)
	return dict.SaveUnsafe()
}

func (dict *ReadOnlyDictionary) AddWord(word string) error {
	dict.Lock()
	defer dict.Unlock()
	l, err := dict.AddGenericUnsafe(dict.dictionary, word)
	if err != nil {
		return err
	}

	dict.dictionary = l
	return dict.SaveUnsafe()
}

func (dict *ReadOnlyDictionary) AddName(name string) error {
	dict.Lock()
	defer dict.Unlock()
	l, err := dict.AddGenericUnsafe(dict.names, name)
	if err != nil {
		return err
	}

	dict.names = l
	return dict.SaveUnsafe()
}
