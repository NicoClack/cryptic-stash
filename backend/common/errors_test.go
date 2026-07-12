package common_test

import (
	"errors"
	"testing"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/stretchr/testify/require"
)

const errTypeTest = "test category [general]"

func TestGetSuccessfulActionIDs_ReturnsCorrectIDs(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name      string
		actionIDs []string
		errs      []*common.ErrWithStrId
		expected  []string
	}{
		{
			"empty",
			[]string{},
			[]*common.ErrWithStrId{},
			[]string{},
		},
		{
			"no errors",
			[]string{"action1", "action2", "action3"},
			[]*common.ErrWithStrId{},
			[]string{"action1", "action2", "action3"},
		},
		{
			"1/3 errors",
			[]string{"action1", "action2", "action3"},
			[]*common.ErrWithStrId{
				{
					Id:  "action1",
					Err: errors.New("error1"),
				},
			},
			[]string{"action2", "action3"},
		},
		{
			"2/3 errors",
			[]string{"action1", "action2", "action3"},
			[]*common.ErrWithStrId{
				{
					Id:  "action1",
					Err: errors.New("error1"),
				},
				{
					Id:  "action3",
					Err: errors.New("error3"),
				},
			},
			[]string{"action2"},
		},
		{
			"all failed",
			[]string{"action1", "action2", "action3"},
			[]*common.ErrWithStrId{
				{
					Id:  "action2",
					Err: errors.New("error2"),
				},
				{
					Id:  "action1",
					Err: errors.New("error1"),
				},
				{
					Id:  "action3",
					Err: errors.New("error3"),
				},
			},
			[]string{},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			ids := common.GetSuccessfulActionIDs(testCase.actionIDs, testCase.errs)
			require.Equal(t, testCase.expected, ids)
		})
	}
}

func TestError_Error_ReturnsCorrectMessage(t *testing.T) {
	t.Parallel()
	sentinelErr := common.NewErrorWithCategories(
		"test error",
	).CommonError()
	wrappedSentinelErr := sentinelErr.AddCategory("test function")
	databaseErr := common.WrapErrorWithCategories(
		errors.New("database connection failed. details: ..."),
		common.ErrTypeDatabase,
	).CommonError()
	wrappedDatabaseErr := databaseErr.AddCategory("create user")
	packagedDatabaseErr := databaseErr.AddCategory("auth [package]")

	require.Equal(t, "test error", sentinelErr.Error())
	require.Equal(t, "test function error: test error", wrappedSentinelErr.Error())
	require.Equal(t, "database [general] error: database connection failed. details: ...", databaseErr.Error())
	require.Equal(
		t,
		"create user error: database [general] error: database connection failed. details: ...",
		wrappedDatabaseErr.Error(),
	)

	require.Equal(
		t,
		"auth [package] error: database [general] error: database connection failed. details: ...",
		packagedDatabaseErr.Error(),
	)
	packagedDatabaseErr = packagedDatabaseErr.AddCategory("create user")
	require.Equal(
		t,
		"auth [package] error: create user error: database [general] error: database connection failed. details: ...",
		packagedDatabaseErr.Error(),
	)

	repackagedDatabaseErr := packagedDatabaseErr.AddCategory("auth abstraction [package]")
	require.Equal(
		t,
		"auth abstraction [package] error: auth [package] error: create user error: "+
			"database [general] error: database connection failed. details: ...",
		repackagedDatabaseErr.Error(),
	)
	repackagedDatabaseErr = repackagedDatabaseErr.AddCategory("abstraction function")
	require.Equal(
		t,
		"auth abstraction [package] error: abstraction function error: auth [package] error: "+
			"create user error: database [general] error: database connection failed. details: ...",
		repackagedDatabaseErr.Error(),
	)
}

func TestError_worksWithIs(t *testing.T) {
	t.Parallel()
	sentinelErr := common.NewErrorWithCategories(
		"test error, no details",
		errTypeTest,
	).CommonError()
	wrappedSentinelErr := sentinelErr.AddCategory("test function")
	databaseErr := common.WrapErrorWithCategories(
		errors.New("database connection failed. details: ..."),
		common.ErrTypeDatabase,
	).CommonError()

	require.ErrorIs(t, wrappedSentinelErr, sentinelErr)
	require.NotErrorIs(t, sentinelErr, databaseErr)
	require.ErrorIs(t, sentinelErr, sentinelErr.Unwrap())
	require.NotErrorIs(t, sentinelErr.Unwrap(), sentinelErr) // Target is more specific than err

	require.NotSame(t, sentinelErr, wrappedSentinelErr)
	require.ErrorIs(t, wrappedSentinelErr, sentinelErr)
	require.NotErrorIs(t, wrappedSentinelErr, databaseErr)
	require.ErrorIs(t, wrappedSentinelErr, wrappedSentinelErr.Unwrap())
	require.NotErrorIs(t, wrappedSentinelErr.Unwrap(), wrappedSentinelErr) // Target is more specific than err
}

