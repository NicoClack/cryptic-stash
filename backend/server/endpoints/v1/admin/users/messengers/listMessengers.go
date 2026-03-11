package messengers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/user"
	"github.com/NicoClack/cryptic-stash/backend/ent/usermessenger"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-gonic/gin"
)

type ListMessengersResponse struct {
	Errors     []servercommon.ErrorDetail `binding:"required" json:"errors"`
	Messengers []*Messenger               `binding:"required" json:"messengers"`
}

type Messenger struct {
	VersionedType string          `binding:"required" json:"versionedType"`
	Name          string          `binding:"required" json:"name"`
	Created       bool            `binding:"required" json:"created"`
	Enabled       bool            `binding:"required" json:"enabled"`
	CreatedAt     *time.Time      `                   json:"createdAt"`
	UpdatedAt     *time.Time      `                   json:"updatedAt"`
	Options       json.RawMessage `binding:"required" json:"options"`
	OptionsSchema json.RawMessage `binding:"required" json:"optionsSchema"`
}

func ListMessengers(app *servercommon.ServerApp) gin.HandlerFunc {
	return servercommon.NewHandler(func(ginCtx *gin.Context) error {
		userID, ctxErr := servercommon.ParseObjectID(ginCtx.Param("id"))
		if ctxErr != nil {
			return ctxErr
		}
		userOb, stdErr := dbcommon.WithReadTx(
			ginCtx.Request.Context(), app.Database,
			func(tx *ent.Tx, ctx context.Context) (*ent.User, error) {
				return tx.User.Query().
					Where(user.ID(userID)).
					WithMessengers(func(messengerQuery *ent.UserMessengerQuery) {
						messengerQuery.Order(ent.Asc(usermessenger.FieldType), ent.Asc(usermessenger.FieldVersion))
					}).
					Only(ctx)
			},
		)
		if stdErr != nil {
			return servercommon.Send404IfNotFound(stdErr)
		}

		definitions := app.Messengers.AllPublicDefinitions()
		createdMessengerTypes := make(map[string]struct{}, len(definitions))
		responseMessengers := make([]*Messenger, 0, len(definitions))
		for _, messengerOb := range userOb.Edges.Messengers {
			versionedType := common.GetVersionedType(messengerOb.Type, messengerOb.Version)
			definition, ok := app.Messengers.GetPublicDefinition(versionedType)
			if !ok {
				return fmt.Errorf(
					"user %v has %v messenger configured but it has no definition",
					userOb.ID,
					versionedType,
				)
			}

			createdMessengerTypes[versionedType] = struct{}{}

			//exhaustruct:enforce
			responseMessengers = append(responseMessengers, &Messenger{
				VersionedType: versionedType,
				Name:          definition.Name,
				Created:       true,
				Enabled:       messengerOb.Enabled,
				CreatedAt:     common.Pointer(messengerOb.CreatedAt),
				UpdatedAt:     common.Pointer(messengerOb.UpdatedAt),
				Options:       messengerOb.Options,
				OptionsSchema: definition.OptionsSchema,
			})
		}
		for _, definition := range definitions {
			versionedType := common.GetVersionedType(definition.ID, definition.Version)
			_, ok := createdMessengerTypes[versionedType]
			if ok {
				continue
			}

			//exhaustruct:enforce
			responseMessengers = append(responseMessengers, &Messenger{
				VersionedType: versionedType,
				Name:          definition.Name,
				Created:       false,
				Enabled:       false,
				CreatedAt:     nil,
				UpdatedAt:     nil,
				Options:       nil,
				OptionsSchema: definition.OptionsSchema,
			})
		}

		//exhaustruct:enforce
		ginCtx.JSON(http.StatusOK, ListMessengersResponse{
			Errors:     []servercommon.ErrorDetail{},
			Messengers: responseMessengers,
		})
		return nil
	})
}
