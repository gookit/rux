# record

## dispatch flow

```text
request
 |
 V
get new ctx  ---> get empty ctx from pool
 |                init ctx with http.Request and http.ResponseWriter
 |                wrap http.ResponseWriter for init ctx.writer
 |                reset some info by ctx.Reset(), and ctx.Resp == ctx.writer
 V
route match
 |
 V
build handlers -> by match status, build handlers chain. 
 |                append handlers to ctx
 |
 V
handle processing  -> call ctx.Next(), loop call handlers
 |
 V
write response
```