package api

import (
	"mayfly-go/internal/docker/dkm"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
)

type Docker struct {
}

func (d *Docker) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		req.NewGet("/info", d.GetDockerInfo),
	}

	return req.NewConfs("docker", reqs[:]...)
}

func (d *Docker) GetDockerInfo(rc *req.Ctx) {
	host := rc.Query("host")
	cli, err := dkm.GetCli(host)
	biz.ErrIsNil(err)
	info, err := cli.DockerClient.Info(rc.MetaCtx)
	biz.ErrIsNil(err)
	rc.ResData = info
}
