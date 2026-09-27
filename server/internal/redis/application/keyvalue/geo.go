package keyvalue

import (
	"context"

	"mayfly-go/internal/redis/domain/entity"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
)

// ViewGeo GEO 视角：Redis 的 GEO 以 zset 存储，因此本视角服务 zset 类型
const ViewGeo = "geo"

// Extra 中经纬度的键名，同时是表格列与编辑表单的字段名（三者对齐，前端无需映射）
const (
	extraLongitude = "longitude"
	extraLatitude  = "latitude"
)

var geoDesc = &entity.ViewDescriptor{
	View:   ViewGeo,
	Label:  "redis.viewGeo",
	Types:  []entity.KeyType{entity.KeyTypeZset},
	Layout: "table",
	Caps: entity.Capabilities{
		Create: true, Update: true, Delete: true, BatchDelete: true,
		Keyword: true, RankPaging: true, CursorPaging: true, Ops: true,
	},
	// ConsoleHints 命令控制台的快捷命令模板，{key} 由前端换成当前 key 名
	ConsoleHints: []string{
		"GEOPOS {key} member",
		"GEODIST {key} member1 member2",
		"GEOSEARCH {key} FROMMEMBER member BYRADIUS 100 km",
	},
	Columns: []entity.Column{
		sortableColumn("field", "redis.colMember", "text", 200),
		column(extraLongitude, "redis.colLongitude", "number", 140),
		column(extraLatitude, "redis.colLatitude", "number", 140),
		// 不展示 score：GEO 的分值是 geohash 编码后的整数，对用户既不可读也不可改，摆出来只会干扰坐标核对
	},
	Form: form(
		textFieldWith(argField, "redis.colMember", "", true),
		numberField(extraLongitude, "redis.colLongitude", true),
		numberField(extraLatitude, "redis.colLatitude", true),
	),
	Ops: []entity.OpSpec{
		op("geodist", "redis.opGeoDist", false, form(
			textFieldWith(argMember, "redis.colMember", "", true),
			textFieldWith(argMemberTo, "redis.colMemberTo", "", true),
			selectField(argUnit, "redis.colUnit", option("m", "redis.unitMeter"), option("km", "redis.unitKilometer"), option("mi", "redis.unitMile"), option("ft", "redis.unitFeet")),
		)),
		op("geosearch", "redis.opGeoSearch", false, form(
			numberField(extraLongitude, "redis.colLongitude", true),
			numberField(extraLatitude, "redis.colLatitude", true),
			numberField(argRadius, "redis.colRadius", true),
			selectField(argUnit, "redis.colUnit", option("m", "redis.unitMeter"), option("km", "redis.unitKilometer")),
			numberField(argCount, "redis.colCount", false),
		)),
	},
}

type geoHandler struct{}

func init() { Register(&geoHandler{}) }

func (g *geoHandler) Descriptor() *entity.ViewDescriptor { return geoDesc }

func (g *geoHandler) Size(ctx context.Context, cmd redis.Cmdable, key string) (int64, error) {
	return cmd.ZCard(ctx, key).Result()
}

func (g *geoHandler) Load(ctx context.Context, cmd redis.Cmdable, q *entity.MemberQuery) (*entity.MemberPage, error) {
	page, err := loadSortedSet(ctx, cmd, q)
	if err != nil || len(page.Members) == 0 {
		return page, err
	}

	members := make([]string, 0, len(page.Members))
	for _, member := range page.Members {
		member.Field = member.Value
		member.Value = ""
		members = append(members, member.Field)
	}

	positions, err := cmd.GeoPos(ctx, q.Key, members...).Result()
	if err != nil {
		// GEOPOS 失败不影响成员列表本身，仅缺经纬度列
		return page, nil
	}
	for i, member := range page.Members {
		if i >= len(positions) || positions[i] == nil {
			continue
		}
		member.Extra = map[string]string{
			extraLongitude: cast.ToString(positions[i].Longitude),
			extraLatitude:  cast.ToString(positions[i].Latitude),
		}
	}
	return page, nil
}

func (g *geoHandler) BuildWrite(_ context.Context, _ redis.Cmdable, w *entity.MemberWrite) ([][]any, error) {
	switch w.Op {
	case entity.MemberOpCreate, entity.MemberOpUpdate:
		member := trimArg(w.Args, argField)
		if member == "" {
			return nil, errInvalidArg("geo member is required")
		}
		longitude, err := parseScore(w.Args[extraLongitude])
		if err != nil {
			return nil, err
		}
		latitude, err := parseScore(w.Args[extraLatitude])
		if err != nil {
			return nil, err
		}
		// 与 zset 同理：先写新坐标成员再删旧成员，避免第二步失败把成员连同坐标一起丢掉
		if w.Op == entity.MemberOpUpdate && w.Member != nil && w.Member.Field != member {
			return [][]any{{"GEOADD", w.Key, longitude, latitude, member}, {"ZREM", w.Key, w.Member.Field}}, nil
		}
		return [][]any{{"GEOADD", w.Key, longitude, latitude, member}}, nil
	case entity.MemberOpDelete:
		members := pickFields(w, func(m *entity.Member) string { return m.Field })
		if len(members) == 0 {
			return nil, errInvalidArg("no geo member to delete")
		}
		return [][]any{appendAll([]any{"ZREM", w.Key}, members)}, nil
	}
	return nil, errInvalidArg("geo op: " + w.Op)
}

func (g *geoHandler) PlanOp(ctx context.Context, cmd redis.Cmdable, req *entity.OpRequest) ([][]any, error) {
	switch req.Op {
	case "geodist":
		member1, member2 := trimArg(req.Args, argMember), trimArg(req.Args, argMemberTo)
		if member1 == "" || member2 == "" {
			return nil, errInvalidArg("two geo members are required")
		}
		unit := trimArg(req.Args, argUnit)
		if unit == "" {
			unit = "m"
		}
		return [][]any{{"GEODIST", req.Key, member1, member2, unit}}, nil
	case "geosearch":
		longitude, err := parseScore(req.Args[extraLongitude])
		if err != nil {
			return nil, err
		}
		latitude, err := parseScore(req.Args[extraLatitude])
		if err != nil {
			return nil, err
		}
		radius, err := parseScore(req.Args[argRadius])
		if err != nil {
			return nil, err
		}
		unit := trimArg(req.Args, argUnit)
		if unit == "" {
			unit = "m"
		}

		args := []any{"GEOSEARCH", req.Key, "FROMLONLAT", longitude, latitude, "BYRADIUS", radius, unit, "WITHCOORD", "WITHDIST"}
		if count := cast.ToInt64(req.Args[argCount]); count > 0 {
			return [][]any{append(args, "COUNT", count, "ANY")}, nil
		}
		return [][]any{args}, nil
	}
	return nil, errInvalidArg("geo op: " + req.Op)
}
