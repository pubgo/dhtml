
.PHONY: version build build_linux docker_login docker_build docker_push_dev docker_push_pro
.PHONY: rm_stop

Version = v1.1.1
VersionFile = version/version.go
GitCommit = $(shell git rev-parse --short=8 HEAD)
BuildVersion = $(shell date "+%F %T")
GOBIN = $(shell pwd)

ImagesPrefix = registry.cn-hangzhou.aliyuncs.com/ybase/
ImageName = dhtml
ImageTestName = $(ImageName):test
ImageCommitName = $(ImageName):$(GitCommit)
VersionCommitName = $(ImageName):$(Version)

t:
	@echo $(ImageTestName)
	@echo $(ImageCommitName)
	@echo $(GitCommit)
	@echo $(BuildVersion)
	@echo $(GOBIN)

version:
	@echo "项目版本处理"
	@echo "package version" > $(VersionFile)
	@echo "const Version = "\"$(Version)\" >> $(VersionFile)
	@echo "const BuildVersion = "\"$(BuildVersion)\" >> $(VersionFile)
	@echo "const GitCommit = "\"$(GitCommit)\" >> $(VersionFile)

build:
	@echo "开始编译"
	GOBIN=$(GOBIN) go install main.go

build_linux: version
	@echo "交叉编译成linux应用"
	docker run --rm -v $(GOPATH):/go golang:latest go build -o /go/src/github.com/pubgo/dhtml/main /go/src/github.com/pubgo/dhtml/main.go

rm_stop:
	@echo "删除所有的的容器"
	sudo docker rm -f $(sudo docker ps -qa)
	sudo docker ps -a

rm_none:
	@echo "删除所为none的image"
	sudo docker images  | grep none | awk '{print $3}' | xargs docker rmi

docker_push_pro: docker_build
	@echo "docker push pro"
#	sudo docker tag $(ImageName) $(ImagesPrefix)$(ImageName)
	sudo docker tag $(ImageName) $(ImagesPrefix)$(VersionCommitName)
	sudo docker push $(ImagesPrefix)$(VersionCommitName)

docker_push_dev: docker_build
	@echo "docker push test"
	sudo docker tag $(ImageName) $(ImagesPrefix)$(ImageTestName)
	sudo docker push $(ImagesPrefix)$(ImageTestName)

docker_build: build_linux
	@echo "构建docker镜像"
	sudo docker build -t $(ImageName) .

# https://github.com/Zenika/alpine-chrome
#/Applications/Google\ Chrome\ Canary.app/Contents/MacOS/Google\ Chrome\ Canary --headless —remote-debugging-port=9222
#https://github.com/chromedp/chromedp-proxy
test_run:
	docker run --rm -p 8082:8080 -p 9222:9222 -v $(pwd)/tmp:/tmp1 dhtml