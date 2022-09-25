# op-agent
op-agent是一款轻量级的agent，可以实现在物理机上部署任务、管理任务、定时执行、API功能暴露等功能，并采用中心化配置管理方式，可以很方便的为其他系统提供子模块功能。

# 架构
![op-agent logo](https://github.com/liuhr/op-agent/blob/master/docs/images/op-agent.png)

* op-manager： op-manager是管理节点，用来管理所有op-agent，在管理节点上可以配置任务、查看任务、查看任务执行的日志等等。op-manager可以是单个节点，也可以部署成raft集群。
* op-agent： 运行在线上物理机上的agent组件。
