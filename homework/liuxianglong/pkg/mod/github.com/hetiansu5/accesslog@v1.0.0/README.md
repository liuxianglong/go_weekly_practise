## 安装
```shell
go get -u -v github.com/hetiansu5/accesslog
```
    
## 使用

### 初始化`AccessLogger`

```golang
acl, err := accesslog.NewLogger(
    accesslog.Output(os.Stdout),
    accesslog.Pattern(accesslog.DefaultPattern),
)

if err != nil {
    log.Panic(err)
}
```

上述代码中的`Output`和`Pattern`选项均使用的是默认值，与下面代码等效

```golang
acl, err := accesslog.NewLogger()

if err != nil {
    log.Panic(err)
}
```

### 使用`AccessLogger`

#### 与 [gin](https://github.com/gin-gonic/gin) 框架一起使用

1. 获取依赖: `go get -u -v github.com/hetiansu5/accesslog/gin` 
1. import: `import ginacl "github.com/hetiansu5/accesslog/gin"`
1. 组合`gin.HandlerFunc`:

```golang
engine := gin.New()
engine.Use(ginacl.AccessLogFunc(acl))
```

#### 与 `net/http` 包一起使用

1. 获取依赖: `go get -u -v github.com/hetiansu5/accesslog/nethttp` 
1. import: `import httpacl "github.com/hetiansu5/accesslog/nethttp"`
1. Wrap `http.HandlerFunc`:

```golang
func handler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}

func main() {
    http.HandleFunc("/", httpacl.NewHandlerFuncWithAccessLog(handler))
    http.ListenAndServe(":8080", nil)
}
```
