package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/ent" // Note: will have to reorganise if I end up needing to use the common module in schemas
	"modernc.org/sqlite"
)

// TODO: rename to ErrType1 and ErrType2?
const (
	// General categories
	ErrTypeDatabase = "database [general]"
	ErrTypeTotp     = "totp [general]"
	ErrTypeTimeout  = "timeout [general]"
	ErrTypeNetwork  = "network [general]"
	ErrTypeDisk     = "disk [general]"
	ErrTypeMemory   = "memory [general]"
	ErrTypeAPI      = "api [general]"
	ErrTypeClient   = "client [general]"
	// If there's no applicable general category, there should be no [general] category on the error.
	// Functions that return a category should return an empty string
	// But you might use this to maintain the hierarchy
	ErrTypeOther = "other"

	// Package categories
	ErrTypeCommon          = "common [package]"
	ErrTypeSetup           = "setup [package]"
	ErrTypeCore            = "core [package]"
	ErrTypeJobs            = "jobs [package]"
	ErrTypeTwoFactorAction = "two factor action [package]"
	ErrTypeMessengers      = "messengers [package]"
	ErrTypeRateLimiting    = "rate limiting [package]"
	ErrTypeDbCommon        = "db common [package]"
	ErrTypeServerCommon    = "server common [package]"
	ErrTypeServices        = "services [package]"
	// Similar idea here if it's unknown
)

func HasErrors(errs []error) bool {
	for _, err := range errs {
		if err != nil {
			return true
		}
	}
	return false
}

func GetSuccessfulActionIDs(actionIDs []string, errs []*ErrWithStrId) []string {
	successfulActionIDs := make([]string, len(actionIDs))
	copy(successfulActionIDs, actionIDs)

	for _, err := range errs {
		index := slices.Index(successfulActionIDs, err.Id)
		if index != -1 {
			successfulActionIDs = slices.Delete(successfulActionIDs, index, index+1)
		}
	}
	return successfulActionIDs
}

type Error struct {
	err error
	// Shouldn't be used by the program, this is only to improve the logs
	errType                reflect.Type
	categories             []string
	errDuplicatesCategory  bool
	maxRetries             int
	retryBackoffBase       time.Duration
	retryBackoffMultiplier float64
	debugValues            []DebugValue
}
type DebugValue struct {
	Name    string `json:"name"`
	Message string `json:"message"`
	Value   any    `json:"value"`
}
type WrappedError interface {
	error
	json.Marshaler
	Unwrap() error
	// No `Is` method because we just want to compare the unwrapped errors,
	// it doesn't really matter if the categories are different

	// StandardError isn't needed because you can't accidentally wrap a nil in an interface once it's already one
	// common.Error still has this though since it's a concrete type

	CommonError() *Error
	CloneAsWrappedError() WrappedError

	Categories() []string
	ErrDuplicatesCategory() bool
	GeneralCategory() string
	HighestCategory() string
	LowestCategory() string
	HasCategories(requiredCategories ...string) bool

	AddCategoriesMut(categories ...string)
	RemoveHighestCategoryMut()
	RemoveLowestCategoryMut()

	MaxRetries() int
	RetryBackoffBase() time.Duration
	RetryBackoffMultiplier() float64
	ConfigureRetriesMut(maxRetries int, baseBackoff time.Duration, backoffMultiplier float64)
	DisableRetriesMut()
	SetMaxRetriesMut(value int)
	SetRetryBackoffBaseMut(value time.Duration)
	SetRetryBackoffMultiplierMut(value float64)

	Dump() string
	DebugValues() []DebugValue
	AddDebugValuesMut(values ...DebugValue)
}

func (commErr *Error) Error() string {
	reversedCategories := slices.Clone(commErr.categories)
	slices.Reverse(reversedCategories) // Highest to lowest level
	if commErr.errDuplicatesCategory {
		// Ignore the lowest (last) category since it duplicates the error
		reversedCategories = DeleteSliceIndex(reversedCategories, -1)
	}

	var builder strings.Builder
	for _, category := range reversedCategories {
		builder.WriteString(category)
		builder.WriteString(" error: ")
	}
	builder.WriteString(commErr.err.Error())

	return builder.String()
}

