package radio

import (
	"fmt"
	"github.com/digineo/go-uci"
)

// fakeUciTree stubs the uci.Tree interface for testing purposes.
type fakeUciTree struct {
	valuesForGet  map[string]string
	valuesFromSet map[string]string
	committed     bool
}

func (tree fakeUciTree) SetType(config, section, option string, typ uci.OptionType, values ...string) bool {
	if typ != uci.TypeOption {
		panic("not implemented")
	}
	tree.valuesFromSet[fmt.Sprintf("%s.%s.%s", config, section, option)] = values[0]
	return true
}

func (tree fakeUciTree) GetLast(config, section, option string) (string, bool) {
	return tree.valuesForGet[fmt.Sprintf("%s.%s.%s", config, section, option)], true
}

func (tree fakeUciTree) Commit() error {
	tree.committed = true
	return nil
}

func (tree fakeUciTree) LoadConfig(name string, forceReload bool) error {
	panic("not implemented")
}

func (tree fakeUciTree) Revert(configs ...string) {
	panic("not implemented")
}

func (tree fakeUciTree) GetSections(config, secType string) ([]string, bool) {
	panic("not implemented")
}

func (tree fakeUciTree) Get(config, section, option string) ([]string, bool) {
	panic("not implemented")
}

func (tree fakeUciTree) GetBool(config, section, option string) (bool, bool) {
	panic("not implemented")
}

func (tree fakeUciTree) Set(config, section, option string, values ...string) bool {
	panic("not implemented")
}

func (tree fakeUciTree) Del(config, section, option string) {
	panic("not implemented")
}

func (tree fakeUciTree) AddSection(config, section, typ string) error {
	panic("not implemented")
}

func (tree fakeUciTree) DelSection(config, section string) {
	panic("not implemented")
}
