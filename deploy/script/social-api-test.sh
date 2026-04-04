reso_addr='crpi-x7po5hftx788xubp.cn-hangzhou.personal.cr.aliyuncs.com/yllmis-im/user-rpc-dev'
tag='latest'

container_name="yllmis-im-social-api-test"

pod_idb="114.55.85.76"

docker stop ${container_name}

docker rm ${container_name}

docker rmi ${reso_addr}:${tag}

docker pull ${reso_addr}:${tag}


#如果需要指定配置文件的
# docker run -p 10oo1:8080 --network imooc_easy-im -v /easy-im/config/user-rpc:/user/conf/-name=$fcontainer_name} -d $freso_addr}:${tag]
docker run -p 8881:8881 -e POD_IP=${pod_idb} --network im-grpc_yllmis-im \
-v /home/yjs/go_test/IM-Grpc/apps/social/api/etc/dev/social.yaml:/user/conf/social.yaml \
--name=${container_name} -d ${reso_addr}:${tag}