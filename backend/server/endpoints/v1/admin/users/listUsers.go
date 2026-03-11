package users

import (
	"context"
	"net/http"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/user"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
)

type ListUsersResponse struct {
	Errors []servercommon.ErrorDetail `binding:"required" json:"errors"`
	Users  []*User                    `binding:"required" json:"users"`
}
type User struct {
	ID        string    `binding:"required" json:"id"`
	Username  string    `binding:"required" json:"username"`
	CreatedAt time.Time `binding:"required" json:"createdAt"`
	UpdatedAt time.Time `binding:"required" json:"updatedAt"`
}

func ListUsers(app *servercommon.ServerApp) gin.HandlerFunc {
	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		username := ginCtx.Query("username")
		userObs, stdErr := dbcommon.WithReadTx(
			ginCtx.Request.Context(), app.Database,
			func(tx *ent.Tx, ctx context.Context) ([]*ent.User, error) {
				return tx.User.Query().
					Where(user.UsernameHasPrefix(username)).
					Select(user.FieldID, user.FieldUsername, user.FieldCreatedAt, user.FieldUpdatedAt).
					Order(ent.Desc(user.FieldCreatedAt)).
					All(ctx)
			},
		)
		if stdErr != nil {
			return stdErr
		}

		responseUsers := make([]*User, 0, len(userObs))
		for _, userOb := range userObs {
			responseUsers = append(responseUsers, &User{
				ID:        userOb.ID.String(),
				Username:  userOb.Username,
				CreatedAt: userOb.CreatedAt,
				UpdatedAt: userOb.UpdatedAt,
			})
		}

		//exhaustruct:enforce
		ginCtx.JSON(http.StatusOK, ListUsersResponse{
			Errors: []servercommon.ErrorDetail{},
			Users:  responseUsers,
		})
		return nil
	})
}
