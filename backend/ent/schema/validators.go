package schema

import "fmt"

type EncryptedValidator[T any] = func(val T) error

type HasLength interface {
	~string | *string | ~[]any | *[]any
}

func getLength[T HasLength](val T) int {
	switch v := any(val).(type) {
	case string:
		return len(v)
	case *string:
		if v == nil {
			return -1
		}
		return len(*v)
	case []any:
		return len(v)
	case *[]any:
		if v == nil {
			return -1
		}
		return len(*v)
	default:
		panic(fmt.Sprintf("getLength: unexpected type %T", val))
	}
}

func MinLen[T HasLength](n int) EncryptedValidator[T] {
	return func(val T) error {
		l := getLength(val)
		if l != -1 && l < n {
			return fmt.Errorf("EncryptedValidator: value is less than the required length of %d", n)
		}
		return nil
	}
}
func MaxLen[T HasLength](n int) EncryptedValidator[T] {
	return func(val T) error {
		l := getLength(val)
		if l != -1 && l > n {
			return fmt.Errorf("EncryptedValidator: value is greater than the allowed length of %d", n)
		}
		return nil
	}
}
