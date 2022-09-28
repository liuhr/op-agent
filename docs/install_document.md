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








