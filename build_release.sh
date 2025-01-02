#bin/bash

#----------------------------------------------
# 前后端打包编译至指定目录,即快速制作发行版
#----------------------------------------------

project_path=`pwd`
# 构建后的二进制执行文件名
exec_file_name="mayfly-go"
# web项目目录
web_folder="${project_path}/frontend"
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
    copy2Server=$1

    echo_yellow "-------------------Start bundling frontends-------------------"
    yarn run build
    if [ "${copy2Server}" == "2" ] ; then
        echo_green 'Copy the packaged static files to server/static/static'
        rm -rf ${server_folder}/static/static && mkdir -p ${server_folder}/static/static && cp -r ${web_folder}/dist/* ${server_folder}/static/static
    fi
    echo_yellow ">>>>>>>>>>>>>>>>>>>End of packaging frontend<<<<<<<<<<<<<<<<<<<<\n"
}

function build() {
    cd ${project_path}

    # 打包产物的输出目录
    toFolder=$1
    os=$2
    arch=$3
    copyDocScript=$4

    echo_yellow "-------------------Start a bundle build - ${os}-${arch}-------------------"

    cd ${server_folder}
    echo_green "Package build executables..."

    execFileName=${exec_file_name}
    # 如果是windows系统,可执行文件需要添加.exe结尾
    if [ "${os}" == "windows" ];then
        execFileName="${execFileName}.exe"
    fi
    go mod tidy
    CGO_ENABLE=0 GOOS=${os} GOARCH=${arch} go build -ldflags=-w -o ${execFileName} main.go

    if [ -d ${toFolder} ] ; then
        echo_green "The desired folder already exists. Clear the folder"
        sudo rm -rf ${toFolder}
    fi
    echo_green "Create '${toFolder}' Directory"
    mkdir ${toFolder}

    echo_green "Move binary to '${toFolder}'"
    mv ${server_folder}/${execFileName} ${toFolder}

    # if [ "${copy2Server}" == "1" ] ; then
    #     echo_green "拷贝前端静态页面至'${toFolder}/static'"
    #     mkdir -p ${toFolder}/static && cp -r ${web_folder}/dist/* ${toFolder}/static
    # fi

    if [ "${copyDocScript}" == "1" ] ; then
        echo_green "Copy resources such as scripts [config.yml.example、mayfly-go.sql、mayfly-go.sqlite、readme.txt、startup.sh、shutdown.sh]"
        cp ${server_folder}/config.yml.example ${toFolder}
        cp ${server_folder}/readme.txt ${toFolder}
        cp ${server_folder}/readme_en.txt ${toFolder}
        cp ${server_folder}/resources/script/startup.sh ${toFolder}
        cp ${server_folder}/resources/script/shutdown.sh ${toFolder}
        cp ${server_folder}/resources/script/sql/mayfly-go.sql ${toFolder}
        cp ${server_folder}/resources/data/mayfly-go.sqlite ${toFolder}
    fi

    echo_yellow ">>>>>>>>>>>>>>>>>>> ${os}-${arch} - Bundle build complete <<<<<<<<<<<<<<<<<<<<\n"
}

function buildLinuxAmd64() {
    build "$1/mayfly-go-linux-amd64" "linux" "amd64" $2
}

function buildLinuxArm64() {
    build "$1/mayfly-go-linux-arm64" "linux" "arm64" $2
}

function buildWindows() {
    build "$1/mayfly-go-windows" "windows" "amd64" $2
}

function buildMac() {
    build "$1/mayfly-go-mac" "darwin" "amd64" $2
}

function buildDocker() {
    echo_yellow "-------------------Start building the docker image-------------------"
    imageVersion=$1
    imageName="mayfly/mayfly-go:${imageVersion}"
    docker build --no-cache --platform linux/amd64 --build-arg MAYFLY_GO_VERSION="${imageVersion}" -t "${imageName}" .
    echo_green "The docker image is built -> [${imageName}]"
    echo_yellow "-------------------Finished building the docker image-------------------"
}

function buildxDocker() {
    echo_yellow "-------------------The docker buildx build image starts-------------------"
    imageVersion=$1
    imageName="ccr.ccs.tencentyun.com/mayfly/mayfly-go:${imageVersion}"
    docker buildx build --no-cache --push --platform linux/amd64,linux/arm64 --build-arg MAYFLY_GO_VERSION="${imageVersion}" -t "${imageName}" .
    echo_green "The docker multi-architecture version image is built -> [${imageName}]"
    echo_yellow "-------------------The docker buildx image is finished-------------------"
}

function runBuild() {
    read -p "Select build version [0 | Other->Other than docker image 1->linux-amd64 2->linux-arm64 3->windows 4->mac 5->docker 6->docker buildx]: " buildType

    toPath="."
    imageVersion="latest"
    copyDocScript="1"

    if [[ "${buildType}" != "5" ]] && [[ "${buildType}" != "6" ]] ; then
        # 构建结果的目的路径
        read -p "Please enter the build product output directory [default current path]: " toPath
        if [ ! -d ${toPath} ] ; then
            echo_red "Build product output directory does not exist!"
            exit;
        fi
        if [ "${toPath}" == "" ] ; then
            toPath="."
        fi

        read -p "Whether to copy documents & Scripts [0-> No 1-> Yes][Default yes]: " copyDocScript
        if [ "${copyDocScript}" == "" ] ; then
            copyDocScript="1"
        fi

        # 进入目标路径,并赋值全路径
        cd ${toPath}
        toPath=`pwd`

        # read -p "是否构建前端[0|其他->否 1->是 2->构建并拷贝至server/static/static]: " runBuildWeb
        runBuildWeb="2"
        # 编译web前端
        buildWeb ${runBuildWeb}
    fi

    if [[ "${buildType}" == "5" ]] || [[ "${buildType}" == "6" ]] ; then
        read -p "Please enter the docker image version (default latest) : " imageVersion

        if [ "${imageVersion}" == "" ] ; then
            imageVersion="latest"
        fi
    fi

    case ${buildType} in
         "1")
            buildLinuxAmd64 ${toPath} ${copyDocScript}
        ;;
         "2")
            buildLinuxArm64 ${toPath} ${copyDocScript}
        ;;
        "3")
            buildWindows ${toPath} ${copyDocScript}
        ;;
        "4")
            buildMac ${toPath} ${copyDocScript}
        ;;
        "5")
            buildDocker ${imageVersion}
        ;;
        "6")
            buildxDocker ${imageVersion}
        ;;
        *)
            buildLinuxAmd64 ${toPath} ${copyDocScript}
            buildLinuxArm64 ${toPath} ${copyDocScript}
            buildWindows ${toPath} ${copyDocScript}
            buildMac ${toPath} ${copyDocScript}
        ;;
    esac

    if [[ "${buildType}" != "5" ]] && [[ "${buildType}" != "6" ]] ; then
        echo_green "Delete static assets under ['${server_folder}/static/static']."
        # 删除静态资源文件，保留一个favicon.ico，否则后端启动会报错
        rm -rf ${server_folder}/static/static/assets
        rm -rf ${server_folder}/static/static/config.js
        rm -rf ${server_folder}/static/static/index.html
    fi
}

runBuild
