# op-agent
op-agent是一款轻量级的agent，可以实现在物理机上部署任务、管理任务、定时执行、API功能暴露等功能，并采用中心化配置管理方式，可以很方便的为其他系统提供子模块功能。

# 架构
![op-agent logo](https://github.com/liuhr/op-agent/blob/master/docs/images/op-agent.png)

* op-manager： op-manager是管理节点，用来管理op-agent，在管理节点上可以配置任务、查看任务、查看任务执行的日志等等。op-manager可以是单个节点，也可以部署成raft集群。
* op-agent： 运行在线上物理机上的agent组件。

# 主要功能
* 部署定时任务：
通过op-agent管理的定时任务更具可控性，可设置脚本执行超时时间，可防止同一时间多次执行，可查看脚本执行日志等
* 部署一次性任务：可以设置脚本一次性执行
* 设置某些机器为白名单或黑名
* 通过op-agent提供的API功能远程调用脚本任务：
可以异步/同步地远程调用任务，并可接收参数传递, 此功能可把agent功能作为其他模块的子模块，比如初始化系统、备份系统、巡检系统等
* 脚本文件或项目目录上传：
可以把本地python脚本推送到所有安装了op-agent的机器上
* 任务编辑、查看、管理等：
op-agent最核心的功能就是运行任务，我们可以编辑一个任务的属性，比如任务执行路径、执行周期、超时控制等功能
* 节点状态查看：
查看agent的状态，比如哪些机器的agent挂掉了或者agent没有上报自己的心跳

# 安装和使用
### 快速上手

[1.最简架构部署](./docs/install_document.md)

[2.主要功能使用](./docs/how_to_use_document.md)

### 线上标准部署