// Use when you need to cast to an error interface and the *Error might be nil
//
// Otherwise you'll get a non-nil error interface that panics when you try to use it
func (commErr *Error) StandardError() error {
	if commErr == nil {
		return nil
	}
	return commErr
}
func (commErr *Error) WrappedError() WrappedError {
	if commErr == nil {
		return nil
	}
	return commErr
}
func (commErr *Error) CommonError() *Error {
	return commErr
}
func (commErr *Error) Unwrap() error {
	if commErr == nil {
		return nil
	}
	return commErr.err
}
func (commErr *Error) Is(target error) bool {
	// TODO: is this needed?
	// Needed so that errors.Is(err.AddCategory("extra category"), err) == true
	// We don't really care if the properties on this struct are different, only that the underlying error is the same

	if target == nil {
		return false
	}
	targetCommErr, ok := target.(*Error)
	if !ok {
		return false
	}
	return commErr.err == targetCommErr.err
}
func (commErr *Error) MarshalJSON() ([]byte, error) {
	type jsonError struct {
		Error                  string        `json:"error"`
		InnerError             string        `json:"innerError"`
		InnerErrorType         string        `json:"innerErrorType"`
		Categories             []string      `json:"categories"`
		ErrDuplicatesCategory  bool          `json:"errDuplicatesCategory"`
		MaxRetries             int           `json:"maxRetries"`
		RetryBackoffBase       time.Duration `json:"retryBackoffBase"`
		RetryBackoffMultiplier float64       `json:"retryBackoffMultiplier"`
		DebugValues            []DebugValue  `json:"debugValues"`
	}
	return json.Marshal(&jsonError{
		Error:                  commErr.Error(),
		InnerError:             commErr.err.Error(),
		InnerErrorType:         commErr.errType.String(),
		Categories:             commErr.categories,
		ErrDuplicatesCategory:  commErr.errDuplicatesCategory,
		MaxRetries:             commErr.maxRetries,
		RetryBackoffBase:       commErr.retryBackoffBase,
		RetryBackoffMultiplier: commErr.retryBackoffMultiplier,
		DebugValues:            commErr.debugValues,
	})
}
func (commErr *Error) Dump() string {
	message, stdErr := json.MarshalIndent(commErr, "", "  ")
	if stdErr != nil {
		return fmt.Sprintf("common.Error.Dump marshall error:\n%v", stdErr)
	}
	return fmt.Sprintf("common.Error.Dump successful:\n%v", string(message))
}

func (commErr *Error) Categories() []string {
	return slices.Clone(commErr.categories)
}
func (commErr *Error) ErrDuplicatesCategory() bool {
	return commErr.errDuplicatesCategory
}

// Note: this number might not be reached if there are other errors with less or no retries in the WithRetries call.
// And the context could always time out before this number is reached.
//
// -1 means no limit
func (commErr *Error) MaxRetries() int {
	return commErr.maxRetries
}
func (commErr *Error) RetryBackoffBase() time.Duration {
	return commErr.retryBackoffBase
}
func (commErr *Error) RetryBackoffMultiplier() float64 {
	return commErr.retryBackoffMultiplier
}
func (commErr *Error) DebugValues() []DebugValue {
	return slices.Clone(commErr.debugValues)
}

func (commErr *Error) GeneralCategory() string {
	category, _ := GetLastCategoryWithTag(commErr.categories, CategoryTagGeneral)
	return category
}

