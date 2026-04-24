package mocks

import (
	"time"
)

type EmptyTempKeyValueService struct{}

func NewEmptyTempKeyValueService() *EmptyTempKeyValueService {
    return &EmptyTempKeyValueService{}
}

func (m *EmptyTempKeyValueService) Get(storeName string, key string, ptr any) bool {
    return false
}
func (m *EmptyTempKeyValueService) Set(storeName string, key string, value any, expiresAt time.Time) {
}
func (m *EmptyTempKeyValueService) Delete(storeName string, key string) {
}
func (m *EmptyTempKeyValueService) Prune(storeName string) {
}
func (m *EmptyTempKeyValueService) PruneAll() {
}
