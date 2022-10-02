# 单机部署测试集群
整个系统主要有控制端op-manager、受控端的op-agent、用户使用命令行工具agentCli组成，下面是3个个组件的部署流程。

# 部署前环境准备
准备一台部署主机。
* 部署需要使用部署主机的root用户启动相关组件
  
* 部署主机关闭防火墙或者开放集群的节点间通信所需端口

* 安装CentOS 7.X版本（非centos7.x 需要自行编译op-manager、op-agent、agentCli）


# 准备元数据库
部署数据库MySQL实例。
1. 安装步骤略过，请自行安装（在任意一台机器安装个MySQL实例，op-manager、op-agent、agentCli组件能连接上）。

2. 创建元数据库和访问用户授权 
   
   create database op_meta;

create user op_meta_user@'%' identified by 'user_password';

grant all privileges on op_meta.* to op_meta_user@'%';
flush privileges;

# 安装op-manager
mkdir -p /data/op-manager/log

cd /data/op-manager

wget https://github.com/liuhr/op-agent/blob/master/releases/centOS_release_7.x/op-manager

chmod +x op-manager

wget https://github.com/liuhr/op-agent/blob/master/releases/config/op-manager.conf.json

vim op-manager.conf.json

"ListenAddress": ":8090" #op-manager监听的端口

"BackendDbHosts": "127.0.0.1",
"BackendDbPort":3306,
"BackendDbUser":"op_meta_user",
"BackendDbPass":"user_password",
"BackendDb":"op_meta",

/data/op-manager/op-manager --config=/data/op-manager/op-manager.conf.json  #启动op-manager

# 安装agentCli命令行工具
cd /data/op-manager

wget https://github.com/liuhr/op-agent/blob/master/releases/centOS_release_7.x/agentCli

chmod +x agentCli

wget https://github.com/liuhr/op-agent/blob/master/releases/config/agentCli.conf.json

vim agentCli.conf.json

{
//配置连接元数据库
"BackendDbHosts": "127.0.0.1",
"BackendDbPort":3306,
"BackendDbUser":"op_meta_user",
"BackendDbPass":"user_password",
"BackendDb":"op_meta",

//配置连接agent api
"OpAgentUser": "opuser",
"OpAgentPass": "w95fa8cw403fc220db1f4csde2130bsfd",
"OpAgentPort": 7070,
"OpAgentApiEndpoint": "/api/opAgent"
}

./agentCli -h

# 安装op-agent
mkdir /data/op-agent/log

cd /data/op-agent/

wget https://github.com/liuhr/op-agent/blob/master/releases/centOS_release_7.x/op-agent

chmod +x op-agent

wget https://github.com/liuhr/op-agent/blob/master/releases/config/op-agent.conf.json

vim /data/op-agent/op-agent.conf.json

"ListenAddress": ":7070",

"BackendDbHosts": "127.0.0.1",
"BackendDbPort":3306,
"BackendDbUser":"op_meta_user",
"BackendDbPass":"user_password",
"BackendDb":"op_meta",

"OpServers": ["127.0.0.1"],
"OpServerLeader": "",
"OpServerPort": 8090,
"OpServerUser": "opuser",
"OpServerPass": "w95fa8cw403fc220db1f4csde2130bsfd",
"OpServerApiEndPoint": "/api/opManager",

/data/op-agent/op-agent --config=/data/op-agent/op-agent.conf.json  #启动op-agent


# 验证
cd /data/op-manager

./agentCli get nodes
