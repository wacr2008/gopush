gopush
=============================
`gopush` 是一个面向小规模的实时消息推送服务
有tcp和websocket两种协议，保持长连接，故只能做小规模推送。
后续有时间改成udp以支持大规模推送

--------------------------------

## 特性
 * 轻量级
 * 高性能
 * 纯Golang实现
 * 支持单个、多个用户推送，广播及topic广播
 * 心跳支持
 * 支持安全验证（未授权用户不能订阅）
 * 多协议支持（websocket，tcp）
 * 可拓扑的架构
 * 异步消息推送



## 安装部署
### 一、安装依赖
> * 安装redis
> * 安装kafka[这里](http://kafka.apache.org/documentation.html#quickstart)或者nsq[这里](http://nsq.io/overview/design.html)消息队列

### 二、编译comet, admin, push

### 三、部署
> * 1.启动消息队列和redis集群
> * 2.运行admin -> push -> comet

## 服务
### client注册流程
1. client将注册到comet，然后保持心跳
2. comet再通知admin用户上线
3. comet查询redis是否有离线消息，有则发送到消息队列

### 推送流程
1. admin的http接口发送消息，
2. admin将消息发送到消息队列，若是可靠性消息，则先存入redis
3. push从消息队列获取消息，再推送给相应的comet节点
4. comet再将消息推送给相应用户，广播消息则推送给所有相关用户；可靠性消息则再通知push节点消息成功推送
5. push节点从redis中删除成功消息


### comet
面向client，client注册到comet，comet再根据userid进行一致性hash来选择需要连接的admin，将用户上线通知到admin。
若是admin崩溃，则会进行重新注册到admin

### admin
管理用户信息，及推送接口。可以选择推送单个用户，多个用户，广播，topic广播。
> * 单个用户，多个用户：若是可靠性消息，则需要保存到redis集群，然后再入消息队列，消息发送给用户后，再由push从redis中删除。若用户不在线，则下次用户登录会进行推送，保证消息推送顺序；
> * 广播：广播给所有在线用户
> * 广播topic：将消息推送给订阅了某topic的在线用户，如推送消息给所有订阅了娱乐topic的用户

现在信息保留在内存。如果崩溃，下次运行后comet会重新注册用户到admin

### push
从消息队列中获取消息，然后推送给相应的comet。无状态
如果可靠性消息：当comet推送给client成功后，comet会通知push，push再从redis中将消息删除。
广播消息：将消息发给所有的comet，再有comet推送到已注册的用户。




![架构](https://github.com/ikenchina/gopush/blob/master/arch.png)

