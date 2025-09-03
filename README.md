### cmder

- **api**
```shell
method POST  /api/cmd/run?name=test
{
    cmd: "echo 'hello world'"
}
# 返回 {"task_id": "0929129c-a44b-4bab-8f64-99a7cba45339"}

method GET /api/cmd/out?task_id=0929129c-a44b-4bab-8f64-99a7cba45339

method GET /api/cmd/ids
```

- **示例**  
  ![web示例](docs/cmd-runner.png)


<!-- - **接口测试**
```bash
curl -X POST "http://127.0.0.1:5533/api/cmd/run?name=test"   -d '{"cmd":"for((i=0;i<100;i++)) do echo hello;sleep 1;done"}' -H 'Content-Type: application/json'

# npm install -G wscat
wscat -c "ws://127.0.0.1:5533/api/cmd/out?name=test&task_id=0929129c-a44b-4bab-8f64-99a7cba45339"
``` -->