package ordermodel

import (
	"github.com/qit-team/snow-core/db"
	"sync"
)

var (
	once sync.Once
	m    *bannerModel
)
/**
 * Banner实体
 */
type Order struct {
	Id        int64     `xorm:"pk autoincr"` //注：使用getOne 或者ID() 需要设置主键
	OrderNo       string `xorm:"'order_no'"`
}

/**
 * 表名规则
 * @wiki http://gobook.io/read/github.com/go-xorm/manual-zh-CN/chapter-02/3.tags.html
 */
func (m *Order) TableName() string {
	return "orders"
}

/**
 * 私有化，防止被外部new
 */
type bannerModel struct {
	db.Model //组合基础Model，集成基础Model的属性和方法
}

//单例模式
func GetInstance() *bannerModel {
	once.Do(func() {
		m = new(bannerModel)
		m.DiName = "test_qu" //设置数据库实例连接，默认db.SingletonMain
	})
	return m
}

func (m *bannerModel) GetOrderInfoById(id int) (order *Order,err error) {
	m.GetDb().ShowSQL()
	order = new(Order)
	_, err = m.GetOne(id, order)
	return
}

func (m *bannerModel) SaveOrderNo(orderNo string) (err error) {
	m.GetDb().ShowSQL()
	order := new(Order)
	order.OrderNo = orderNo
	_, err = m.Insert(order)
		//m.GetOne(id, order)
	return err
}

//func (m *bannerModel) GetListByPid(pid int, limits ...int) (banners []*Banner, err error) {
//	banners = make([]*Banner, 0)
//	err = m.GetList(&banners, "pid = ?", []interface{}{pid}, limits)
//	return
//}
//func (m *)
