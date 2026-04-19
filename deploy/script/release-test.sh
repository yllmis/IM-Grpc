need_start_server_shell=(
    # rpc启动脚本
    user-rpc-test.sh
    social-rpc-test.sh
    im-rpc-test.sh
    im-ws-test.sh

    # api启动脚本
    user-api-test.sh
    social-api-test.sh
    im-api-test.sh

    # task启动脚本
    task-mq-test.sh
)

for i in ${need_start_server_shell[*]}; do
    chmod +x $i
    ./$i
done

docker ps


docker exec -it etcd etcdctl get --prefix ""