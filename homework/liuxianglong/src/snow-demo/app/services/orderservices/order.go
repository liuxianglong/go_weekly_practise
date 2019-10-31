package orderservices

import "snow-demo/app/models/ordermodel"

func GetOrderInfoById(id int) (order *ordermodel.Order, err error){
	order, err = ordermodel.GetInstance().GetOrderInfoById(id)

	return
}

func SaveOrderNo(orderNo string) (err error){
	err = ordermodel.GetInstance().SaveOrderNo(orderNo)
	return
}
