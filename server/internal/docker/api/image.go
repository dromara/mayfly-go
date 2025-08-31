package api

import (
	"fmt"
	"io"
	"mayfly-go/internal/docker/api/form"
	"mayfly-go/internal/docker/api/vo"
	"mayfly-go/internal/docker/dkm"
	"mayfly-go/internal/docker/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/jsonx"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

type Image struct {
}

func (d *Image) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		req.NewGet("", d.GetImages),

		req.NewPost("/remove", d.ImageRemove).Log(req.NewLogSaveI(imsg.LogDockerImageRemove)),

		req.NewGet("/save", d.ImageExport).NoRes(),

		req.NewPost("/load", d.ImageLoad).Log(req.NewLogSaveI(imsg.LogDockerImageLoad)),
	}

	return req.NewConfs("docker/images", reqs[:]...)
}

func (d *Image) GetImages(rc *req.Ctx) {
	cli, err := dkm.GetCli(rc.Query("host"))
	biz.ErrIsNil(err)
	is, err := cli.ImageList()
	biz.ErrIsNil(err)

	containers, _ := cli.ContainerList()
	imageId2Container := collx.ArrayToMap(containers, func(item container.Summary) string {
		return item.ImageID
	})
	rc.ResData = collx.ArrayMap[image.Summary, vo.Image](is, func(val image.Summary) vo.Image {
		c := vo.Image{
			Id:         val.ID,
			Size:       val.Size,
			CreateTime: time.Unix(val.Created, 0),
			Tags:       val.RepoTags,
			IsUse:      imageId2Container[val.ID].ID != "",
		}

		return c
	})
}

func (d *Image) ImageRemove(rc *req.Ctx) {
	imageOp := &form.ImageOp{}
	biz.ErrIsNil(rc.BindJSON(imageOp))

	rc.ReqParam = collx.Kvs("host", imageOp.Host, "imageId", imageOp.ImageId)
	cli, err := dkm.GetCli(imageOp.Host)
	biz.ErrIsNil(err)
	err = cli.ImageRemove(imageOp.ImageId)
	biz.ErrIsNil(err)
}

func (d *Image) ImageLoad(rc *req.Ctx) {
	host := rc.PostForm("host")
	biz.NotEmpty(host, "host cannot be empty")
	rc.ReqParam = host

	fileheader, err := rc.FormFile("file")
	biz.ErrIsNilAppendErr(err, "read form file error: %s")

	file, err := fileheader.Open()
	biz.ErrIsNil(err)
	defer file.Close()

	cli, err := dkm.GetCli(host)
	biz.ErrIsNil(err)
	resp, err := cli.DockerClient.ImageLoad(rc.MetaCtx, file)
	biz.ErrIsNil(err)
	defer resp.Body.Close()

	content, err := io.ReadAll(resp.Body)
	biz.ErrIsNil(err)
	errMsg, _ := jsonx.GetStringByBytes(content, "errorDetail.message")
	biz.IsTrue(errMsg == "", "%s", errMsg)
}

func (d *Image) ImageExport(rc *req.Ctx) {
	host := rc.Query("host")
	biz.NotEmpty(host, "host cannot be empty")
	tag := rc.Query("tag")
	biz.NotEmpty(tag, "tag cannot be empty")

	cli, err := dkm.GetCli(host)
	biz.ErrIsNil(err)

	reader, err := cli.DockerClient.ImageSave(rc.MetaCtx, []string{tag}, client.ImageSaveWithPlatforms())
	biz.ErrIsNil(err)
	defer reader.Close()

	filename := rc.QueryDefault("filename", tag)
	rc.Download(reader, fmt.Sprintf("%s.tar", filename))
}