// requiredCategories is highest to lowest level e.g "auth [package]", "create user", common.ErrTypeDatabase
func (commErr *Error) HasCategories(requiredCategories ...string) bool {
	reversedCategories := slices.Clone(commErr.categories)
	slices.Reverse(reversedCategories) // Highest to lowest level
	return CheckPathPattern(reversedCategories, slices.Concat(requiredCategories, []string{"***"}))
}
func (commErr *Error) HighestCategory() string {
	return commErr.categories[len(commErr.categories)-1]
}
func (commErr *Error) LowestCategory() string {
	return commErr.categories[0]
}
func (commErr *Error) AddCategories(categories ...string) *Error {
	copiedErr := commErr.Clone()
	for _, category := range categories {
		hasPackageTag := slices.Contains(ParseCategoryTags(category), CategoryTagPackage)
		insertIndex := -1
		if !hasPackageTag {
			_, insertIndex = GetLastCategoryWithTag(copiedErr.categories, CategoryTagPackage)
		}

		if insertIndex < 0 {
			if len(copiedErr.categories) == 0 || copiedErr.categories[len(copiedErr.categories)-1] != category {
				copiedErr.categories = append(copiedErr.categories, category)
			}
		} else {
			if copiedErr.categories[insertIndex] != category {
				copiedErr.categories = slices.Insert(copiedErr.categories, insertIndex, category)
			}
		}
	}
	return copiedErr
}
func (commErr *Error) AddCategory(category string) *Error {
	return commErr.AddCategories(category)
}
func (commErr *Error) RemoveHighestCategory() *Error {
	// TODO: should highest category include packages for this and HighestCategory()?
	copiedErr := commErr.Clone()
	catCount := len(copiedErr.categories)
	if catCount != 0 {
		copiedErr.categories = slices.Delete(copiedErr.categories, catCount-1, catCount)
	}

	return copiedErr
}
func (commErr *Error) RemoveLowestCategory() *Error {
	copiedErr := commErr.Clone()
	if len(copiedErr.categories) != 0 {
		copiedErr.categories = slices.Delete(copiedErr.categories, 0, 1)
	}

	return copiedErr
}

func (commErr *Error) ConfigureRetries(maxRetries int, baseBackoff time.Duration, backoffMultiplier float64) *Error {
	copiedErr := commErr.Clone()
	copiedErr.maxRetries = maxRetries
	copiedErr.retryBackoffBase = baseBackoff
	copiedErr.retryBackoffMultiplier = backoffMultiplier

	return copiedErr
}
func (commErr *Error) DisableRetries() *Error {
	return commErr.ConfigureRetries(0, 0, 0)
}
func (commErr *Error) SetMaxRetries(value int) *Error {
	copiedErr := commErr.Clone()
	copiedErr.maxRetries = value

	return copiedErr
}
func (commErr *Error) SetRetryBackoffBase(value time.Duration) *Error {
	copiedErr := commErr.Clone()
	copiedErr.retryBackoffBase = value

	return copiedErr
}
func (commErr *Error) SetRetryBackoffMultiplier(value float64) *Error {
	copiedErr := commErr.Clone()
	copiedErr.retryBackoffMultiplier = value

	return copiedErr
}

func (commErr *Error) AddDebugValue(value DebugValue) *Error {
	return commErr.AddDebugValues(value)
}
func (commErr *Error) AddDebugValues(values ...DebugValue) *Error {
	copiedErr := commErr.Clone()
	copiedErr.debugValues = append(copiedErr.debugValues, values...)

	return copiedErr
}
func (commErr *Error) Clone() *Error {
	if commErr == nil {
		return nil
	}
	copiedErr := *commErr
	copiedErr.categories = slices.Clone(commErr.categories)
	copiedErr.debugValues = slices.Clone(copiedErr.debugValues)

	return &copiedErr
}
func (commErr *Error) CloneAsWrappedError() WrappedError {
	return commErr.Clone()
}

