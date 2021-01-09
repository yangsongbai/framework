# First Node Up
第一个节点，启动的时候，没有指定要加入的集群，其为 master！
第二个节点，启动的时候，指定要加入的集群地址：HOST:PORT；
或者通过广播的方式来申请加入集群；

第一个节点，可以在启动的时候，直接指定其为新的集群，这样可以直接生成新的集群 ID; 
如果数据里面的集群信息已经存在过了，则需要根据集群来进行处理：

- 如果集群里面有多个节点，需要和每个节点进行通信；
- 如果集群里面只有他自己，则直接启动集群进入正常状态；

如果申请加入集群，一直没有办法加入集群，可以手动自己成为 Master，生成独立的集群 ID；


第一个节点收到第二个节点的地址，加入到候选地址列表；

第一个节点可以通过 API，允许加入集群列表，或者拒绝其加入到集群列表，可以设置为自动加入模式；