func TestError_HasCategories(t *testing.T) {
	t.Parallel()
	sentinelErr := common.NewErrorWithCategories(
		"test error, no details",
		errTypeTest,
	).CommonError()
	flatDatabaseErr := common.WrapErrorWithCategories(
		errors.New("database connection failed. details: ..."),
		common.ErrTypeDatabase,
	).CommonError()
	detailedDatabaseErr := common.WrapErrorWithCategories(
		errors.New("duplicate key error. details: ..."),
		common.ErrTypeDatabase,
		"create user",
	).CommonError()

	require.True(t, sentinelErr.HasCategories(errTypeTest))
	require.True(t, sentinelErr.HasCategories("*"))
	require.False(t, sentinelErr.HasCategories(common.ErrTypeDatabase))
	require.True(t, sentinelErr.HasCategories(errTypeTest, "test error, no details"))
	require.True(t, sentinelErr.HasCategories(errTypeTest, "*"))
	require.True(t, sentinelErr.HasCategories("*", "test error, no details"))
	require.False(t, sentinelErr.HasCategories(common.ErrTypeDatabase, "test error, no details"))
	require.False(t, sentinelErr.HasCategories(common.ErrTypeDatabase, "*"))

	require.True(t, flatDatabaseErr.HasCategories(common.ErrTypeDatabase))
	require.True(t, flatDatabaseErr.HasCategories("*"))
	require.False(t, flatDatabaseErr.HasCategories(errTypeTest))
	require.False(t, flatDatabaseErr.HasCategories(common.ErrTypeDatabase, "some other category"))
	require.False(t, flatDatabaseErr.HasCategories(common.ErrTypeDatabase, "*"))
	require.False(t, flatDatabaseErr.HasCategories("*", "some other category"))
	require.False(t, flatDatabaseErr.HasCategories("*", "*"))

	require.False(t, detailedDatabaseErr.HasCategories(common.ErrTypeDatabase))
	require.False(t, detailedDatabaseErr.HasCategories(errTypeTest))
	require.True(t, detailedDatabaseErr.HasCategories("create user", common.ErrTypeDatabase))
	require.True(t, detailedDatabaseErr.HasCategories("create user", "*"))
	require.False(t, detailedDatabaseErr.HasCategories("create user", errTypeTest))
	require.True(t, detailedDatabaseErr.HasCategories("*", common.ErrTypeDatabase))
	require.True(t, detailedDatabaseErr.HasCategories("*", "*"))
	require.False(t, detailedDatabaseErr.HasCategories("*", errTypeTest))
}

func TestError_Clone(t *testing.T) {
	t.Parallel()
	sentinelErr := common.NewErrorWithCategories(
		"test error, no details",
		errTypeTest,
	)
	copiedErr := sentinelErr.Clone()

	require.Equal(t, sentinelErr, copiedErr)
	require.NotSame(t, sentinelErr, copiedErr)

	copiedErr = copiedErr.AddCategory("new category")
	require.NotEqual(t, sentinelErr, copiedErr)
}

func TestErrorWrapper(t *testing.T) {
	t.Parallel()
	databaseErrWrapper := common.NewErrorWrapper(
		common.ErrTypeDatabase,
	)
	createUserErrNoPackageWrapper := common.NewErrorWrapper(
		common.ErrTypeDatabase,
		"create user",
	)
	createUserErrWrapper := common.NewErrorWrapper(
		common.ErrTypeDatabase,
		"create user",
		"auth [package]",
	)

	rootError := errors.New("duplicate key error. details: ...")
	require.Equal(
		t,
		common.WrapErrorWithCategories(rootError, common.ErrTypeDatabase),
		databaseErrWrapper.Wrap(rootError),
	)
	require.Equal(
		t,
		common.WrapErrorWithCategories(rootError, common.ErrTypeDatabase, "create user"),
		createUserErrNoPackageWrapper.Wrap(rootError),
	)
	require.Equal(
		t,
		common.WrapErrorWithCategories(rootError, common.ErrTypeDatabase, "create user", "auth [package]"),
		createUserErrWrapper.Wrap(rootError),
	)
}

