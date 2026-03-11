package keyvalue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"reflect"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/keyvalue"
)

type Registry struct {
	definitions map[string]*Definition
	App         *common.App
}

type Definition struct {
	Name          string
	Type          any
	Init          func() any
	reflectedType reflect.Type
}

func NewRegistry(app *common.App) *Registry {
	return &Registry{
		definitions: make(map[string]*Definition),
		App:         app,
	}
}
func (registry *Registry) Register(definition *Definition) {
	if _, exists := registry.definitions[definition.Name]; exists {
		log.Fatalf("key/value definition with ID \"%s\" already exists", definition.Name)
	}
	prepareDefinition(definition)
	registry.definitions[definition.Name] = definition
}
func prepareDefinition(definition *Definition) {
	if definition.Name == "" {
		log.Fatalf("key/value definition name cannot be empty")
	}
	if definition.Type == nil {
		log.Fatalf("key/value definition %s type cannot be nil", definition.Name)
	}
	if definition.Init == nil {
		log.Fatalf("key/value definition %s Init function cannot be nil", definition.Name)
	}

	definition.reflectedType = reflect.TypeOf(definition.Type)
}

func (registry *Registry) Get(name string, ptr any, ctx context.Context) common.WrappedError {
	definition, exists := registry.definitions[name]
	if !exists {
		return ErrWrapperGetValue.Wrap(ErrUnknownName)
	}
	ptrType := reflect.TypeOf(ptr)
	ptrKind := ptrType.Kind()
	if ptrKind != reflect.Pointer || ptrType.Elem() != definition.reflectedType {
		return ErrWrapperGetValue.Wrap(ErrWrongPointerType)
	}

	tx := ent.TxFromContext(ctx)
	if tx == nil {
		return ErrWrapperGetValue.Wrap(common.ErrNoTxInContext)
	}

	valueOb, stdErr := tx.KeyValue.Query().
		Where(keyvalue.Key(name)).
		Only(ctx)
	if stdErr != nil {
		if ent.IsNotFound(stdErr) {
			newValue := definition.Init()
			newType := reflect.TypeOf(newValue)
			if newType != definition.reflectedType {
				return ErrWrapperGetValue.Wrap(
					ErrWrapperInitInvalidType.Wrap(
						fmt.Errorf(
							"init function for definition %s returned %s instead of %s",
							definition.Name,
							newValue,
							definition.reflectedType,
						),
					),
				)
			}

			reflect.ValueOf(ptr).Elem().Set(reflect.ValueOf(newValue))
			return nil
		}
		return ErrWrapperGetValue.Wrap(common.ErrWrapperDatabase.Wrap(stdErr))
	}
	stdErr = json.Unmarshal(valueOb.Value, ptr)
	if stdErr != nil {
		return ErrWrapperGetValue.Wrap(ErrWrapperDecode.Wrap(stdErr))
	}
	return nil
}
func (registry *Registry) Set(name string, value any, ctx context.Context) common.WrappedError {
	definition, exists := registry.definitions[name]
	if !exists {
		return ErrWrapperSetValue.Wrap(ErrUnknownName)
	}
	if reflect.TypeOf(value) != definition.reflectedType {
		return ErrWrapperSetValue.Wrap(ErrWrongPointerType)
	}

	tx := ent.TxFromContext(ctx)
	if tx == nil {
		return ErrWrapperSetValue.Wrap(common.ErrNoTxInContext)
	}

	encoded, stdErr := json.Marshal(value)
	if stdErr != nil {
		return ErrWrapperSetValue.Wrap(ErrWrapperEncode.Wrap(stdErr))
	}

	now := registry.App.Clock.Now()
	stdErr = tx.KeyValue.Create().
		SetCreatedAt(now).
		SetUpdatedAt(now).
		SetKey(name).
		SetValue(encoded).
		OnConflict().
		UpdateUpdatedAt().
		UpdateValue().
		Exec(ctx)
	if stdErr != nil {
		return ErrWrapperSetValue.Wrap(common.ErrWrapperDatabase.Wrap(stdErr))
	}
	return nil
}
func (registry *Registry) InitAll(ctx context.Context) common.WrappedError {
	tx := ent.TxFromContext(ctx)
	if tx == nil {
		return ErrWrapperInitAll.Wrap(common.ErrNoTxInContext)
	}

	for _, definition := range registry.definitions {
		exists, stdErr := tx.KeyValue.Query().
			Where(keyvalue.Key(definition.Name)).
			Exist(ctx)
		if stdErr != nil {
			return ErrWrapperInitAll.Wrap(common.ErrWrapperDatabase.Wrap(stdErr))
		}
		if exists {
			continue
		}
		newValue := definition.Init()
		newType := reflect.TypeOf(newValue)
		if newType != definition.reflectedType {
			return ErrWrapperInitAll.Wrap(
				ErrWrapperInitInvalidType.Wrap(
					fmt.Errorf(
						"init function for definition %s returned %s instead of %s",
						definition.Name,
						newValue,
						definition.reflectedType,
					),
				),
			)
		}
		encoded, stdErr := json.Marshal(newValue)
		if stdErr != nil {
			return ErrWrapperInitAll.Wrap(ErrWrapperEncode.Wrap(stdErr))
		}

		now := registry.App.Clock.Now()
		stdErr = tx.KeyValue.Create().
			SetCreatedAt(now).
			SetUpdatedAt(now).
			SetKey(definition.Name).
			SetValue(encoded).
			Exec(ctx)
		if stdErr != nil {
			return ErrWrapperInitAll.Wrap(common.ErrWrapperDatabase.Wrap(stdErr))
		}
	}
	return nil
}
