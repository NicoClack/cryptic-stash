package tempkeyvalue

import (
	"fmt"
	"log"
	"reflect"
	"time"
)

type Registry struct {
	definitions map[string]*Definition
	// It's a bit inefficient to box each value in an interface like this but it's fine for this scale.
	// If more scale was needed, Redis should be used anyway
	stores map[string]*Store[any]
}
type Definition struct {
	Name          string
	Type          any
	reflectedType reflect.Type
}

func NewRegistry() *Registry {
	return &Registry{
		definitions: map[string]*Definition{},
		stores:      map[string]*Store[any]{},
	}
}

func (registry *Registry) Register(definition *Definition) {
	if _, exists := registry.definitions[definition.Name]; exists {
		log.Fatalf("tempkeyvalue store definition with name %q already exists", definition.Name)
	}

	definition.reflectedType = reflect.TypeOf(definition.Type)
	registry.definitions[definition.Name] = definition
	registry.stores[definition.Name] = NewStore[any]()
}

func (registry *Registry) Get(storeName string, key string, ptr any, now time.Time) bool {
	definition, store := registry.mustGetDefinitionAndStore(storeName)

	ptrType := reflect.TypeOf(ptr)
	if ptrType.Kind() != reflect.Pointer || ptrType.Elem() != definition.reflectedType {
		panic(fmt.Sprintf("tempkeyvalue ptr is not a pointer to the correct type for storeName %q", storeName))
	}

	value, exists := store.Get(key, now)
	if !exists {
		return false
	}

	reflect.ValueOf(ptr).Elem().Set(reflect.ValueOf(value))
	return true
}

func (registry *Registry) Set(storeName string, key string, value any, expiresAt time.Time) {
	definition, store := registry.mustGetDefinitionAndStore(storeName)
	if reflect.TypeOf(value) != definition.reflectedType {
		panic(fmt.Sprintf("wrong value type for tempkeyvalue storeName %q", storeName))
	}

	store.Set(key, value, expiresAt)
}

func (registry *Registry) Delete(storeName string, key string) {
	_, store := registry.mustGetDefinitionAndStore(storeName)
	store.Delete(key)
}

func (registry *Registry) Prune(storeName string, now time.Time) {
	_, store := registry.mustGetDefinitionAndStore(storeName)
	store.Prune(now)
}

func (registry *Registry) PruneAll(now time.Time) {
	for _, store := range registry.stores {
		store.Prune(now)
	}
}

func (registry *Registry) mustGetDefinitionAndStore(storeName string) (*Definition, *Store[any]) {
	definition, definitionExists := registry.definitions[storeName]
	if !definitionExists {
		panic(fmt.Sprintf("unknown tempkeyvalue storeName %q", storeName))
	}
	store, storeExists := registry.stores[storeName]
	if !storeExists {
		panic(fmt.Sprintf("missing tempkeyvalue storeName %q, this shouldn't happen!", storeName))
	}

	return definition, store
}
