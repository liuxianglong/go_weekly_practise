package jobs

import (
	"fmt"
	"github.com/qit-team/work"
	"time"
	"snow-demo/app/http/entities"
	"encoding/json"
	"log"
	"snow-demo/app/services/orderservices"
)

func order(task work.Task) (work.TaskResult) {
	time.Sleep(time.Millisecond * 5)
	s, err := work.JsonEncode(task)
	if err != nil {
		//work.StateFailed 不会进行ack确认
		//work.StateFailedWithAck 会进行actk确认
		//return work.TaskResult{Id: task.Id, State: work.StateFailed}
		return work.TaskResult{Id: task.Id, State: work.StateFailedWithAck}
	} else {
        //work.StateSucceed 会进行ack确认
		fmt.Println("do task", s)
		var t entities.OrderValidatorRequest
		err := json.Unmarshal([]byte(task.Message), &t)
		if err != nil {
			fmt.Println(err)
			log.Fatal(err)
		}
		fmt.Println("do task", t)
		
		err = orderservices.SaveOrderNo(t.OrderNo)
		if err != nil {
			fmt.Println(err)
			log.Fatal(err)
		}
		fmt.Println("suc")
		return work.TaskResult{Id: task.Id, State: work.StateSucceed}
	}

}