func (commErr *Error) AddCategoriesMut(categories ...string) {
	newErr := commErr.AddCategories(categories...)
	commErr.categories = newErr.categories
}
func (commErr *Error) ConfigureRetriesMut(maxRetries int, baseBackoff time.Duration, backoffMultiplier float64) {
	newErr := commErr.ConfigureRetries(maxRetries, baseBackoff, backoffMultiplier)
	commErr.maxRetries = newErr.maxRetries
	commErr.retryBackoffBase = newErr.retryBackoffBase
	commErr.retryBackoffMultiplier = newErr.retryBackoffMultiplier
}
func (commErr *Error) DisableRetriesMut() {
	newErr := commErr.DisableRetries()
	commErr.maxRetries = newErr.maxRetries
	commErr.retryBackoffBase = newErr.retryBackoffBase
	commErr.retryBackoffMultiplier = newErr.retryBackoffMultiplier
}
func (commErr *Error) SetMaxRetriesMut(value int) {
	newErr := commErr.SetMaxRetries(value)
	commErr.maxRetries = newErr.maxRetries
}
func (commErr *Error) SetRetryBackoffBaseMut(value time.Duration) {
	newErr := commErr.SetRetryBackoffBase(value)
	commErr.retryBackoffBase = newErr.retryBackoffBase
}
func (commErr *Error) SetRetryBackoffMultiplierMut(value float64) {
	newErr := commErr.SetRetryBackoffMultiplier(value)
	commErr.retryBackoffMultiplier = newErr.retryBackoffMultiplier
}
func (commErr *Error) AddDebugValuesMut(values ...DebugValue) {
	newErr := commErr.AddDebugValues(values...)
	commErr.debugValues = newErr.debugValues
}
func (commErr *Error) RemoveHighestCategoryMut() {
	newErr := commErr.RemoveHighestCategory()
	commErr.categories = newErr.categories
}
func (commErr *Error) RemoveLowestCategoryMut() {
	newErr := commErr.RemoveLowestCategory()
	commErr.categories = newErr.categories
}

// categories is lowest to highest level except packages go before their categories
// e.g. "auth [package]", "constraint", common.ErrTypeDatabase, "create profile", "create user"
func NewErrorWithCategories(message string, categories ...string) *Error {
	stdErr := errors.New(message)
	return &Error{
		err:                   stdErr,
		errType:               reflect.TypeOf(stdErr),
		categories:            slices.Concat([]string{message}, categories),
		errDuplicatesCategory: true,
	}
}

// categories is lowest to highest level except packages go before their categories
// e.g. "auth [package]", "constraint", common.ErrTypeDatabase, "create profile", "create user"
func WrapErrorWithCategories(stdErr error, categories ...string) WrappedError {
	// Also use errors.As?
	if stdErr == nil {
		return nil
	}
	//nolint:errorlint // we don't want to unwrap the error and lose details, so instead we'll double wrap it
	wrappedErr, ok := stdErr.(WrappedError)
	if ok {
		wrappedErr = wrappedErr.CloneAsWrappedError()
	} else {
		wrappedErr = &Error{
			err:        stdErr,
			errType:    reflect.TypeOf(stdErr),
			categories: []string{},
		}
	}

	wrappedErr.AddCategoriesMut(categories...)
	return wrappedErr
}

const (
	CategoryTagGeneral = "general"
	CategoryTagPackage = "package"
)

func ParseCategoryTags(category string) []string {
	rawTags := strings.Split(GetStringBetween(category, "[", "]"), ",")

	knownTags := []string{CategoryTagGeneral, CategoryTagPackage}
	tags := []string{}
	for _, rawTag := range rawTags {
		tag := strings.Trim(rawTag, " ")
		if tag == "" {
			continue
		}

		if !slices.Contains(knownTags, tag) {
			panic(fmt.Sprintf("ParseCategoryTags: %v is not a valid tag. category string:\n%v", tag, category))
		}
		tags = append(tags, tag)
	}
	return tags
}
func GetCategoryType(categoryTags []string) string {
	knownCategories := []string{CategoryTagGeneral, CategoryTagPackage}

	for _, tag := range categoryTags {
		if slices.Contains(knownCategories, tag) {
			return tag
		}
	}
	return ""
}
func GetLastCategoryWithTag(categories []string, requiredTag string) (string, int) {
	for i, category := range slices.Backward(categories) {
		tags := ParseCategoryTags(category)
		if slices.Contains(tags, requiredTag) {
			return category, i
		}
	}
	return "", -1
}

