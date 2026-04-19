
reso_addr='crpi-x7po5hftx788xubp.cn-hangzhou.personal.cr.aliyuncs.com/yllmis-im/im-api-dev'
tag='latest'

container_name="yllmis-im-im-api-test"

docker stop ${container_name}

docker rm ${container_name}

docker rmi ${reso_addr}:${tag}

docker pull ${reso_addr}:${tag}


#如果需要指定配置文件的
# docker run -p 10000:10000 --network imooc_easy-im -v /easy-im/config/user-rpc:/user/conf/-name=$fcontainer_name} -d $freso_addr}:${tag]
docker run -p 8882:8882 --network im-grpc_yllmis-im \
-v /home/ubuntu/go_projects/IM-Grpc/apps/im/api/etc/dev/im.yaml:/im/conf/api.yaml \
--name=${container_name} -d ${reso_addr}:${tag}