package application

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/internal/machine/imsg"
	"mayfly-go/internal/machine/mcm"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"

	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)

// MachineHostKey 主机公钥指纹信任库。
//
// 信任策略（首次信任/指纹比对/失配拒绝）是业务决策，收敛在本层并经 ssh.HostKeyCallback
// 标准接缝下发给 mcm 连接层；mcm 不感知策略与存储，策略演进不触碰连接代码。
//
// 不做进程内指纹缓存：多实例部署下实例间的撤销信任无法同步缓存，缓存命中会绕过
// fail-closed 校验；连接建立频率低（状态轮询为每台每 2 分钟一次），直查信任库足够。
type MachineHostKey interface {
	base.App[*entity.MachineHostKey]

	// VerifyCallback 返回该机器连接时的主机公钥校验回调：
	// 无记录按 TOFU 采集指纹放行，指纹匹配放行，失配按中间人攻击处理拒绝连接
	VerifyCallback(mi *mcm.MachineInfo) ssh.HostKeyCallback
}

type machineHostKeyAppImpl struct {
	base.AppImpl[*entity.MachineHostKey, repository.MachineHostKey]

	// tofuLocks 首次信任按地址互斥（addr -> lock）：并发首连同主机时先到者落库、
	// 后到者双检命中，避免信任库出现重复指纹行。多实例部署仍有微小重复窗口，
	// 但重复行内容一致不影响校验
	tofuLocks sync.Map
}

var _ MachineHostKey = (*machineHostKeyAppImpl)(nil)

func (m *machineHostKeyAppImpl) VerifyCallback(mi *mcm.MachineInfo) ssh.HostKeyCallback {
	return func(_ string, _ net.Addr, key ssh.PublicKey) error {
		addr := mi.HostKeyAddr()
		fingerprint := ssh.FingerprintSHA256(key)

		record, err := m.lookupForTrust(addr, key, fingerprint, mi)
		if err != nil {
			return err
		}

		if record.Fingerprint == fingerprint {
			return nil
		}

		// 指纹失配：既可能是主机换钥/重装，也可能是中间人攻击，平台无法替管理员裁决，
		// 只能拒绝并给出可操作的核对路径（比对两侧指纹后撤销旧信任重连）
		return errorx.NewBizI(context.Background(), imsg.ErrHostKeyMismatch,
			"addr", addr,
			"oldFp", record.Fingerprint,
			"newFp", fingerprint,
		)
	}
}

// lookupForTrust 查询主机的信任记录；无记录时按 TOFU 采集指纹落库。
// 按地址互斥 + 双检，避免并发首连同主机落重复指纹行
func (m *machineHostKeyAppImpl) lookupForTrust(addr string, key ssh.PublicKey, fingerprint string, mi *mcm.MachineInfo) (*entity.MachineHostKey, error) {
	record := &entity.MachineHostKey{HostAddr: addr}
	err := m.GetByCond(record)
	if err == nil {
		return record, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		// 信任库读不出来时不能放行：放行等于把「存储故障」降级成「不校验」
		return nil, errorx.NewBizf("verify host key failed: %s", err.Error())
	}

	// 首次信任（TOFU）：当前会话无法证明对端身份，采集指纹备案后放行，
	// 后续连接以此指纹为准。指纹属自动采集的系统事实，不归属操作者
	v, _ := m.tofuLocks.LoadOrStore(addr, &sync.Mutex{})
	mu := v.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()

	// 双检：等锁期间并发首连可能已落库
	record = &entity.MachineHostKey{HostAddr: addr}
	if err := m.GetByCond(record); err == nil {
		return record, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errorx.NewBizf("verify host key failed: %s", err.Error())
	}

	record = &entity.MachineHostKey{
		HostAddr:    addr,
		KeyType:     key.Type(),
		Fingerprint: fingerprint,
		Remark:      fmt.Sprintf("auto trust on first connect: %s", mi.Name),
	}
	if err := m.Insert(context.Background(), record); err != nil {
		return nil, err
	}
	return record, nil
}
