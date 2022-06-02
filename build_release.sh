#bin/bash

#----------------------------------------------
# 前后端打包编译至指定目录,即快速制作发行版
#----------------------------------------------

project_path=`pwd`
# 构建后的二进制执行文件名
exec_file_name="mayfly-go"
# web项目目录
web_folder="${project_path}/mayfly_go_web"
# server目录
server_folder="${project_path}/server"

function echo_red() {
    echo -e "\033[1;31m$1\033[0m"
}

function echo_green() {
    echo -e "\033[1;32m$1\033[0m"
}

function echo_yellow() {
    echo -e "\033[1;33m$1\033[0m"
}

function buildWeb() {
    cd ${web_folder}
    echo_yellow "-------------------打包前端开始-------------------"
    yarn run build
    echo_green '将打包后的静态文件拷贝至server/static'
    rm -rf ${server_folder}/static
    mkdir -p ${server_folder}/static && cp -r ${web_folder}/dist/* ${server_folder}/static
    echo_yellow ">>>>>>>>>>>>>>>>>>>打包前端结束<<<<<<<<<<<<<<<<<<<<\n"
}

function build() {
    cd ${project_path}

    # 打包产物的输出目录
    toFolder=$1
    os=$2
    arch=$3

    echo_yellow "-------------------${os}-${arch}打包构建开始-------------------"

    cd ${server_folder}
    echo_green "打包构建可执行文件..."
    CGO_ENABLE=0 GOOS=${os} GOARCH=${arch} go build -o ${exec_file_name} main.go

    if [ -d ${toFolder} ] ; then
        echo_green "目标文件夹已存在,清空文件夹"
        sudo rm -rf ${toFolder}
    fi
    echo_green "创建'${toFolder}'目录"
    mkdir ${toFolder}

    echo_green "移动二进制文件至'${toFolder}'"
    mv ${server_folder}/${exec_file_name} ${toFolder}

    echo_green "拷贝前端静态页面至'${toFolder}/static'"
    mkdir -p ${toFolder}/static && cp -r ${web_folder}/dist/* ${toFolder}/static

    echo_green "拷贝脚本等资源文件[config.yml、mayfly-go.sql、readme.txt、startup.sh、shutdown.sh]"
    cp ${server_folder}/config.yml ${toFolder}
    cp ${server_folder}/mayfly-go.sql ${toFolder}
    cp ${server_folder}/readme.txt ${toFolder}
    cp ${server_folder}/startup.sh ${toFolder}
    cp ${server_folder}/shutdown.sh ${toFolder}

    echo_yellow ">>>>>>>>>>>>>>>>>>>${os}-${arch}打包构建完成<<<<<<<<<<<<<<<<<<<<\n"
}

function buildLinuxAmd64() {
    build "$1/mayfly-go-linux-amd64" "linux" "amd64"
}

function buildLinuxArm64() {
    build "$1/mayfly-go-linux-arm64" "linux" "arm64"
}

function buildWindows() {
    build "$1/mayfly-go-windows" "windows" "amd64"
}

function runBuild() {
    # 构建结果的目的路径
    read -p "请输入构建产物输出目录: " toPath
    if [ ! -d ${toPath} ] ; then
        echo_red "构建产物输出目录不存在!"
        exit;
    fi
    # 进入目标路径,并赋值全路径
    cd ${toPath}
    toPath=`pwd`

    read -p "是否构建前端[0|其他->否 1->是]: " runBuildWeb
    read -p "请选择构建版本[0|其他->全部 1->linux-amd64 2->linux-arm64 3->windows]: " buildType
    
    if [ "${runBuildWeb}" == "1" ];then
        buildWeb
    fi

    if [ "${buildType}" == "1" ];then
        buildLinuxAmd64 ${toPath}
        exit;
    fi

    if [ "${buildType}" == "2" ];then
        buildLinuxArm64 ${toPath}
        exit;
    fi

    if [ "${buildType}" == "3" ];then
        buildWindows ${toPath}
        exit;
    fi

    buildLinuxAmd64 ${toPath}
    buildLinuxArm64 ${toPath}
    buildWindows ${toPath}
}

runBuild