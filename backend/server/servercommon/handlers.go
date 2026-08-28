package servercommon

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func NewHandler(
	handler func(ginCtx *gin.Context) error,
) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		stdErr := handler(ginCtx)
		if stdErr != nil {
			ginCtx.Error(stdErr)
		}
	}
}

func NewObjectIDHandler(
	handler func(id uuid.UUID, ginCtx *gin.Context) error,
) gin.HandlerFunc {
	return NewHandler(func(ginCtx *gin.Context) error {
		id, serverErr := ParseObjectID(ginCtx.Param("id"))
		if serverErr != nil {
			return serverErr
		}
		return handler(id, ginCtx)
	})
}
