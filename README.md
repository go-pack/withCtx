# withCtx 批量添加Ctx参数

## 案例1 

将原有函数方法名首字符小写, 并添加ctx参数, 同时生成一个同名同参方法调用小写的私有方法

```go
//go:generate withCtx -m
//https://github.com/go-pack/withCtx

func NewIndexService() *IndexService {
	return &IndexService{}
}

func (t *IndexService) GetConfig() string {
	return "123"
}
```

生成代码如下

```go

func (t *IndexService) getConfig(ctx context.Context) string {
	return "123"
}
func (t *IndexService) GetConfig(ctx context.Context) string {
	return t.getConfig(ctx)
}


```

## 案例2

将原有函数方法添加ctx参数

```go
//go:generate withCtx -m -a
//https://github.com/go-pack/withCtx

func NewIndexService() *IndexService {
	return &IndexService{}
}

func (t *IndexService) GetConfig() string {
	return "123"
}
```

生成代码如下

```go

func (t *IndexService) GetConfig(ctx context.Context) string {
	return "123"
}

```
 
## 案例3

将原有函数方法都不变, 创建一个新方法

```go
//go:generate withCtx -m -x
//https://github.com/go-pack/withCtx

func NewIndexService() *IndexService {
	return &IndexService{}
}

func (t *IndexService) GetConfig(name string) string {
	return name
}
```

生成代码如下

```go

func (t *IndexService) GetConfigWithCtx(ctx context.Context,name string) string {
	return t.GetConfig(name)
}

```
 