func AutoWrapError(err error) WrappedError {
	var wrappedErr WrappedError
	if errors.As(err, &wrappedErr) {
		wrappedErr = wrappedErr.CloneAsWrappedError()
		return wrappedErr
	}

	wrappedErr = WrapErrorWithCategories(err, ErrTypeCommon, "auto wrapped")
	if IsErrorType[*sqlite.Error](err) {
		return ErrWrapperDatabase.Wrap(wrappedErr)
	}
	if ent.IsConstraintError(err) ||
		ent.IsNotFound(err) ||
		ent.IsNotLoaded(err) ||
		ent.IsNotSingular(err) ||
		ent.IsValidationError(err) ||
		errors.Is(err, ent.ErrTxStarted) {
		return ErrWrapperDatabase.Wrap(wrappedErr)
	}

	return wrappedErr
}

type ErrorWrapper interface {
	Wrap(err error) WrappedError
}

//nolint:recvcheck
type ConstantErrorWrapper struct {
	Categories []string
	Child      ErrorWrapper
}

func NewErrorWrapper(categories ...string) *ConstantErrorWrapper {
	return &ConstantErrorWrapper{
		Categories: categories,
	}
}

func (errWrapper *ConstantErrorWrapper) Wrap(stdErr error) WrappedError {
	if errWrapper.Child != nil {
		wrappedErr := errWrapper.Child.Wrap(stdErr)
		wrappedErr.AddCategoriesMut(errWrapper.Categories...)
		return wrappedErr
	} else {
		return WrapErrorWithCategories(stdErr, errWrapper.Categories...)
	}
}
func (errWrapper *ConstantErrorWrapper) HasWrapped(err error) bool {
	if len(errWrapper.Categories) == 0 {
		return false
	}

	var errCategories []string
	{
		var wrappedErr WrappedError
		if !errors.As(err, &wrappedErr) {
			return false
		}
		errCategories = wrappedErr.Categories()
	}

	// Ensure the [package] categories are in the right order
	requiredCategories := errWrapper.Wrap(errors.New("")).Categories() // TODO: cache this?
	requiredIndex := 0
	for _, category := range errCategories {
		requiredCategory := requiredCategories[requiredIndex]
		requiredCategoryHasPackageTag := slices.Contains(ParseCategoryTags(requiredCategory), CategoryTagPackage)
		if category == requiredCategory ||
			(requiredCategoryHasPackageTag && !slices.Contains(ParseCategoryTags(category), CategoryTagPackage)) {
			requiredIndex++
			if requiredIndex >= len(requiredCategories) {
				return true
			}
		} else {
			requiredIndex = 0
		}
	}
	return false
}

func (errWrapper *ConstantErrorWrapper) SetChild(child ErrorWrapper) *ConstantErrorWrapper {
	newErrWrapper := errWrapper.Clone()
	newErrWrapper.Child = child
	return newErrWrapper
}

func (errWrapper ConstantErrorWrapper) Clone() *ConstantErrorWrapper {
	copiedErrWrapper := errWrapper
	copiedErrWrapper.Categories = slices.Clone(copiedErrWrapper.Categories)

	return &copiedErrWrapper
}

type DynamicErrorWrapper struct {
	callback func(err error) WrappedError // Mostly just private so IDEs autocomplete to Wrap instead
}

func NewDynamicErrorWrapper(callback func(err error) WrappedError) *DynamicErrorWrapper {
	return &DynamicErrorWrapper{
		callback: callback,
	}
}
func (errWrapper *DynamicErrorWrapper) Wrap(err error) WrappedError {
	return errWrapper.callback(err)
}

func IsErrorType[T error](err error) bool {
	var target T
	return errors.As(err, &target)
}

func PanicIfError(err error) {
	if err != nil {
		panic(err)
	}
}
