package api

import (
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
)

type Docker struct {
}

func (d *Docker) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		req.NewGet("/info", d.GetDockerInfo),
	}

	return req.NewConfs("docker/:id", reqs[:]...)
}

func (d *Docker) GetDockerInfo(rc *req.Ctx) {
	cli := GetCli(rc)
	info, err := cli.DockerClient.Info(rc.MetaCtx)
	biz.ErrIsNil(err)
	rc.ResData = info
}
