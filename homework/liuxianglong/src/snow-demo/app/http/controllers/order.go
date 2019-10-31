package controllers

import (
	"github.com/gin-gonic/gin"
	"snow-demo/app/services/orderservices"
	"strconv"
	"snow-demo/app/http/entities"
	"snow-demo/app/constants/errorcode"
	"snow-demo/app/jobs/basejob"
	"github.com/qit-team/work"
	"math/rand"
	"encoding/json"
	"fmt"
	"context"
)

// request和response的示例
// HandleGetOrderInfo godoc
// @Summary request和response的示例
// @Description request和response的示例
// @Tags snow
// @Accept  json
// @Produce  json
// @Param test body entities.OrderValidatorRequest true "test request"
// @Success 200 {array} entities.TestResponse
// @Failure 400 {object} controllers.HTTPError
// @Failure 404 {object} controllers.HTTPError
// @Failure 500 {object} controllers.HTTPError
// @Router /test [post]
func HandleGetOrderInfo(c *gin.Context) {
	// OrderValidatorRequest
	orderIdStr := c.Query("orderId")
	orderId, _ := strconv.Atoi(orderIdStr)

	order,err := orderservices.GetOrderInfoById(orderId)
	if err != nil {
		Error500(c)
		return
	}
	//logger.Debug(c, "GetOrderInfo", orderId)
	data := map[string]interface{}{
		"id" : orderId,
		"orderNo" : order.OrderNo,
	}
	Success(c, data)
	return
}

func HandleOrder(c *gin.Context){
	request := new(entities.OrderValidatorRequest)
	err := GenRequest(c, request)
	if err != nil {
		Error(c, errorcode.ParamError)
		return
	}

	jsonBytes, err := json.Marshal(request)
	if err != nil {
		fmt.Println(err);
		Error(c, 500)
		return
	}
	
	task := work.Task{
		Id:      strconv.Itoa(rand.Intn(100)),
		Message: string(jsonBytes),
	}
	ok, err := basejob.EnqueueWithTask(context.TODO(), "", task)
	if ok {
		Success(c, nil)

	} else {
		fmt.Println(err)
		Error500(c)
	}
	return
}