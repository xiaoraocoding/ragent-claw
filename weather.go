package main

import (
	"fmt"
	"time"
)

func main() {
	// 模拟武汉天气信息
	fmt.Printf("武汉天气情况 (%s)\n", time.Now().Format("2006-01-02 15:04"))
	fmt.Println("=================================")
	fmt.Println("天气状况: 多云")
	fmt.Println("当前温度: 25.5°C")
	fmt.Println("体感温度: 26.8°C")
	fmt.Println("最低温度: 22.0°C")
	fmt.Println("最高温度: 30.0°C")
	fmt.Println("气压: 1013hPa")
	fmt.Println("湿度: 75%")
	fmt.Println("风速: 3.5 m/s")
	fmt.Println("=================================")
	
	// 提供一些武汉天气的一般信息
	fmt.Println("\n武汉气候特点:")
	fmt.Println("- 武汉属于亚热带季风气候")
	fmt.Println("- 夏季炎热潮湿，有火炉之称")
	fmt.Println("- 冬季寒冷，偶有降雪")
	fmt.Println("- 春秋两季较为温和，是旅游的好时节")
}
