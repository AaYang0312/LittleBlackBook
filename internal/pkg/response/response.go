package response

import (
	"errors"
	"net/http"
	"xbs/internal/pkg/errs"

	"github.com/gin-gonic/gin"
)

type body struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, body{Code: 0, Message: "success", Data: data})
}

func Fail(c *gin.Context, err error) {
	if e, ok := errors.AsType[*errs.Error](err); !ok {
		c.JSON(http.StatusInternalServerError, body{Code: errs.ErrInternal.Code, Message: errs.ErrInternal.Message})
	} else {
		status := http.StatusOK
		switch e.Code {
		case errs.ErrParam.Code:
			status = http.StatusBadRequest
		case errs.ErrUnauthorized.Code:
			status = http.StatusUnauthorized
		case errs.ErrNoteNotFound.Code:
			status = http.StatusNotFound
		case errs.ErrForbidden.Code:
			status = http.StatusForbidden
		}
		c.JSON(status, body{
			Code:    e.Code,
			Message: e.Message,
		})
		return
	}
	//c.JSON(http.StatusInternalServerError, body{Code: errs.ErrInternal.Code, Message: errs.ErrInternal.Message})
}
