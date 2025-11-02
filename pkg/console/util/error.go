package util

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/console/model"
)

func HandleServiceError(ctx *gin.Context, err error) {
	var e bizerror.Error
	if !errors.As(err, &e) {
		e = bizerror.NewBizError(bizerror.UnknownError, err.Error())
	}
	ctx.JSON(http.StatusOK, model.NewBizErrorResp(e))
}

func HandleArgumentError(ctx *gin.Context, err error) {
	e := bizerror.NewBizError(bizerror.InvalidArgument, err.Error())
	ctx.JSON(http.StatusOK, model.NewBizErrorResp(e))
}
