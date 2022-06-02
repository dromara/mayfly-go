package scheduler

func init() {
	SaveMachineMonitor()
}

func SaveMachineMonitor() {
	AddFun("@every 60s", func() {
		// for _, m := range models.GetNeedMonitorMachine() {
		// 	m := m
		// 	go func() {
		// 		cli, err := machine.GetCli(uint64(utils.GetInt4Map(m, "id")))
		// 		if err != nil {
		// 			mlog.Log.Error("获取客户端失败：", err.Error())
		// 			return
		// 		}
		// 		mm := cli.GetMonitorInfo()
		// 		if mm != nil {
		// 			err := model.Insert(mm)
		// 			if err != nil {
		// 				mlog.Log.Error("保存机器监控信息失败: ", err.Error())
		// 			}
		// 		}
		// 	}()
		// }
	})
}
