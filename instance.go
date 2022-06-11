package event

import (
	"strings"

	. "github.com/chefsgo/base"
	"github.com/chefsgo/chef"
)

// 收到消息
// 待优化，加入协程池，限制单个事件的并发
func (this *Instance) Serve(alias string, data []byte) {
	if strings.HasPrefix(alias, this.config.Prefix) {
		alias = strings.TrimPrefix(alias, this.config.Prefix)
	}

	ctx := &Context{inst: this}

	relate := this.module.relateKey(this.name, alias)
	if name, ok := this.module.relates[relate]; ok {
		ctx.Name = name
		if cfg, ok := this.module.events[ctx.Name]; ok {
			ctx.Config = &cfg
		}

		// 解析元数据
		metameta := chef.Metadata{}
		err := chef.Unmarshal(this.config.Codec, data, &metameta)
		if err == nil {
			ctx.Metadata(metameta)
		}
	}

	//开始执行
	this.serve(ctx)
	chef.CloseMeta(&ctx.Meta)
}

func (this *Instance) serve(ctx *Context) {
	//清理执行线
	ctx.clear()

	//request拦截器
	ctx.next(this.module.requestFilters...)
	ctx.next(this.request)

	//开始执行
	ctx.Next()
}

// request 请求处理
func (this *Instance) request(ctx *Context) {
	ctx.clear()

	//request拦截器
	ctx.next(this.finding)     //存在否
	ctx.next(this.authorizing) //身份验证
	ctx.next(this.arguing)     //参数处理
	ctx.next(this.execute)

	//开始执行
	ctx.Next()

	//response得在这里
	//这里才能被覆盖在request拦截器中
	this.response(ctx)
}

// execute 执行线
func (this *Instance) execute(ctx *Context) {
	ctx.clear()

	//execute拦截器
	ctx.next(this.module.executeFilters...)
	if ctx.Config.Actions != nil || len(ctx.Config.Actions) > 0 {
		ctx.next(ctx.Config.Actions...)
	}
	if ctx.Config.Action != nil {
		ctx.next(ctx.Config.Action)
	}

	//开始执行
	ctx.Next()
}

// response 响应线
func (this *Instance) response(ctx *Context) {
	ctx.clear()

	//response拦截器
	ctx.next(this.module.responseFilters...)
	// ctx.next(this.body)

	//开始执行
	ctx.Next()
}

// finding 判断不
func (this *Instance) finding(ctx *Context) {
	if ctx.Config == nil {
		ctx.Found()
	} else {
		ctx.Next()
	}
}

// authorizing token验证
func (this *Instance) authorizing(ctx *Context) {
	// 待处理
	ctx.Next()
}

// arguing 参数解析
func (this *Instance) arguing(ctx *Context) {
	if ctx.Config.Args != nil {
		argsValue := Map{}
		res := chef.Mapping(ctx.Config.Args, ctx.Value, argsValue, ctx.Config.Nullable, false, ctx.Timezone())
		if res != nil && res.Fail() {
			ctx.Failed(res)
		}

		for k, v := range argsValue {
			ctx.Args[k] = v
		}
	}
	ctx.Next()
}

func (this *Instance) found(ctx *Context) {
	ctx.clear()

	//把处理器加入调用列表
	if ctx.Config.Found != nil {
		ctx.next(ctx.Config.Found)
	}
	ctx.next(this.module.foundHandlers...)

	ctx.Next()
}

func (this *Instance) error(ctx *Context) {
	ctx.clear()

	//把处理器加入调用列表
	if ctx.Config.Error != nil {
		ctx.next(ctx.Config.Error)
	}
	ctx.next(this.module.errorHandlers...)

	ctx.Next()
}

func (this *Instance) failed(ctx *Context) {
	ctx.clear()

	//把处理器加入调用列表
	if ctx.Config.Failed != nil {
		ctx.next(ctx.Config.Failed)
	}
	ctx.next(this.module.failedHandlers...)

	ctx.Next()
}

func (this *Instance) denied(ctx *Context) {
	ctx.clear()

	//把处理器加入调用列表
	if ctx.Config.Denied != nil {
		ctx.next(ctx.Config.Denied)
	}
	ctx.next(this.module.deniedHandlers...)

	ctx.Next()
}

// 最终的默认body响应
// 其实 event 目前没有任何需要反馈的
// func (this *Instance) body(ctx *Context) {
// 	switch body := ctx.Body.(type) {
// 	default:
// 		this.bodyDefault(ctx)
// 	}
// }

// // bodyDefault 默认的body处理
// func (this *Instance) bodyDefault(ctx *Context) {
// 	//here is nothing todo
// }