func TestErrorWrapper_removesDuplicatePackages(t *testing.T) {
	t.Parallel()
	errWrapperDbPackageA := common.NewErrorWrapper(
		"package A [package]", common.ErrTypeDatabase,
	)
	errWrapperCreateUserPackageA := common.NewErrorWrapper(
		"package A [package]", "create user",
	)
	errWrapperCreateUserPackageB := common.NewErrorWrapper(
		"package B [package]", "create team",
	)
	errWrapperSomethingPackageA := common.NewErrorWrapper(
		"package A [package]", "some category that is back in package A again",
	)

	rootError := errors.New("duplicate key error. details: ...")
	require.Equal(
		t,
		common.WrapErrorWithCategories(rootError, "package A [package]", common.ErrTypeDatabase),
		errWrapperDbPackageA.Wrap(rootError),
	)
	require.Equal(
		t,
		// The package should only appear once
		common.WrapErrorWithCategories(rootError, "package A [package]", common.ErrTypeDatabase, "create user"),
		errWrapperCreateUserPackageA.Wrap(errWrapperDbPackageA.Wrap(rootError)),
	)
	require.Equal(
		t,
		common.WrapErrorWithCategories(
			rootError, "package A [package]", common.ErrTypeDatabase, "create user",
			"package B [package]", "create team",
		),
		errWrapperCreateUserPackageB.Wrap(
			errWrapperCreateUserPackageA.Wrap(errWrapperDbPackageA.Wrap(rootError)),
		),
	)
	require.Equal(
		t,
		common.WrapErrorWithCategories(
			rootError, "package A [package]", common.ErrTypeDatabase, "create user",
			"package B [package]", "create team",
			"package A [package]", "some category that is back in package A again",
		),
		errWrapperSomethingPackageA.Wrap(
			errWrapperCreateUserPackageB.Wrap(
				errWrapperCreateUserPackageA.Wrap(errWrapperDbPackageA.Wrap(rootError)),
			),
		),
	)
}

func TestErrorWrapper_canAddPackageToPackagelessError(t *testing.T) {
	t.Parallel()
	commonErrWrapper := common.NewErrorWrapper(
		common.ErrTypeDatabase,
	)

	rootError := errors.New("duplicate key error. details: ...")
	wrappedError := commonErrWrapper.Wrap(rootError).CommonError().AddCategory("users [package]")
	require.Equal(
		t,
		[]string{common.ErrTypeDatabase, "users [package]"},
		wrappedError.Categories(),
	)
	require.Equal(
		t,
		common.WrapErrorWithCategories(rootError, "users [package]", common.ErrTypeDatabase),
		wrappedError,
	)
}

func TestErrorWrapper_HasWrapped(t *testing.T) {
	t.Parallel()
	// TODO: wrap by passing the error through each wrapper
	// TODO: check this checks the packages properly
	commonDatabaseErrWrapper := common.NewErrorWrapper(
		common.ErrTypeDatabase,
	)
	authDatabaseErrWrapper := common.NewErrorWrapper(
		"auth [package]",
		common.ErrTypeDatabase,
	)
	createUserErrWrapper := common.NewErrorWrapper(
		common.ErrTypeDatabase,
		"auth [package]",
		"create user",
	)
	createUserAbstractionErrWrapper := common.NewErrorWrapper(
		common.ErrTypeDatabase,
		"auth [package]",
		"create user",
		"auth abstraction [package]",
		"abstraction function",
	)
	authPackageWrapper := common.NewErrorWrapper(
		"auth [package]",
	)

	rootError := errors.New("duplicate key error. details: ...")
	wrappedCommonDatabaseErr := commonDatabaseErrWrapper.Wrap(rootError).CommonError()
	wrappedAuthDatabaseErr := authDatabaseErrWrapper.Wrap(rootError).CommonError()
	wrappedCreateUserErr := createUserErrWrapper.Wrap(rootError).CommonError()
	wrappedCreateUserAbstractionErr := createUserAbstractionErrWrapper.Wrap(rootError).CommonError()

	require.False(t, createUserErrWrapper.HasWrapped(errors.New("generic error")))
	require.True(t, commonDatabaseErrWrapper.HasWrapped(wrappedCommonDatabaseErr))
	require.True(
		// It compares the categories by value rather than tracking which wrappers were used
		t, commonDatabaseErrWrapper.HasWrapped(wrappedAuthDatabaseErr),
	)
	require.True(t, authDatabaseErrWrapper.HasWrapped(wrappedAuthDatabaseErr))
	require.False(t, authDatabaseErrWrapper.HasWrapped(wrappedCommonDatabaseErr))
	require.False(t, createUserErrWrapper.HasWrapped(wrappedCommonDatabaseErr))
	require.False(t, createUserErrWrapper.HasWrapped(wrappedAuthDatabaseErr))
	require.True(t, commonDatabaseErrWrapper.HasWrapped(wrappedCreateUserErr))
	require.True(t, authDatabaseErrWrapper.HasWrapped(wrappedCreateUserErr))

	require.True(t, createUserErrWrapper.HasWrapped(wrappedCreateUserErr))
	require.False(t, createUserAbstractionErrWrapper.HasWrapped(wrappedCreateUserErr))
	require.True(t, createUserErrWrapper.HasWrapped(
		wrappedCreateUserErr.
			AddCategory("auth abstraction [package]").
			AddCategory("abstraction function"),
	))
	require.True(t, createUserErrWrapper.HasWrapped(
		wrappedCreateUserAbstractionErr,
	))

	require.True(t, authPackageWrapper.HasWrapped(
		wrappedCreateUserAbstractionErr,
	))
}
